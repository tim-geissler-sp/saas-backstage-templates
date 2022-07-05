ARG BUILDER_IMAGE=golang:1.16-alpine3.13
ARG RUNTIME_IMAGE=alpine:3.13

# builder image
FROM $BUILDER_IMAGE AS builder

WORKDIR /app
RUN GOBIN=/app/bin GOOS=linux go install github.com/kisielk/errcheck@v1.6.0

COPY . .
RUN apk --no-cache add build-base pkgconfig librdkafka-dev
RUN GOOS=linux /app/bin/errcheck ./...
RUN GOOS=linux go test -tags musl -count=1 ./internal/...
RUN GOOS=linux go build -tags musl -o app ./cmd/sp-connect

# runtime image
FROM $RUNTIME_IMAGE
RUN apk --no-cache add ca-certificates librdkafka
WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/dist .
EXPOSE 7100
EXPOSE 7200

# default environment
ENV ATLAS_PRODUCTION true

CMD ["./app"]
