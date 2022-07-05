// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func TestConnectorSpecs(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	e.GET("/sp-connect/connector-specifications").
		Expect().
		Status(http.StatusOK).JSON().Array().Length().Ge(0)

	internalSpec := e.GET("/sp-connect/connector-specifications/internal").
		Expect().
		Status(http.StatusOK).JSON().Object()
	internalSpec.Value("topology").String().Equal("internal")
	internalSpec.Value("commands").Array().Length().Gt(0)
	internalSpec.Value("visibility").String().Equal("private")
}

func TestValidateSpecs(t *testing.T) {
	e := httpexpect.New(t, *orgUrl)
	e = e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	spec := readSpecFile("spec-valid.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(spec)).
		Expect().
		Status(http.StatusOK).JSON().Object()

	specInvalidCreateTemplateAccountGenerator := readSpecFile("spec-invalid-create-template-account-generator.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplateAccountGenerator)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

	specInvalidCreateTemplateIdentityAttribute := readSpecFile("spec-invalid-create-template-identity-attribute.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplateIdentityAttribute)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

	specInvalidCreateTemplatePasswordGenerator := readSpecFile("spec-invalid-create-template-password-generator.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplatePasswordGenerator)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

	specInvalidCreateTemplateStatic := readSpecFile("spec-invalid-create-template-static.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplateStatic)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

	specInvalidCreateTemplateUnknownType := readSpecFile("spec-invalid-create-template-unknown-type.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplateUnknownType)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

	specInvalidCreateTemplateMissingInitialValue := readSpecFile("spec-invalid-create-template-no-initial-value.json")
	e.POST("/sp-connect/connector-specifications/validate").
		WithJSON(json.RawMessage(specInvalidCreateTemplateMissingInitialValue)).
		Expect().
		Status(http.StatusBadRequest).JSON().Object()

}
