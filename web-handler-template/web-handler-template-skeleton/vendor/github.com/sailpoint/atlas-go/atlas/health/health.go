// Copyright (c) 2022. Sailpoint Technologies, Inc. All rights reserved.
package health

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/sailpoint/atlas-go/atlas"
)

// Status is an enumeration for the result of a health check.
type Status string

const (
	// StatusOK means the check was successfull
	StatusOK Status = "OK"

	// StatusWarn means the service isn't unhealthy and shouldn't be
	// terminated, but something is odd that warrants attention or could
	// be indicative of a problem that may self-correct.
	StatusWarn = "WARN"

	// Error means the health check failed and the system is unhealthy. Repeated
	// health checks that result in StatusError may result in the container
	// being killed.
	StatusError = "ERROR"
)

// AggregateCheckResult is a combination of the result of multiple health checks.
// The status of the aggregate result is the "lowest" of the contained checks.
type AggregateCheckResult struct {
	Timestamp atlas.Time              `json:"timestamp"`
	Status    Status                  `json:"status"`
	Checks    map[string]*CheckResult `json:"checks"`
}

// CheckResult is the result of an individual health check.
type CheckResult struct {
	Timestamp atlas.Time             `json:"timestamp"`
	Status    Status                 `json:"status"`
	Details   map[string]interface{} `json:"details"`
}

// Check is an interface for an entity that can check the health of some
// part of the system.
type Check interface {
	// CheckHealth returns the result of the health check, or an error.
	CheckHealth(ctx context.Context) (*CheckResult, error)
}

// CheckFunc is an adapter to map a health check function to the interface.
type CheckFunc func(context.Context) (*CheckResult, error)

func (cf CheckFunc) CheckHealth(ctx context.Context) (*CheckResult, error) {
	return cf(ctx)
}

// cachedCheck is a Check implementation that caches
// the result of an upstream health check for a specified
// duration.
type cachedCheck struct {
	check    Check
	duration time.Duration

	mu         sync.RWMutex
	lastResult *CheckResult
	lastErr    error
	expiration time.Time
}

// newCachedCheck constructs a new cachedCheck with the specified
// delegate check and cache duration.
func newCachedCheck(check Check, duration time.Duration) *cachedCheck {
	c := &cachedCheck{}
	c.check = check
	c.duration = duration

	return c
}

// CheckHealth will return the last result of the healt check if it has
// been cached for less than the cached duration, otherwise the delegate
// Check will be evaluated and result cached.
func (c *cachedCheck) CheckHealth(ctx context.Context) (*CheckResult, error) {
	if c.isValid() {
		return c.lastResult, c.lastErr
	}

	return c.updateCache(ctx)
}

// isExpired gets whether or not the cached value is valid (not expired).
func (c *cachedCheck) isValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Now().UTC().Before(c.expiration)
}

// updateCache will invoke the delegate Check and persist the results in the cache.
func (c *cachedCheck) updateCache(ctx context.Context) (*CheckResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check again, in case another thread updated the result
	if time.Now().UTC().Before(c.expiration) {
		return c.lastResult, c.lastErr
	}

	result, err := c.check.CheckHealth(ctx)

	c.lastResult = result
	c.lastErr = err
	c.expiration = time.Now().UTC().Add(c.duration)

	return result, err
}

// global data for the check registry
var registeredChecksMu sync.RWMutex
var registeredChecks map[string]Check

// init sets up the check registry and registers default checks
func init() {
	registeredChecks = make(map[string]Check)

	RegisterCheck("go-runtime", CheckFunc(RuntimeCheck))
}

// NewAggregateCheckResult will construct a new, empty AggregateCheckResult.
func NewAggregateCheckResult() *AggregateCheckResult {
	r := &AggregateCheckResult{}
	r.Timestamp = atlas.Time(time.Now().UTC())
	r.Status = StatusOK
	r.Checks = make(map[string]*CheckResult)

	return r
}

// CheckAll will evaluate the health of all registered checks,
// returning the aggregate result.
func CheckAll(ctx context.Context) *AggregateCheckResult {
	registeredChecksMu.RLock()
	defer registeredChecksMu.RUnlock()

	result := NewAggregateCheckResult()

	for name, check := range registeredChecks {
		result.AddCheck(ctx, name, check)
	}

	return result
}

// RegisterCheck adds a new check to the global check registry.
func RegisterCheck(name string, check Check) {
	registeredChecksMu.Lock()
	defer registeredChecksMu.Unlock()

	registeredChecks[name] = newCachedCheck(check, 5*time.Second)
}

// NewCheckResult constructs a new CheckResult with the specified status.
func NewCheckResult(status Status) *CheckResult {
	r := &CheckResult{}
	r.Timestamp = atlas.Time(time.Now().UTC())
	r.Status = status
	r.Details = make(map[string]interface{})

	return r
}

// CheckResultOK constructs a new CheckResult with the OK status.
func CheckResultOK() *CheckResult {
	return NewCheckResult(StatusOK)
}

// CheckResultWarn constructs a new CheckResult with the Warn status.
func CheckResultWarn() *CheckResult {
	return NewCheckResult(StatusWarn)
}

// CheckResultError constructs a new CheckResult with the Error status.
func CheckResultError() *CheckResult {
	return NewCheckResult(StatusError)
}

// AddCheck will run the specified health check and incoroprate the result.
func (cr *AggregateCheckResult) AddCheck(ctx context.Context, name string, check Check) {
	result, err := check.CheckHealth(ctx)
	if err != nil {
		result = CheckResultError().
			AddError(err)
	}

	cr.addCheckResult(name, result)
}

// computeStatus compares the current status and the status of an incoming
// check and returns the resulting aggregate status.
func computeStatus(current Status, checkStatus Status) Status {
	if current == StatusError || checkStatus == StatusError {
		return StatusError
	}

	if current == StatusWarn || checkStatus == StatusWarn {
		return StatusWarn
	}

	return StatusOK
}

// addCheckResult incorporates the specified check result into the aggregate, updating
// the overall status if necessary.
func (cr *AggregateCheckResult) addCheckResult(name string, result *CheckResult) {
	cr.Checks[name] = result
	cr.Status = computeStatus(cr.Status, result.Status)
}

// AddError appends a detail value containing the error text to the CheckResult.
func (cr *CheckResult) AddError(err error) *CheckResult {
	cr.Add("error", err.Error())

	return cr
}

// Add appends a detail value to the CheckResult.
func (cr *CheckResult) Add(key string, value interface{}) *CheckResult {
	cr.Details[key] = value

	return cr
}

// RuntimeCheck is a CheckFunc that returns detailed information
// about the go runtime.
func RuntimeCheck(ctx context.Context) (*CheckResult, error) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	check := CheckResultOK().
		Add("alloc", stats.Alloc).
		Add("totalAlloc", stats.TotalAlloc).
		Add("sys", stats.Sys).
		Add("gcCount", stats.NumGC).
		Add("cpuCount", runtime.NumCPU()).
		Add("goroutineCount", runtime.NumGoroutine())

	return check, nil
}
