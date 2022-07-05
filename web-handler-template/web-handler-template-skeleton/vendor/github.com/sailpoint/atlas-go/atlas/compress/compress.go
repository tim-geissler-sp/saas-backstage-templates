// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package compress

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
)

// Compress64 compresses a UTF-8 string to a Base64-encoded compressed string.
func Compress64(input string) (string, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write([]byte(input))
	if err != nil {
		zw.Close()
		return "", err
	}

	if err := zw.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decompress64 decompresses a base64-encoded string into a UTF-8 string.
func Decompress64(input string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(decoded)

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}

	var outputBuffer bytes.Buffer
	if _, err := io.Copy(&outputBuffer, zr); err != nil {
		return "", err
	}

	if err := zr.Close(); err != nil {
		return "", err
	}

	return string(outputBuffer.Bytes()), nil
}
