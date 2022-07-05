// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package main

import (
	log "github.com/sailpoint/atlas-go/atlas/log"
	"github.com/sailpoint/sp-connect/internal/sp/connect/infra"
)

func main() {
	service, err := infra.NewConnectService()
	if err != nil {
		log.Global().Sugar().Fatalf("init: %v", err)
	}
	defer service.Close()

	if err := service.Run(); err != nil {
		log.Global().Sugar().Fatalf("run: %v", err)
	}
}
