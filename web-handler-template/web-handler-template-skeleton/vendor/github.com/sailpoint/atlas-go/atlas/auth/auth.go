// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/log"
)

var (
	// ErrTokenSignatureInvalid is an error returned during token validation that indicates that the signature
	// did not match our verification key, was expired, or otherwise invalid
	ErrTokenSignatureInvalid = errors.New("Token Signature Invalid")
)

// Token is a type that represents the data contained in a JWT signed
// by Oathkeeper.
type Token struct {

	// Encoded is the original JWT encoding.
	Encoded string

	// TenantID is the unique id of the tenant associated with this token.
	TenantID atlas.TenantID

	// Pod is the name of the pod associated with this token.
	Pod atlas.Pod

	// Org is the name of the tenant associated with this token.
	Org atlas.Org

	// IdentityID is the unique id of the identity associated with this token (may be empty).
	IdentityID atlas.IdentityID

	// IdentityName is the unique name of the identity associated with this token (may be empty).
	IdentityName atlas.IdentityName

	// Internal is an indicator that this token was minted on behalf of an OAuth client that represents
	// a first-party application (eg. The IdentityNow UI, CCG, Org Portal)
	Internal bool

	// Capabilities is a slice containing the capabilities associated with this token.
	Authorities []Authority

	// Scopes is a slice containing the scopes associated with this token.
	Scopes []Scope

	// Expiration is the timestamp after which this token is considered invalid.
	Expiration time.Time

	// ClientID is the client id of the Oauth token when token belongs to an Oauth client
	ClientID string
}

// TokenValidator is an interface for types that can parse and validate an encoded token.
type TokenValidator interface {

	// Parse decodes and validates an encoded token.
	Parse(encoded string) (*Token, error)
}

// Authority is a named bundle of access (aka Role) associated with a token.
type Authority string

// Scope is a named bundle of client api access (aka Role) associated with a token.
type Scope string

type KeyAndAlgorithm struct {
	SigningKey interface{}
	Algorithm  jwt.SigningMethod
}
type ComposedTokenValidator struct {
	ValidationList []KeyAndAlgorithm
}

type contextKey int

const (
	authTokenKey contextKey = iota
)

// GetToken extracts an access token from the specified context.
// Nil is returned if no token is associated with the context.
func GetToken(ctx context.Context) *Token {
	v := ctx.Value(authTokenKey)

	t, _ := v.(*Token)
	return t
}

// WithToken returns a new context derivied from the specified context and associated
// with the specified token.
func WithToken(ctx context.Context, t *Token) context.Context {
	return context.WithValue(ctx, authTokenKey, t)
}

// CreateRequestContext creates a new RequestContext with data associated
// with the access token.
func (t *Token) CreateRequestContext() *atlas.RequestContext {
	rc := &atlas.RequestContext{}
	rc.TenantID = t.TenantID
	rc.Pod = t.Pod
	rc.Org = t.Org
	rc.IdentityID = t.IdentityID
	rc.IdentityName = t.IdentityName

	return rc
}

// IsExpired gets whether or not this token has passed its' expiration date.
func (t *Token) IsExpired() bool {
	return t.Expiration.Before(time.Now().UTC())
}

// HasAuthority gets whether or not this token contains the specified
// authority.
func (t *Token) HasAuthority(authority Authority) bool {
	for _, a := range t.Authorities {
		if strings.EqualFold(string(a), string(authority)) {
			return true
		}
	}

	return false
}

// NewComposedTokenValidator constructs a new DefaultTokenValidator using the specified signing key
func NewComposedTokenValidator(signingKey []byte, signingMethod jwt.SigningMethod) *ComposedTokenValidator {
	v := &ComposedTokenValidator{}
	v.ValidationList = make([]KeyAndAlgorithm, 0, 1)
	v.AddValidator(signingKey, signingMethod)
	return v
}

func (v *ComposedTokenValidator) AddValidator(signingKey []byte, signingMethod jwt.SigningMethod) error {
	validationCombo := KeyAndAlgorithm{}
	validationCombo.Algorithm = signingMethod

	if _, ok := signingMethod.(*jwt.SigningMethodRSA); ok {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(signingKey)
		if err != nil {
			log.Errorf(nil, "error parsing signing key from PEM: %s", err)
			return err
		}
		validationCombo.SigningKey = publicKey
	} else {
		validationCombo.SigningKey = signingKey
	}

	v.ValidationList = append(v.ValidationList, validationCombo)
	return nil
}

// UseValidatorsToParse iterates through validationList and attempts to parse with each one, until
// one of the validators is sucessful.  If none can parse the string, then return nil.
func (v *ComposedTokenValidator) UseValidatorsToParse(s string) (*jwt.Token, error) {
	for i, tkValidator := range v.ValidationList {
		token, err := jwt.Parse(s, func(token *jwt.Token) (interface{}, error) {
			// Assert that Token.Method is either HMAC or RSA concrete type
			_, isRSA := token.Method.(*jwt.SigningMethodRSA)
			_, isHMAC := token.Method.(*jwt.SigningMethodHMAC)
			if isRSA || isHMAC {
				return tkValidator.SigningKey, nil
			} else {
				return nil, ErrTokenSignatureInvalid
			}
		})
		if err == nil {
			return token, err
		} else {
			log.Debugf(nil, "Couldn't parse token with Validator %d using %s method: %s", i, tkValidator.Algorithm.Alg(), err)
		}

	}
	return nil, errors.New("could not parse token")
}

// Parse parses and validates an encoded access token.
func (v *ComposedTokenValidator) Parse(s string) (*Token, error) {

	token, err := v.UseValidatorsToParse(s)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenSignatureInvalid
	}

	authToken := &Token{}
	authToken.Encoded = s
	authToken.Expiration = getTime(claims, "exp")

	if internal, ok := claims["internal"].(bool); ok {
		authToken.Internal = internal
	}

	if pod, ok := claims["pod"].(string); ok {
		authToken.Pod = atlas.Pod(pod)
	}

	if org, ok := claims["org"].(string); ok {
		authToken.Org = atlas.Org(org)
	}

	if tenantID, ok := claims["tenant_id"].(string); ok {
		authToken.TenantID = atlas.TenantID(tenantID)
	}

	if identityID, ok := claims["identity_id"].(string); ok {
		authToken.IdentityID = atlas.IdentityID(identityID)
	}

	if identityName, ok := claims["user_name"].(string); ok {
		authToken.IdentityName = atlas.IdentityName(identityName)
	}

	if authorities, ok := claims["authorities"].([]interface{}); ok {
		for _, authority := range authorities {
			if a, ok := authority.(string); ok {
				authToken.Authorities = append(authToken.Authorities, Authority(a))
			}
		}
	}

	if scopes, ok := claims["scope"].([]interface{}); ok {
		for _, scope := range scopes {
			if a, ok := scope.(string); ok {
				authToken.Scopes = append(authToken.Scopes, Scope(a))
			}
		}
	}

	if clientID, ok := claims["client_id"].(string); ok {
		authToken.ClientID = clientID
	}

	return authToken, nil
}

// getTime extracts a time from a JWT claim.
func getTime(m jwt.MapClaims, k string) time.Time {
	switch n := m[k].(type) {
	case float64:
		return time.Unix(int64(n), 0)
	case json.Number:
		v, _ := n.Int64()
		return time.Unix(v, 0)
	}

	return time.Time{}
}
