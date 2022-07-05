#!/bin/sh -ex
apk --no-cache add build-base pkgconfig librdkafka-dev
AWS_ACCESS_KEY_ID=none \
  AWS_SECRET_ACCESS_KEY=none \
  AWS_REGION=none \
  GOOS=linux go test -v -tags musl,integration -count=1 ./internal/...
