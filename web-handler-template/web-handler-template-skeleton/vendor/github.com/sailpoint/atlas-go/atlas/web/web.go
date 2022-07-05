// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/auth"
	"github.com/sailpoint/atlas-go/atlas/auth/access"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/health"
	"github.com/sailpoint/atlas-go/atlas/log"
	"github.com/sailpoint/atlas-go/atlas/trace"

	"go.uber.org/zap"
)

// requestDurations is the prometheus metric used to capture HTTP request durations.
var requestDurations = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "http_request_duration",
	Help:    "Duration of http requests",
	Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 15.0, 30.0, 60.0, 120.0},
}, []string{"httpMethod", "uriPath", "status"})

// requestCount is the prometheus metric used to capture number of HTTP requests handled.
var requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_request_count",
	Help: "Count of http requests",
}, []string{"httpMethod", "uriPath", "status"})

// requestIDHeader is the name of the customer HTTP request header used to propagate
// request id's across service boundaries.
const requestIDHeader = "SLPT-Request-ID"

// defaultWriteTimeout is the default amount of time to wait for a write to complete
const defaultWriteTimeout = 15 * time.Second

// defaultReadTimeout is the default amount of time to wait for a read to complete
const defaultReadTimeout = 15 * time.Second

// defaultIdleTimeout is the default amount of time to keep an idle connection alive
const defaultIdleTimeout = 60 * time.Second

// ErrorMessage is the standard API error response message type.
type ErrorMessage struct {
	Locale       string `json:"locale"`
	LocaleOrigin string `json:"localeOrigin"`
	Text         string `json:"text"`
}

// Error is the standard API error response type.
type Error struct {
	statusCode int
	DetailCode string         `json:"detailCode"`
	TrackingID string         `json:"trackingId"`
	Messages   []ErrorMessage `json:"messages"`
}

// newError constructs a new standard error with the specified default text.
func newError(ctx context.Context, statusCode int, messageText string) Error {
	message := ErrorMessage{}
	message.Locale = "en-US"
	message.LocaleOrigin = "DEFAULT"
	message.Text = messageText

	e := Error{}
	e.statusCode = statusCode
	e.DetailCode = http.StatusText(statusCode)
	e.Messages = []ErrorMessage{message}

	if tc := trace.GetTracingContext(ctx); tc != nil {
		e.TrackingID = string(tc.RequestID)
	}

	return e
}

// RunConfig is the configuration data required for running an HTTP server.
type RunConfig struct {
	Host         string
	Port         int
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	IdleTimeout  time.Duration
}

// MetricsConfig is the configuration data required for metrics processing.
type MetricsConfig struct {
	RunConfig
}

// statusCapture is a ResponseWriter implementation that captures the statusCode of a response.
type statusCapture struct {
	http.ResponseWriter
	statusCode int
}

// IsCountHeaderRequested returns true if client requests for the count header
func IsCountHeaderRequested(r *http.Request) bool {
	if c, err := strconv.ParseBool(r.URL.Query().Get(count)); err == nil {
		return c
	}
	return false
}

// WriteHeader captures the specified status code and delegates to the underlying ResponseWriter.
func (s *statusCapture) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

// NewRunConfig use the configuration source to get values for a RunConfig source.
func NewRunConfig(cfg config.Source) RunConfig {
	c := RunConfig{}
	c.Host = config.GetString(cfg, "ATLAS_HOST", "0.0.0.0")
	c.Port = config.GetInt(cfg, "ATLAS_REST_PORT", 7100)
	c.ReadTimeout = config.GetDuration(cfg, "ATLAS_HTTP_READ_TIMEOUT", defaultReadTimeout)
	c.WriteTimeout = config.GetDuration(cfg, "ATLAS_HTTP_WRITE_TIMEOUT", defaultWriteTimeout)
	c.IdleTimeout = config.GetDuration(cfg, "ATLAS_HTTP_IDLE_TIMEOUT", defaultIdleTimeout)

	return c
}

