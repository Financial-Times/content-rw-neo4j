# Content Reader/Writer for Neo4j (content-rw-neo4j)
[![Circle CI](https://circleci.com/gh/Financial-Times/content-rw-neo4j.svg?style=shield)](https://circleci.com/gh/Financial-Times/content-rw-neo4j)[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/content-rw-neo4j)](https://goreportcard.com/report/github.com/Financial-Times/content-rw-neo4j) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/content-rw-neo4j/badge.svg)](https://coveralls.io/github/Financial-Times/content-rw-neo4j)

__An API for reading/writing Content into Neo4j. Expects the content json supplied by the ingester. This is the equivalent to the content-writer-sesame but for writing to Neo4j not GraphDB__

## Installation

For the first time:

`go get github.com/Financial-Times/content-rw-neo4j`

or update:

`go get -u github.com/Financial-Times/content-rw-neo4j`

## Running

`$GOPATH/bin/content-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --graphiteTCPAddress=graphite.ft.com:2003 --graphitePrefix=content.{env}.content.rw.neo4j.{hostname} --logMetrics=false

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080, batchSize of 1024, graphiteTCPAddress of "" (meaning metrics won't be written to Graphite), graphitePrefix of "" and logMetrics false.

## Updating the model
Currently we use a subset of the fields that we get from the Ingester but if more fields are needed to be pulled in then update the model.go

The flow of information is as follows:

Kafka (CMSPublication) => Ingester => content-rw-neo4j

## Building

Continuosly built be CircleCI. The docker image of the service is built by Dockerhub based on the git release tag. 
To prepare a new git release, go to the repo page on GitHub and create a new release.

## Endpoints
/content/{uuid}
### PUT
The only mandatory field is the uuid, and the uuid in the body must match the one used on the path.

Every request results in an attempt to update the content

A successful PUT results in 200.

**PLEASE NOTE:**

We  are only interested in pieces of Content that can be annotated therefore we only want to ingest/write pieces of Content that have a body.

If the incoming JSON doesn't have a body then we ignore it and only log that we are ignoring. TODO: We should return a 204 not a 200

We run queries in batches. If a batch fails, all failing requests will get a 500 server error response.

Invalid json body input, or uuids that don't match between the path and the body will result in a 400 bad request response.

Example:
    `curl -XPUT -H "X-Request-Id: 123" -H "Content-Type: application/json" localhost:8080/content/3fa70485-3a57-3b9b-9449-774b001cd965 --data '{ "uuid": "4f1e764a-0744-11e4-bc71-002128161462", "title": "Profits plunge at Vatican bank", "byline": "By Giulia Segreti in Rome", "identifiers":[ { "authority": "http://api.ft.com/system/FTCOM-METHODE", "identifierValue": "4f1e764a-0744-11e4-bc71-002128161462" }], "publishedDate": "2014-07-08T13:52:52.000Z", "body": "<body><p>GeorgeProfits at the Vatican bank plunged last year after thousands of accounts were closed as part of an overhaul of the scandal-ridden institution.</p>\n<p>A total of 3,000 accounts have been closed and more than 2,000 have been blocked since April last year following the election of Pope Francis who has made rebuilding the tarnished image of the Vatican bank a priority.</p>\n<p>The vast majority of the closed accounts were small and had not been used for a long time, but the remaining 396 were closed after a screening process found that they did not meet the criteria for holding an account at the Vatican bank, which is officially known as the Institute for Religious Works.</p>\n</body>", "brands": [ { "id": "http://api.ft.com/things/5c7592a8-1f0c-11e4-b0cb-b2227cce2b54" } ], "storyPackage": "14a68464-c398-4fd4-bcc1-c06b30bf8d45" }'`


### GET
The internal read should return what got written (i.e., there isn't a public read for this representation of content and this is not intended to ever be public either because the public read is the /content endpoint served by mongo)

If not found, you'll get a 404 response.

Empty fields are omitted from the response.
`curl -H "X-Request-Id: 123" localhost:8080/content/344fdb1d-0585-31f7-814f-b478e54dbe1f`

### DELETE
Will return 204 if successful, 404 if not found
`curl -XDELETE -H "X-Request-Id: 123" localhost:8080/content/344fdb1d-0585-31f7-814f-b478e54dbe1f`

### Admin endpoints
Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping) or [http://localhost:8080/__ping](http://localhost:8080/__ping)
