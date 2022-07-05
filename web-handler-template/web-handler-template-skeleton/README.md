# sp-connect


## Overview

sp-connect is an IDN service responsible for providing extensible connectivity with various SaaS applications via
their respective connector(s).


## Getting Started

Pre-requisites:
- [Go 1.16](https://golang.org/dl/)

To use a specific branch of [atlas-go](https://github.com/sailpoint/atlas-go) (Optional) :
```bash
export GOPRIVATE=github.com/sailpoint

go get github.com/sailpoint/atlas-go@{branch/commitSHA}
go mod vendor
```

To run unit tests:
```bash
make mocks test
```

To run service in locally, assuming on megapod, you will need these environment variables:
```bash
export ATLAS_JWT_KEY_SSM=/service/oathkeeper/dev/encryption_string
export ATLAS_BASE_URL_PATTERN=https://%s.api.cloud.sailpoint.com
export SERVICE_LOCATION_PATTERN=https://$org.api.cloud.sailpoint.com/$service
export ATLAS_KAFKA_SERVERS=kafka-broker-megapod-us-east-11.cloud.sailpoint.com:9092,kafka-broker-megapod-us-east-12.cloud.sailpoint.com:9092,kafka-broker-megapod-us-east-13.cloud.sailpoint.com:9092,kafka-broker-megapod-us-east-14.cloud.sailpoint.com:9092,kafka-broker-megapod-us-east-15.cloud.sailpoint.com:9092,kafka-broker-megapod-us-east-16.cloud.sailpoint.com:9092
export AWS_REGION=us-east-1
export AWS_SECRET_ACCESS_KEY=<your-aws-secret-key>
export AWS_ACCESS_KEY_ID=<your-aws-access-key-id>
export INTERNAL_COMMAND_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/406205545357/sp-connect-us-east-1-command-internal.fifo
export CONFIG_SECRET=9f155dcc81ed622cfe244e99b3c75c54
export RESPONSE_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/406205545357/sp-connect-megapod-useast1.fifo
export RUNTIME_COMMAND_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/406205545357/sp-connect-command-runtime-megapod-useast1.fifo
export CONNECTOR_INVOCATION_TABLE_NAME=connector-invocation-megapod-useast1
```

To run service in [Beacon](https://sailpoint.atlassian.net/wiki/x/_4BiDQ) mode:
```bash
export BEACON_TENANT={org-name}:{vpn-name}
export ATLAS_REST_PORT=7100

make run
```

---

To build a docker image:
```bash
make docker/build
```

To build and publish a docker image:
```bash
make docker/push
```
