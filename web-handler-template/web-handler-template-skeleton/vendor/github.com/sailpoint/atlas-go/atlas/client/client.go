// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package client

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sailpoint/atlas-go/atlas/trace"
)

const (
	target     = "target"
	statusCode = "statusCode"
)

var requestDurationHistogram = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "client_request_duration",
		Help:    "Duration of internal client http requests",
		Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
	})

var requestCounterVec = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "client_request_total",
		Help: "Count of internal client http requests",
	},
	[]string{target, statusCode},
)

// requestIDHeader is the name of the custom HTTP request header used to propagate
// request id's across service boundaries.
const requestIDHeader = "SLPT-Request-ID"

// originHeader is the name of the custom HTTP request header used to identify the name
// of the service making the request.
const originHeader = "SLPT-Origin"

// authorizationHeader is the name of the standard HTTP authorization header.
const authorizationHeader = "Authorization"

// Config is captures the required information to construct a new Client.
type Config struct {
	// Stack is the name of the service making the call. (Optional) (eg. "sp-scheduler")
	Stack string

	// TokenURL is the URL used to get a token. (eg. "https://acme.api.sailpoint.com/oauth/token")
	TokenURL string

	// ClientID is the oauth client ID.
	ClientID string

	// ClientSecret is the oauth client secret.
	ClientSecret string
}

// TokenSource is an interface for types that retrieve a token from a given context.
type TokenSource interface {
	GetToken(ctx context.Context) (*Token, error)
}

// Token is a struct that wraps an encoded token with an expiration time.
type Token struct {
	EncodedToken string
	Expiration   time.Time
}

type DefaultTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	client       *http.Client
}

type clientTransport struct {
	tokenSource TokenSource
	mu          sync.RWMutex
	token       *Token
	stack       string
}

// New constructs an HTTP client that uses OAuth 2.0 from Oathkeeper for authentication
func New(config Config) *http.Client {
	client := &http.Client{}

	ts := NewTokenSource(http.DefaultClient, config.TokenURL, config.ClientID, config.ClientSecret)
	client.Transport = newClientTransport(config.Stack, ts)

	return client
}

// IsNearlyExpired gets whether or not the token is expired (or close to expiration).
func (t *Token) IsNearlyExpired() bool {
	now := time.Now().UTC().Add(2 * time.Minute)
	return now.After(t.Expiration)
}

// newClientTransport constructs a new client transport using the specified token source.
func newClientTransport(stack string, tokenSource TokenSource) *clientTransport {
	ct := &clientTransport{}
	ct.stack = stack
	ct.tokenSource = tokenSource

	return ct
}

// isTokenValid gets whether the token associated with the transport exists and is not nearly expired.
func (ct *clientTransport) isTokenValid() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.token != nil && !ct.token.IsNearlyExpired()
}

// updateToken gets a new token from the token source if the current
// token is not valid.
func (ct *clientTransport) updateToken(ctx context.Context) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.token == nil || ct.token.IsNearlyExpired() {
		token, err := ct.tokenSource.GetToken(ctx)
		if err != nil {
			return err
		}

		ct.token = token
	}

	return nil
}

// ensureToken makes sure that the current token is valid, loading a new one
// from the TokenSource if the current token is expired.
func (ct *clientTransport) ensureToken(ctx context.Context) error {
	if ct.isTokenValid() {
		return nil
	}

	return ct.updateToken(ctx)
}

func GetTarget(ctx context.Context) string {
	v := ctx.Value(contextKeyTarget)

	if v == nil {
		return ""
	}

	return v.(string)
}

// RoundTrip forwards an HTTP request to the default transport, adding
// the authorization header, and SLPT-Origin (if stack is non-empty).
func (ct *clientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := ct.ensureToken(req.Context()); err != nil {
		return nil, err
	}

	ctx := req.Context()
	targetStr := GetTarget(ctx)

	req = cloneRequest(req)
	req.Header.Add(authorizationHeader, "Bearer "+ct.token.EncodedToken)

	if tc := trace.GetTracingContext(req.Context()); tc != nil {
		req.Header.Add(requestIDHeader, string(tc.RequestID))
	}

	if ct.stack != "" {
		req.Header.Add(originHeader, ct.stack)
	}

	start := time.Now()

	resp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		return nil, err
	}

	requestDurationHistogram.Observe(time.Since(start).Seconds())

	counterLabels := prometheus.Labels{}
	counterLabels[target] = targetStr
	counterLabels[statusCode] = strconv.Itoa(resp.StatusCode)

	requestCounterVec.With(counterLabels).Inc()

	return resp, err
}

