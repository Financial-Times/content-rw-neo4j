# Content Reader/Writer for Neo4j (content-rw-neo4j)

[![Circle CI](https://circleci.com/gh/Financial-Times/content-rw-neo4j.svg?style=shield)](https://circleci.com/gh/Financial-Times/content-rw-neo4j)
[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/content-rw-neo4j)](https://goreportcard.com/report/github.com/Financial-Times/content-rw-neo4j)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/content-rw-neo4j/badge.svg)](https://coveralls.io/github/Financial-Times/content-rw-neo4j)

__An API for reading/writing Content into Neo4j. Expects the content JSON supplied by the ingester.__

## Installation

Download the source code, dependencies, and build the binary:

```
go get github.com/Financial-Times/content-rw-neo4j
cd $GOPATH/src/github.com/Financial-Times/content-rw-neo4j
go install
```

## Running

Run unit tests only:

```shell script
go test -v -race ./...
```

To execute the integration tests, you must provide `GITHUB_USERNAME` and
`GITHUB_TOKEN` because the service depends on internal repositories:

```shell script
GITHUB_USERNAME="<user-name>" GITHUB_TOKEN="<personal-access-token>" \
docker-compose -f docker-compose-tests.yml up -d --build && \
docker logs -f test-runner && \
docker-compose -f docker-compose-tests.yml down -v
```

To run the binary:

```
$GOPATH/bin/content-rw-neo4j \
   --neo-url={neo4jUrl} \
   --port={port} \
   --batchSize=50 \
```

All arguments are optional, run the following command to see the defaults:

```
$GOPATH/bin/content-rw-neo4j --help
```

## Building

The application is continuously built by CircleCI.

The docker image of the service is built by Dockerhub based on the git release tag.

To prepare a new git release, go to the repo page on GitHub and create a new release.

## Updating the Model

Currently we use a subset of the fields that we get from the Ingester but if more fields are needed to be pulled in then update the model.go

The flow of information is as follows: Kafka (CMSPublication) => Ingester => content-rw-neo4j

## Content Types

Currently, the following content types are eligible for being written into Neo:

* Article
* ContentPackage
* Content
* Video
* Graphic
* Audio

Additionally, any content payloads which contain a `body` property, will be written to Neo.

## API

Write content to Neo4j:

```
curl http://localhost:8080/content/:uuid -XPUT -H'Content-Type: application/json' --data '{"uuid":":uuid","body":"<body></body>"}'
```

Read content from Neo4j:

```
curl http://localhost:8080/content/:uuid'
```

Count content in Neo4j:

```
curl http://localhost:8080/content/__count'
```

Delete content from Neo4j:

```
curl http://localhost:8080/content/:uuid -XDELETE '
```

Please see the OpenAPI [spec](./api/api.yml) for details.

### Logging

* The application uses [go-logger](https://github.com/Financial-Times/go-logger ); the log file is initialised in [main.go](main.go).