// NewMetricsConfig uses the configuration source to get values for a MetricsConfig source.
func NewMetricsConfig(cfg config.Source) MetricsConfig {
	c := MetricsConfig{}
	c.Host = config.GetString(cfg, "ATLAS_HOST", "0.0.0.0")
	c.Port = config.GetInt(cfg, "ATLAS_METRICS_PORT", 7200)
	c.ReadTimeout = config.GetDuration(cfg, "ATLAS_METRICS_READ_TIMEOUT", defaultReadTimeout)
	c.WriteTimeout = config.GetDuration(cfg, "ATLAS_METRICS_WRITE_TIMEOUT", defaultWriteTimeout)
	c.IdleTimeout = config.GetDuration(cfg, "ATLAS_METRICS_IDLE_TIMEOUT", defaultIdleTimeout)

	return c
}

// WriteJSON serializes an input value to JSON and writes it
// to the HTTP response. If an error is encountered while
// serializing the value to JSON, an InternalServerError
// is written.
func WriteJSON(ctx context.Context, w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		InternalServerError(ctx, w, err)
	} else {
		w.Header().Add("content-type", "application/json")
		w.Write(js)
	}
}

// NoContent writes the 204 status code to the writer.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// BadRequest writes a 400 error to the writer in standard JSON format.
func BadRequest(ctx context.Context, w http.ResponseWriter, err error) {
	e := newError(ctx, http.StatusBadRequest, err.Error())
	writeError(ctx, w, e)
}

// NotFound writes a 404 error to the writer in standard JSON format.
func NotFound(ctx context.Context, w http.ResponseWriter) {
	e := newError(ctx, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	writeError(ctx, w, e)
}

// NotFoundWithError writes a 404 error with error message to the writer in standard JSON format.
func NotFoundWithError(ctx context.Context, w http.ResponseWriter, err error) {
	e := newError(ctx, http.StatusNotFound, err.Error())
	writeError(ctx, w, e)
}

// Forbidden writes a 403 error to the writer in standard JSON format.
func Forbidden(ctx context.Context, w http.ResponseWriter) {
	e := newError(ctx, http.StatusForbidden, http.StatusText(http.StatusForbidden))
	writeError(ctx, w, e)
}

// Unauthorized writes a 401 error to the writer in standard JSON format.
func Unauthorized(ctx context.Context, w http.ResponseWriter) {
	e := newError(ctx, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	writeError(ctx, w, e)
}

// InternalServerError writes a 500 error to the writer in standard JSON format.
func InternalServerError(ctx context.Context, w http.ResponseWriter, err error) {
	e := newError(ctx, http.StatusInternalServerError, err.Error())
	writeError(ctx, w, e)
}

// writeError writes an error of the specified status to the writer in standard JSON format.
func writeError(ctx context.Context, w http.ResponseWriter, e Error) {
	errorJSON, err := json.Marshal(e)
	if err != nil {
		log.Errorf(ctx, "HTTP error: %v", err)
		InternalServerError(ctx, w, err)
	} else {
		log.Errorf(ctx, "HTTP error: %s", string(errorJSON))
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(e.statusCode)
		w.Write(errorJSON)
	}
}

// HealthCheck returns an HTTP middleware function that returns
// a standard atlas-compatible heatlh check.
func HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		result := health.CheckAll(ctx)

		if result.Status == health.StatusError {
			w.Header().Add("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
		}

		WriteJSON(ctx, w, result)
	}
}

// ResponseLogger returns an HTTP middleware function that logs
// the response stauts and times of all requests.
func ResponseLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip the built-in health-check...
			if r.URL.Path == "/health/system" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			sc := &statusCapture{w, 200}
			next.ServeHTTP(sc, r)
			dt := time.Since(start)

			log := log.Get(r.Context()).With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", sc.statusCode),
				zap.Int64("elapsed", dt.Milliseconds()),
			)

			if sc.statusCode >= 400 {
				log.Sugar().Errorf("request (%s)", dt)
			} else {
				log.Sugar().Infof("request (%s)", dt)
			}
		})
	}
}

