// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package access

import (
	"context"
	"encoding/hex"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/sailpoint/atlas-go/atlas/auth"
)

// cachedSummarizer is a Summarizer implementation that caches results from a delegate.
type cachedSummarizer struct {
	delegate Summarizer
	mu       sync.RWMutex
	cache    map[string]*cachedSummary
}

// cachedSummary is a cached item that ties a summary to an expiration timestamp.
type cachedSummary struct {
	summary    *Summary
	expiration time.Time
}

// isValid determines whether or not the specified summary is valid. A valid cached summary is
// non-nil and not expired.
func (cs *cachedSummary) isValid() bool {
	if cs == nil {
		return false
	}

	return time.Now().Before(cs.expiration)
}

// newCachedSummarizer constructs a new cachedSummarizer using the specified
// to delegate to load cache values.
func newCachedSummarizer(delegate Summarizer) *cachedSummarizer {
	s := &cachedSummarizer{}
	s.delegate = delegate
	s.cache = make(map[string]*cachedSummary)

	return s
}

// Summarize generates a summary for the specified token. If a valid summary is cached, it is returned. Otherwise
// the delegate Summarizer is invoked. The resulting Summary is then cached for later use.
func (s *cachedSummarizer) Summarize(ctx context.Context, t *auth.Token) (*Summary, error) {
	key := cacheKey(t)

	if summary := s.getCachedSummary(key); summary != nil {
		return summary, nil
	}

	return s.updateSummary(ctx, key, t)
}

// getCachedSummary retrieves a value from the cache. If the cached value is invalid, nil is returned.
func (s *cachedSummarizer) getCachedSummary(key string) *Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.cache[key]
	if !cs.isValid() {
		return nil
	}

	return cs.summary
}

// updateSummary retrieves a summary from the delegate Summarizer if the one in the cache is non-existant or expired.
// The summary is cached locally for later use.
func (s *cachedSummarizer) updateSummary(ctx context.Context, key string, t *auth.Token) (*Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.cache[key]
	if cs.isValid() {
		return cs.summary, nil
	}

	summary, err := s.delegate.Summarize(ctx, t)
	if err != nil {
		return nil, err
	}

	cs = &cachedSummary{}
	cs.summary = summary
	cs.expiration = time.Now().Add(5 * time.Minute)

	s.cache[key] = cs
	return summary, nil
}

// cacheKey computes the key to use for the specified token.
// Note: this is different from the java implementation since it requires the
// authorities are sorted and generates a hash
func cacheKey(t *auth.Token) string {
	h := fnv.New128()
	h.Write([]byte(t.TenantID))

	// Make sure the authorities are in sorted order so that the
	// hash function remains stable.
	sortedAuthorities := make([]string, 0, len(t.Authorities))
	for _, a := range t.Authorities {
		sortedAuthorities = append(sortedAuthorities, string(a))
	}
	sort.Strings(sortedAuthorities)

	for _, a := range sortedAuthorities {
		h.Write([]byte(a))
	}

	return hex.EncodeToString(h.Sum(nil))
}
