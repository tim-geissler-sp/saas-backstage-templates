// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package crypto

import (
	"fmt"

	jose "gopkg.in/square/go-jose.v2"
)

// Encoder is an interface for encoding of bytes.
type Encoder interface {
	Encode(value []byte) ([]byte, error)
}

// Decoder is an interface for decoding bytes.
type Decoder interface {
	Decode(encoded []byte) ([]byte, error)
}

type JWECodec struct {
	encrypter jose.Encrypter
	secret    []byte
}

// NewJWECodec constructs an object capable of encoding and decoding bytes using a JWE token
func NewJWECodec(secret []byte) (*JWECodec, error) {
	if len(secret) != 16 {
		return nil, fmt.Errorf("secret must be exactly 16 bytes long")
	}

	encrypter, err := jose.NewEncrypter(jose.A128GCM, jose.Recipient{Algorithm: jose.A128GCMKW, Key: secret}, nil)
	if err != nil {
		return nil, err
	}

	c := &JWECodec{}
	c.encrypter = encrypter
	c.secret = secret

	return c, nil
}

// Encode will encode the specified bytes using the JWE secret
func (c *JWECodec) Encode(value []byte) ([]byte, error) {
	obj, err := c.encrypter.Encrypt(value)
	if err != nil {
		return nil, err
	}

	serialized, err := obj.CompactSerialize()
	if err != nil {
		return nil, err
	}

	return []byte(serialized), nil
}

// Decode will decode the specified bytes using the JWE secret
func (c *JWECodec) Decode(encoded []byte) ([]byte, error) {
	if len(encoded) == 0 {
		return nil, nil
	}

	obj, err := jose.ParseEncrypted(string(encoded))
	if err != nil {
		return nil, err
	}

	return obj.Decrypt(c.secret)
}