// NewRouter returns a new router, with the default atlas routes configured for:
// - response logging
// - authentication
// - count/latency metrics
// - standard health check
func NewRouter(authenticationConfig AuthenticationConfig) *mux.Router {
	r := mux.NewRouter()
	r.Use(Recover())
	r.Use(Trace())
	r.Use(Authenticate(authenticationConfig))
	r.Use(ResponseLogger())
	r.Use(HTTPMetrics())

	r.HandleFunc("/health/system", HealthCheck()).Methods("GET")

	return r
}

// StartMetricsServer runs the embedded prometheus HTTP server
func StartMetricsServer(ctx context.Context, config MetricsConfig) error {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())

	return RunServer(ctx, config.RunConfig, r)
}

// containsAny gets whether or not any of the specified rights are contained
// within the specified access summary.
func containsAny(summary *access.Summary, rights []string) bool {
	for _, r := range rights {
		if summary.ContainsRight(access.Right(r)) {
			return true
		}
	}

	return false
}

// RequireRights returns an HTTP middleware function that ensures the token associated
// with the current request hs at last one of the specified flattened rights.
// If none of the rights are associated with a token, the request is halted with a 403 Forbidden
// response. Failure to summarize the token will result in an InternalServerError and halting
// of the current request.
func RequireRights(summarizer access.Summarizer, rights ...string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if len(rights) > 0 {
				token := auth.GetToken(r.Context())
				if token == nil {
					Forbidden(ctx, w)
					return
				}

				summary, err := summarizer.Summarize(r.Context(), token)
				if err != nil {
					InternalServerError(ctx, w, err)
					return
				}

				if !containsAny(summary, rights) {
					Forbidden(ctx, w)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Trace returns an HTTP middleware function that sets up a tracing context for
// logging and request ID propagation.
func Trace() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc := trace.NewTracingContext(trace.RequestID(r.Header.Get(requestIDHeader)))

			ctx := trace.WithTracingContext(r.Context(), tc)
			ctx = log.WithFields(ctx,
				zap.String("request_id", string(tc.RequestID)),
				zap.String("span_id", string(tc.SpanID)),
			)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// TokenExtractor is an interface for retrieving a token from an HTTP request.
type TokenExtractor interface {
	ExtractToken(r *http.Request) string
}

// TokenExtractorFunc is a type for functions that adhere to the TokenExtractor interface.
type TokenExtractorFunc func(*http.Request) string

// ExtractToken maps the function to the TokenExtractor interface.
func (f TokenExtractorFunc) ExtractToken(r *http.Request) string {
	return f(r)
}

// AuthenticationConfig contains the various options for how the Authenticate middleware works
type AuthenticationConfig struct {
	TokenValidator auth.TokenValidator
	TokenExtractor TokenExtractor
	IgnoredPaths   []*regexp.Regexp
}

// IgnorePath adds a new path to the ignore-list. The path is a regular expression.
func (cfg *AuthenticationConfig) IgnorePath(path string) {
	cfg.IgnoredPaths = append(cfg.IgnoredPaths, regexp.MustCompile(path))
}

// IsPathIgnored gets whether or not the specified path is ignored by this configuration.
func (cfg *AuthenticationConfig) IsPathIgnored(path string) bool {
	for _, ip := range cfg.IgnoredPaths {
		if ip.MatchString(path) {
			return true
		}
	}

	return false
}

// DefaultAuthenticationConfig constructs an AuthenticationConfig with the
// default options.
func DefaultAuthenticationConfig(v auth.TokenValidator) AuthenticationConfig {
	cfg := AuthenticationConfig{}
	cfg.TokenValidator = v
	cfg.TokenExtractor = TokenExtractorFunc(GetBearerToken)
	cfg.IgnorePath("/health/system")

	return cfg
}

// Authenticate returns an HTTP middleware function that authenticates requests
// using the specified configuration. Requests that are missing a token, or receive
// an invalidate token are rejected with a 401 response.
func Authenticate(cfg AuthenticationConfig) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if cfg.IsPathIgnored(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			rawToken := cfg.TokenExtractor.ExtractToken(r)
			if rawToken == "" {
				Unauthorized(ctx, w)
				return
			}

			token, err := cfg.TokenValidator.Parse(rawToken)
			if err != nil {
				Unauthorized(ctx, w)
				return
			}

			ctx = auth.WithToken(ctx, token)

			rc := token.CreateRequestContext()
			ctx = atlas.WithRequestContext(ctx, rc)

			fields := []zap.Field{
				zap.String("pod", string(rc.Pod)),
				zap.String("org", string(rc.Org)),
			}

			if rc.IdentityName != "" {
				fields = append(fields, zap.String("identity_name", string(rc.IdentityName)))
			}

			ctx = log.WithFields(ctx, fields...)

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// HTTPMetrics returns an HTTP middleware function that captures request count and latency metrics
// and sends them the default prometheus registry.
func HTTPMetrics() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc := &statusCapture{w, 200}

			start := time.Now()
			next.ServeHTTP(sc, r)
			dt := time.Since(start)

			route := mux.CurrentRoute(r)
			if route == nil {
				return
			}

			path, err := route.GetPathTemplate()
			if err != nil {
				return
			}

			labels := prometheus.Labels{
				"httpMethod": r.Method,
				"uriPath":    path,
				"status":     strconv.Itoa(sc.statusCode),
			}

			requestDurations.With(labels).Observe(float64(dt.Seconds()))
			requestCount.With(labels).Inc()
		})
	}
}