// cloneRequest makes a copy of a request object since the contract for the client
// Transport requires that the input request is not modified.
func cloneRequest(req *http.Request) *http.Request {
	clone := &http.Request{}
	*clone = *req

	clone.Header = make(http.Header, len(req.Header))
	for k, s := range req.Header {
		clone.Header[k] = append([]string(nil), s...)
	}

	return clone
}

// NewTokenSource constructs a new source with the specified credentials.
func NewTokenSource(client *http.Client, tokenURL string, clientID string, clientSecret string) *DefaultTokenSource {
	ts := &DefaultTokenSource{}
	ts.tokenURL = tokenURL
	ts.clientID = clientID
	ts.clientSecret = clientSecret
	ts.client = client

	return ts
}

// GetToken gets a new token from the token URL.
func (s *DefaultTokenSource) GetToken(ctx context.Context) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.tokenURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(s.clientID, s.clientSecret)

	q := req.URL.Query()
	q.Set("grant_type", "client_credentials")
	req.URL.RawQuery = q.Encode()

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status %d", res.StatusCode)
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	decoder := json.NewDecoder(res.Body)

	tr := &tokenResponse{}
	if err = decoder.Decode(tr); err != nil {
		return nil, err
	}

	token := &Token{}
	token.EncodedToken = tr.AccessToken
	token.Expiration = time.Now().UTC().Add(time.Duration(tr.ExpiresIn) * time.Second)

	return token, nil
}

type PasswordTokenSource struct {
	tokenURL string
	username string
	password string
	isProd   bool
	client   *http.Client
}

// NewPasswordTokenSource constructs a new source with the specified credentials.
func NewPasswordTokenSource(client *http.Client, tokenURL string, username string, password string, isProd bool) *PasswordTokenSource {
	ts := &PasswordTokenSource{}
	ts.tokenURL = tokenURL
	ts.username = username
	ts.password = password
	ts.isProd = isProd
	ts.client = client
	return ts
}

// GetToken gets a new token from the token URL.
func (s *PasswordTokenSource) GetToken(ctx context.Context) (*Token, error) {
	u, err := url.Parse(s.tokenURL)
	if err != nil {
		return nil, err
	}
	orgName := strings.Split(u.Host, ".")[0]

	orgData, err := retrieveOrgData(orgName, s.isProd)
	if err != nil {
		return nil, err
	}
	slug := orgData.CCAPIUser + ":" + orgData.CCAPIKey
	basic := base64.StdEncoding.EncodeToString([]byte(slug))

	// Override password for production envs
	if len(orgData.Password) > 0 {
		s.password = orgData.Password
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.tokenURL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("grant_type", "password")
	q.Set("username", s.username)
	q.Set("password", usernamePasswordHash(s.username, s.password))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", "Basic "+basic)

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status %d", res.StatusCode)
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	decoder := json.NewDecoder(res.Body)

	tr := &tokenResponse{}
	if err = decoder.Decode(tr); err != nil {
		return nil, err
	}

	token := &Token{}
	token.EncodedToken = tr.AccessToken
	token.Expiration = time.Now().UTC().Add(time.Duration(tr.ExpiresIn) * time.Second)

	return token, nil
}

func usernamePasswordHash(username string, password string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password+fmt.Sprintf("%x", sha256.Sum256([]byte(username))))))
}

func retrieveOrgData(org string, isProd bool) (*orgData, error) {
	repo := newOrgRepo()
	if isProd {
		return repo.retrieveOrgDataProd(org)
	} else {
		return repo.retrieveOrgDataDev(org)
	}
}
