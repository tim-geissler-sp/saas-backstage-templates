// Copyright (c) 2020, SailPoint Technologies, Inc. All rights reserved.
package infra

import (
	"github.com/gorilla/mux"
	"github.com/sailpoint/atlas-go/atlas/web"
	"github.com/sailpoint/sp-connect/internal/sp/connect/cmd"
	"net/http"
)

// buildRoutes configures all of the HTTP endpoints for the service.
func (s *ConnectService) buildRoutes() *mux.Router {
	r := web.NewRouter(web.DefaultAuthenticationConfig(s.TokenValidator))

	r.Handle("/hello-world", s.returnHelloWorld()).Methods("GET")

	//r.Handle("/connector-specifications", s.requireRight("sp:connector:create", s.createConnectorSpecification())).Methods("POST")
	//r.Handle("/connector-specifications", s.requireRight("sp:connector:read", s.listConnectorSpecifications())).Methods("GET")
	//r.Handle("/connector-specifications/{id}", s.requireRight("sp:connector:read", s.getConnectorSpecification())).Methods("GET")
	//r.Handle("/connector-specifications/{id}", s.requireRight("sp:connector:update", s.updateConnectorSpecification())).Methods("PUT")
	//r.Handle("/connector-specifications/validate", s.requireRight("sp:connector:create", s.validateConnectorSpecification())).Methods("POST")
	//r.Handle("/connector-specification/{id}", s.requireRight("sp:connector:update", s.patchConnectorSpecification())).Methods("PATCH")

	//r.Handle("/connector-instances", s.requireRight("sp:connector:create", s.createConnectorInstance())).Methods("POST")
	//r.Handle("/connector-instances", s.requireRight("sp:connector:read", s.listConnectorInstances())).Methods("GET")
	//r.Handle("/connector-instances/{id}", s.requireRight("sp:connector:delete", s.deleteConnectorInstance())).Methods("DELETE")
	//r.Handle("/connector-instances/{id}", s.requireRight("sp:connector:update", s.updateConnectorInstance())).Methods("PUT")
	//r.Handle("/connector-instances/{id}", s.requireRight("sp:connector:read", s.getConnectorInstance())).Methods("GET")
	//r.Handle("/connector-instances/{id}/commands", s.requireRight("sp:connector:invoke", s.invokeCommand())).Methods("POST")

	//r.Handle("/invocations/{id}/next-result", s.requireRight("sp:connector:invoke", s.iterateInvocationResult())).Methods("POST")
	//r.Handle("/invocations/{id}/cancel", s.requireRight("sp:connector:invoke", s.cancelInvocation())).Methods("POST")

	return r
}

// write http handler to return hello world (code 200)
func (s *ConnectService) returnHelloWorld() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cmd, err := cmd.NewHelloWorld()

		str, err := cmd.Handle(ctx) //output s

		if err != nil {
			//WriteJSONWithError(ctx, w, err)
			return
		}

		web.WriteJSON(ctx, w, str)
	}
}

// requireRight is a middleware function that ensures that the current request
// has the specified right before calling the next handler in the chain.
// Requests that are missing the specified right will be terminated with
// a 403 Forbidden response.
func (s *ConnectService) requireRight(right string, next http.Handler) http.Handler {
	m := web.RequireRights(s.AccessSummarizer, right)
	return m.Middleware(next)
}