// Recover captures a panic and logs it using our logger to ensure it is
// anotated with the correct context (stack, pod, etc)
func Recover() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			defer func() {
				if err := recover(); err != nil && err != http.ErrAbortHandler {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Errorf(ctx, "HTTP panic: %s", string(buf))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RunServer starts a new HTTP server with the specified handler. It will run until the server completes, gracefully
// handling interrupts from the OS.
func RunServer(ctx context.Context, config RunConfig, handler http.Handler) error {
	if config.ReadTimeout == 0 {
		config.ReadTimeout = defaultReadTimeout
	}

	if config.WriteTimeout == 0 {
		config.WriteTimeout = defaultWriteTimeout
	}

	if config.IdleTimeout == 0 {
		config.IdleTimeout = defaultIdleTimeout
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		WriteTimeout: config.WriteTimeout,
		ReadTimeout:  config.ReadTimeout,
		IdleTimeout:  config.IdleTimeout,
		Handler:      handler,
	}

	return runWithGracefulShutdown(ctx, srv)
}

// runWithGracefulShutdown runs the specified HTTP server until the passed in context is closed
// or an error occurs.
func runWithGracefulShutdown(ctx context.Context, server *http.Server) error {
	log.Infof(ctx, "starting HTTP server on %s", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf(ctx, "error running HTTP server: %v", err)
		}
	}()

	defer server.Close()
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	log.Info(ctx, "shutting down HTTP server")
	return server.Shutdown(ctx)
}

// GetBearerToken extracts the bearer token from the HTTP request's authorization header.
// If no authorization header exists, or the request type is no "Bearer", an empty
// string is returned.
func GetBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("authorization")
	if authHeader == "" {
		return ""
	}

	authComponents := strings.Split(authHeader, " ")
	if len(authComponents) != 2 {
		return ""
	}

	if !strings.EqualFold("bearer", strings.TrimSpace(authComponents[0])) {
		return ""
	}

	return strings.TrimSpace(authComponents[1])
}
