// Copyright (c) 2022. Sailpoint Technologies, Inc. All rights reserved.
package atlas

import (
	"context"
	"runtime"

	"github.com/sailpoint/atlas-go/atlas/log"
	"golang.org/x/sync/errgroup"
)

// AtlasRoutines is a wrapper around go's errgroup.
type AtlasRoutines struct {
	group *errgroup.Group
}

// NewRoutineWithContext creates an AtlasRoutines with context.
func NewRoutineWithContext(ctx context.Context) (*AtlasRoutines, context.Context) {
	g, ctx := errgroup.WithContext(ctx)
	return &AtlasRoutines{
		group: g,
	}, ctx
}

// Go starts a go routine with panic recovery. The recovery function logs the error and then panic using the original error.
func (ar *AtlasRoutines) Go(ctx context.Context, f func() error) {
	ar.group.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				log.Errorf(ctx, "atlas routine panic: %s ", string(buf))

				// At this point the routine is effectively dead. There is no need to keep the service running.
				// Panic and kill the service for ECS to restart it.
				panic(err)
			}
		}()
		return f()
	})
}

// Wait waits for all routines to run.
func (ar *AtlasRoutines) Wait() error {
	err := ar.group.Wait()
	return err
}
