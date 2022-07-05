// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.

/*
Package crypto contains various hashing and cryptography utilities.
*/
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"

	"github.com/dgryski/go-farm"
)

// HashFunc is a function alias for hashes
type HashFunc func([]byte) []byte

// ConcatenatedHash builds a new HashFunc out of a series of input hash functions.
// Their output is concatenated.
func ConcatenatedHash(hf ...HashFunc) HashFunc {
	return func(in []byte) []byte {
		var s []byte

		for _, f := range hf {
			s = append(s, f(in)...)
		}

		return s
	}
}

// Hash64 performs a 64-bit farm hash
func Hash64(in []byte) []byte {
	v := farm.Fingerprint64(in)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, v)

	return buf
}

// HashToString is a convenience function that hashes a series of values using Hash and converts
// the output to a hexidecimal string.
func HashToHexString(values ...[]byte) string {
	return hex.EncodeToString(Hash(values...))
}

// Hash uses a standard concatenated 64-bit farm hash.
func Hash(values ...[]byte) []byte {
	var buf []byte

	for _, v := range values {
		buf = append(buf, v...)
	}

	f := ConcatenatedHash(Hash64, Hash64, Hash64, Hash64)
	return f(buf)
}

// GenerateSecret generates cryptographically secure random byte sequence of the specified length.
func GenerateSecret(n int) ([]byte, error) {
	key := make([]byte, n)

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// GenerateHexSecret generates a cryptographically secure random byte sequence of the specified length
// and returns the hex-encoded value.
func GenerateHexSecret(n int) (string, error) {
	bytes, err := GenerateSecret(n)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// GenerateBase64Secret generates a cryptographically secure random byte sequence of the specified length
// and returns the base64-encoded value.
func GenerateBase64Secret(n int) (string, error) {
	bytes, err := GenerateSecret(n)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}
