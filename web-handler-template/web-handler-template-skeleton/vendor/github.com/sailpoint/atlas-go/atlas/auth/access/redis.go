// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package access

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sailpoint/atlas-go/atlas/auth"
	"github.com/sailpoint/atlas-go/atlas/log"
)

// redisSummarizer is a Summarizer implementation that uses redis to store a JSON-encoding of
// a Summary. If the value is not in redis, it is read from a delegate Summarizer and the result is cached
// for later retrieval.
type redisSummarizer struct {
	client   redis.Cmdable
	delegate Summarizer
}

// newRedisSummarizer constructs a new redisSummarizer instance.
func newRedisSummarizer(client redis.Cmdable, delegate Summarizer) *redisSummarizer {
	s := &redisSummarizer{}
	s.client = client
	s.delegate = delegate

	return s
}

// Summarize builds a Summary from the specified token. The value is read from redis, if no entry exists
// in redis, then the delegate summarize is invoked. The result of the summarizer is cached for
// later use. If any redis errors are encountered, the delegate summary is returned and errors are logged.
func (s *redisSummarizer) Summarize(ctx context.Context, t *auth.Token) (*Summary, error) {
	key := fmt.Sprintf("ams:cache:%s", cacheKey(t))

	value, err := s.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		log.Errorf(ctx, "error getting access summary from redis: %v", err)
		return s.cacheDelegateSummary(ctx, key, t)
	}

	if value != "" {
		var summary Summary
		if err := json.Unmarshal([]byte(value), &summary); err != nil {
			log.Errorf(ctx, "error decoding access summary rom redis: %v", err)
			return s.cacheDelegateSummary(ctx, key, t)
		}

		return &summary, nil
	}

	return s.cacheDelegateSummary(ctx, key, t)
}

// cacheDelegateSummary gets a summary from the delegate Summarizer. The resulting summary is written to redis
// using the specified key.
func (s *redisSummarizer) cacheDelegateSummary(ctx context.Context, key string, t *auth.Token) (*Summary, error) {
	summary, err := s.delegate.Summarize(ctx, t)
	if err != nil {
		return nil, err
	}

	if summaryJSON, err := json.Marshal(summary); err == nil {
		if s.client.Set(ctx, key, string(summaryJSON), 5*time.Minute).Err() != nil {
			log.Errorf(ctx, "error writing access summary to redis: %v", err)
		}
	}

	return summary, nil
}
