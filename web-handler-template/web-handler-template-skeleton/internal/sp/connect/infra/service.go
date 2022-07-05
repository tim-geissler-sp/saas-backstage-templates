// Copyright (c) 2020, SailPoint Technologies, Inc. All rights reserved.
package infra

import (
	"context"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/application"
	"github.com/sailpoint/sp-connect/internal/sp/connect/cmd"
)

// ConnectService is the main application structure.
type ConnectService struct {
	*application.Application
	app cmd.App
}

// NewConnectService constructs a new service instance.
func NewConnectService() (*ConnectService, error) {
	application, err := application.New("sp-connect")
	if err != nil {
		return nil, err
	}

	s := &ConnectService{}
	s.Application = application

	return s, nil
}

// Run will start the server execution, waiting for the system to exit.
func (s *ConnectService) Run() error {
	ctx, done := context.WithCancel(context.Background())

	ar, ctx := atlas.NewRoutineWithContext(ctx)
	ar.Go(ctx, func() error { return s.StartBeaconHeartbeat(ctx) })
	//ar.Go(ctx, func() error { return s.StartEventConsumer(ctx, s.bindEventHandlers()) })
	ar.Go(ctx, func() error { return s.StartMetricsServer(ctx) })
	ar.Go(ctx, func() error { return s.StartWebServer(ctx, s.buildRoutes()) })
	ar.Go(ctx, func() error { return s.WaitForInterrupt(ctx, done) })

	if err := ar.Wait(); err != nil && err != context.Canceled {
		return err
	}

	return nil
}
