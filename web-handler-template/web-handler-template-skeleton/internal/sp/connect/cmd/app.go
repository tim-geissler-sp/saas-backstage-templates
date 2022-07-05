// Copyright (c) 2020, SailPoint Technologies, Inc. All rights reserved.
package cmd

import (
	"context"

	"github.com/sailpoint/sp-connect/internal/sp/connect/model"
)

// App is an interface that represents all of the functionality enabled by the application.
type App interface {
	HelloWorld(ctx context.Context, cmd HelloWorld) error //Hello World
}

type DefaultApp struct {
	GenericModel model.GenericModel
}

// GenericService function for a corresponding service
// Pass through relevant models
func (a *DefaultApp) GenericService(ctx context.Context, cmd HelloWorld) (string, error) {
	return cmd.Handle(ctx)
}
