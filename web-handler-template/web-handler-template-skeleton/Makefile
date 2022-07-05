VERSION ?= dev

all: test

run:
	cd dist; go run ../cmd/sp-connect/main.go

clean:
	go clean ./cmd/sp-connect
	rm -f sp-connect

test:
	go test -count=1 ./internal/...

integration:
	go test -count=1 -tags integration ./internal/...

cover:
	go test -tags integration -coverprofile cover.out ./internal/...
	go tool cover -html=cover.out
	rm cover.out

docker/integration:
	docker run -v /var/run/docker.sock:/var/run/docker.sock -v `pwd`:/app -w /app --network host golang:1.16-alpine3.13 ./scripts/docker-integration.sh

docker/login:
	$$(aws ecr get-login --no-include-email --region us-east-1)
	#aws ecr get-login-password | docker login --username AWS --password-stdin 406205545357.dkr.ecr.us-east-1.amazonaws.com

docker/build: docker/login
	docker build -t sailpoint/sp-connect:$(VERSION) --build-arg BUILDER_IMAGE=406205545357.dkr.ecr.us-east-1.amazonaws.com/mirror/golang:1.16-alpine3.13 \
	--build-arg RUNTIME_IMAGE=406205545357.dkr.ecr.us-east-1.amazonaws.com/mirror/alpine:3.13 .

docker/push: docker/build
	docker tag sailpoint/sp-connect:$(VERSION) 406205545357.dkr.ecr.us-east-1.amazonaws.com/sailpoint/sp-connect:$(VERSION)
	docker push 406205545357.dkr.ecr.us-east-1.amazonaws.com/sailpoint/sp-connect:$(VERSION)

mocks:
	# Ref: https://github.com/golang/mock
	mockgen -source=internal/sp/connect/model/common.go -destination=internal/sp/connect/mocks/common.go -package=mocks
	mockgen -source=internal/sp/connect/model/queue_service.go -destination=internal/sp/connect/mocks/queue_service.go -package=mocks
	mockgen -source=internal/sp/connect/model/schema.go -destination=internal/sp/connect/mocks/schema.go -package=mocks
	mockgen -source=internal/sp/connect/model/key_value_store.go -destination=internal/sp/connect/mocks/key_value_store.go -package=mocks
	mockgen -source=internal/sp/connect/model/response_handler.go -destination=internal/sp/connect/mocks/response_handler.go -package=mocks

.PHONY: clean test run docker/build docker/push docker/login mocks

docker/api-test:
	docker build -t sp-connect-test-api -f Dockerfile.api_test .
	docker run -v ${HOME}/.aws:/root/.aws sp-connect-test-api go test -v /app/api-test/... -username $(username) -password $(password) -url $(url) -env $(env)
	