// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package access

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/sailpoint/atlas-go/atlas/auth"
	"github.com/sailpoint/atlas-go/atlas/client"
)

// Right is a type that represents access to a granular unit of functionality (eg. idn:source:create)
type Right string

// RightSetID is the unique identity of a named set of rights (eg. idn:source-management)
type RightSetID string

// Summary is a type that contains all of the granular access held by a token.
type Summary struct {

	// RightSets holds the set of all matched RightSets in the token.
	RightSets []RightSetID `json:"rightSets"`

	// FlattenedRights holds all of the granular rights granted by the token.
	FlattenedRights []Right `json:"flattenedRights"`
}

// Summarizer is an interface that supports summarization of a token's access.
type Summarizer interface {

	// Summarize builds an AccessSummary from a token.
	Summarize(ctx context.Context, t *auth.Token) (*Summary, error)
}

// NewSummarizer constructs a new default summarizer chain that works as follows:
// request -> cache -> redis -> AMS
func NewSummarizer(client redis.Cmdable, baseURLProvider client.BaseURLProvider, internalClientProvider client.InternalClientProvider) Summarizer {
	return newCachedSummarizer(newRedisSummarizer(client, newAmsSummarizer(baseURLProvider, internalClientProvider)))
}

// ContainsRight gets whether or not the specified access summary contains the specified Right.
func (s *Summary) ContainsRight(right Right) bool {
	for _, r := range s.FlattenedRights {
		if r == right {
			return true
		}
	}

	return false
}
