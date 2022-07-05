// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gavv/httpexpect/v2"
)

// readSpecFile reads connector spec JSON file from _data folder
func readSpecFile(fileName string) []byte {
	raw, err := os.ReadFile("_data/" + fileName)
	if err != nil {
		log.Fatal(err)
	}
	return raw
}

// deleteConnectorInstance deletes connector instance by ID
func deleteConnectorInstance(e *httpexpect.Expect, id string) {
	e.DELETE("/sp-connect/connector-instances/" + id).
		Expect().
		Status(http.StatusNoContent)
}

// cleanupInternalConnectorInstances deletes connector instances created for E2E tests
func cleanupInternalConnectorInstances(e *httpexpect.Expect) {
	list := e.GET("/sp-connect/connector-instances").
		Expect().
		Status(http.StatusOK).JSON().Array()

	for _, c := range list.Iter() {
		if c.Object().Value("connectorSpecId").String().Raw() == "internal" {
			deleteConnectorInstance(e, c.Object().Value("id").String().Raw())
		}
	}
}
