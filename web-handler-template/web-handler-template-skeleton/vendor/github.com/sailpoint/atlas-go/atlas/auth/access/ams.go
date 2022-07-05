// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package access

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sailpoint/atlas-go/atlas/auth"
	"github.com/sailpoint/atlas-go/atlas/client"
)

// legacyCapabilityMap maps the legacy role names to the new capability names, as referenced in AMS.
var legacyCapabilityMap = map[string]string{
	"ORG_ADMIN":    "idn:admin",
	"HELPDESK":     "idn:helpdesk",
	"DASHBOARD":    "idn:dashboard",
	"CERT_ADMIN":   "idn:cert-admin",
	"REPORT_ADMIN": "idn:report-admin",
	"ROLE_ADMIN":   "idn:role-admin",
	"ROLE_SUBADMIN": "idn:role-subadmin",
	"SOURCE_ADMIN": "idn:source-admin",
	"SOURCE_SUBADMIN": "idn:source-subadmin",
	"CLOUD_GOV_ADMIN": "cam:cloud-gov-admin",
	"CLOUD_GOV_USER": "cam:cloud-gov-user",
	"API":          "idn:api",
}

// authorizationSignature is the format of the input requests for summarization in AMS.
type authorizationSignature struct {
	Tenant       string   `json:"tenant"`
	Capabilities []string `json:"capabilities"`
	Scopes       []string `json:"scopes"`
}

// amsSummarizer is a Summarizer implementation that calls out to the AMS microservice to get
// an access summary.
type amsSummarizer struct {
	baseURLProvider        client.BaseURLProvider
	internalClientProvider client.InternalClientProvider
}

// newAmsSummarizer constructs a new amsSummarizer implementation.
func newAmsSummarizer(baseURLProvider client.BaseURLProvider, internalClientProvider client.InternalClientProvider) *amsSummarizer {
	s := &amsSummarizer{}
	s.baseURLProvider = baseURLProvider
	s.internalClientProvider = internalClientProvider

	return s
}

// Summarize builds a signature from the specified token and invokes the AMS microservice
// to get a summary result.
func (s *amsSummarizer) Summarize(ctx context.Context, t *auth.Token) (*Summary, error) {
	signature := &authorizationSignature{}
	signature.Tenant = string(t.Org)

	for _, a := range t.Authorities {
		signature.Capabilities = append(signature.Capabilities, mapCapability(string(a)))
	}

	for _, scope := range t.Scopes {
		signature.Scopes = append(signature.Scopes, string(scope))
	}

	jsonBody, err := json.Marshal(signature)
	if err != nil {
		return nil, err
	}

	url := client.BuildAPIURL(s.baseURLProvider, t.Org, "beta/summarize-authorization")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/json")

	client := s.internalClientProvider.GetInternalClient(t.TenantID, t.Org)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status: %d", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)

	summary := &Summary{}
	if err = decoder.Decode(summary); err != nil {
		return nil, err
	}

	return summary, nil
}

// mapCapability converts an input authority to a capability ID. Legacy capabilities (eg. ORG_ADMIN)
// are mapped to their modern counterparts (eg. idn:admin)
func mapCapability(c string) string {
	if v, ok := legacyCapabilityMap[c]; ok {
		return v
	}

	return c
}
