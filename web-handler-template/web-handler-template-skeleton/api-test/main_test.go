// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/sailpoint/atlas-go/atlas/client"
)

var token string

var (
	orgUrl   = flag.String("url", "", "Org URL")
	username = flag.String("username", "", "Username")
	password = flag.String("password", "", "Password")
	env      = flag.String("env", "", "Environment")
)

func TestMain(m *testing.M) {
	flag.Parse()
	var err error
	tokenSource := client.NewPasswordTokenSource(http.DefaultClient, *orgUrl+"/oauth/token", *username, *password, *env == "prod")
	tokenReturn, err := tokenSource.GetToken(context.Background())
	if err != nil {
		log.Fatalf("failed to retrieve access token: %s", err)
	}
	token = tokenReturn.EncodedToken
	os.Exit(m.Run())
}
