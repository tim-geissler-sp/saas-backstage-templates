// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package message

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sailpoint/atlas-go/atlas/compress"
)

const publishScript = `
redis.call('LPUSH', KEYS[2], ARGV[1])
if redis.call('SADD', KEYS[1], KEYS[2]) == 1 then
	redis.call('PUBLISH', 'activeQueuesChannel', '+' .. KEYS[2])
end
`

type redisPublisher struct {
	client redis.Cmdable
}

func NewRedisPublisher(client redis.Cmdable) Publisher {
	p := &redisPublisher{}
	p.client = client

	return p
}

func (p *redisPublisher) PublishAtomicFromContext(ctx context.Context, sd ScopeDescriptor, message *Message, options PublishOptions) error {
	scope, err := NewScopeFromContext(ctx, sd)
	if err != nil {
		return err
	}

	return p.PublishAtomic(ctx, scope, message, options)
}

func (p *redisPublisher) PublishAtomic(ctx context.Context, scope Scope, message *Message, options PublishOptions) error {
	compressedMessage, err := buildCompressedMessage(message)
	if err != nil {
		return err
	}

	if options.Delay > 0 {
		key := getProcessingKey(scope, options.Priority)

		z := redis.Z{
			Score:  getFutureTimestamp(options.Delay),
			Member: compressedMessage,
		}

		if _, err := p.client.ZAdd(ctx, key, &z).Result(); err != nil {
			return fmt.Errorf("publish delayed message: %w", err)
		}

		return nil
	}

	keys := []string{"activeQueues", getKey(scope, options.Priority)}

	if _, err := p.client.Eval(ctx, publishScript, keys, compressedMessage).Result(); err != nil && err != redis.Nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

func getFutureTimestamp(duration time.Duration) float64 {
	return float64(time.Now().UTC().Add(duration).UnixNano() / 1000000)
}

func getKey(scope Scope, priority Priority) string {
	return strings.Join([]string{string(scope.ID()), "queues", string(priority)}, "/")
}

func getProcessingKey(scope Scope, priority Priority) string {
	return getKey(scope, priority) + "/processing"
}

func buildCompressedMessage(message *Message) (string, error) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	// Prefix with a random UUID so it can be added to a set without conflict.
	prefix := strings.ReplaceAll(uuid.New().String(), "-", "")

	messageData := fmt.Sprintf("%s#%s", prefix, string(messageJSON))
	return compress.Compress64(messageData)
}
