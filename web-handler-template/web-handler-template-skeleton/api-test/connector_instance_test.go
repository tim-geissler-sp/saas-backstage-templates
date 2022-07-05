// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func TestCrudOperation(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	// Create
	create := e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	create.Value("id").String()
	create.Value("name").String().Equal("sp-connect karate test")
	create.Value("connectorSpecId").String()
	create.Value("config").Object().Empty()
	create.Value("created").String()

	// List
	list := e.GET("/sp-connect/connector-instances").
		Expect().
		Status(http.StatusOK).JSON().Array()
	list.Contains(create.Raw())

	// Update
	update := e.PUT("/sp-connect/connector-instances/" + create.Value("id").String().Raw()).
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test2",
			"connectorSpecId": "internal",
			"config": map[string]interface{}{
				"mockKey": "mockValue",
			},
		}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	update.Value("id").String().Equal(create.Value("id").String().Raw())
	update.Value("name").String().Equal("sp-connect karate test2")
	update.Value("connectorSpecId").String().Equal(create.Value("connectorSpecId").String().Raw())
	update.Value("created").String().Equal(create.Value("created").String().Raw())
	update.Value("config").Object().Equal(map[string]interface{}{
		"mockKey": "mockValue",
	})

	// Get
	read := e.GET("/sp-connect/connector-instances/" + create.Value("id").String().Raw()).
		Expect().
		Status(http.StatusOK).JSON().Object()

	read.Equal(update.Raw())

	// Delete
	e.DELETE("/sp-connect/connector-instances/" + create.Value("id").String().Raw()).
		Expect().
		Status(http.StatusNoContent)
}

func TestCrudOperationWithoutJwtToken(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)

	// Create connector without JWT should get 401
	e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusUnauthorized)

	// Update connector without JWT should get 401
	e.PUT("/sp-connect/connector-instances/abcd").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusUnauthorized)

	// Get connector without JWT should get 401
	e.GET("/sp-connect/connector-instances/abcd").
		Expect().
		Status(http.StatusUnauthorized)

	// Delete connector without JWT should get 401
	e.DELETE("/sp-connect/connector-instances/abcd").
		Expect().
		Status(http.StatusUnauthorized)
}

func TestBadCrudOperation(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	// Create connector request without spec ID should get 400
	e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"name":   "sp-connect karate test",
			"config": map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusBadRequest)

	// Create connector request without name should get 400
	e.POST("/sp-connect/connector-instances").
		WithJSON(map[string]interface{}{
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusBadRequest)

	// Update connector that does not exist should get 404
	e.PUT("/sp-connect/connector-instances/abcd").
		WithJSON(map[string]interface{}{
			"name":            "sp-connect karate test",
			"connectorSpecId": "internal",
			"config":          map[string]interface{}{},
		}).
		Expect().
		Status(http.StatusNotFound)

	// Get connector that does not exist should get 404
	e.GET("/sp-connect/connector-instances/abcd").
		Expect().
		Status(http.StatusNotFound)

	// Delete connector that does not exist should get 404
	e.DELETE("/sp-connect/connector-instances/abcd").
		Expect().
		Status(http.StatusNotFound)
}
