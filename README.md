# Content Reader/Writer for Neo4j (roles-rw-neo4j)

__An API for reading/writing Content into Neo4j. Expects the content json supplied by the ingester. This is the equivalent to the content-writer-sesame but for writing to Neo4j not GraphDB

## Installation

For the first time:

`go get github.com/Financial-Times/content-rw-neo4j`

or update:

`go get -u github.com/Financial-Times/content-rw-neo4j`

## Running

`$GOPATH/bin/content-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --graphiteTCPAddress=graphite.ft.com:2003 --graphitePrefix=content.{env}.content.rw.neo4j.{hostname} --logMetrics=false

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080, batchSize of 1024, graphiteTCPAddress of "" (meaning metrics won't be written to Graphite), graphitePrefix of "" and logMetrics false.

NB: the default batchSize is much higher than the throughput the instance data ingester currently can cope with.

## Updating the model
Currently we use a subset of the fields that we get from the Ingester but if more fields are needed to be pulled in then update the model.go

The flow of information is as follows:

Kafka (CMSPublication) => Ingester => content-rw-neo4j

## Building

This service is built and deployed via Jenkins.

<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-roles-rw-neo4j/job/content-rw-neo4j-build/">Build job</a>
<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-roles-rw-neo4j/job/content-rw-neo4j-deploy-test/">Deploy job to Test</a>
<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-roles-rw-neo4j/job/content-rw-neo4j-deploy-prod/">Deploy job to Prod</a>

The build works via git tags. To prepare a new release
- update the version in /puppet/ft-content_rw_neo4j/Modulefile, e.g. to 0.0.12
- git tag that commit using `git tag 0.0.12`
- `git push --tags`

The deploy also works via git tag and you can also select the environment to deploy to.

## Endpoints
/content/{uuid}
### PUT
The only mandatory field is the uuid, and the uuid in the body must match the one used on the path.

Every request results in an attempt to update the content

A successful PUT results in 200.

TODO: Do we want batches for say large scale historical loads of content? => We run queries in batches. If a batch fails, all failing requests will get a 500 server error response.

Invalid json body input, or uuids that don't match between the path and the body will result in a 400 bad request response.

Example:
`curl -XPUT -H "X-Request-Id: 123" -H "Content-Type: application/json" localhost:8080/roles/3fa70485-3a57-3b9b-9449-774b001cd965 --data '{ "uuid": "4f1e764a-0744-11e4-bc71-002128161462", "title": "Profits plunge at Vatican bank", "byline": "By Giulia Segreti in Rome", "identifiers":[ { "authority": "http://api.ft.com/system/FTCOM-METHODE", "identifierValue": "4f1e764a-0744-11e4-bc71-002128161462" }], "publishedDate": "2014-07-08T13:52:52.000Z", "body": "<body><p>GeorgeProfits at the Vatican bank plunged last year after thousands of accounts were closed as part of an overhaul of the scandal-ridden institution.</p>\n<p>A total of 3,000 accounts have been closed and more than 2,000 have been blocked since April last year following the election of Pope Francis who has made rebuilding the tarnished image of the Vatican bank a priority.</p>\n<p>The vast majority of the closed accounts were small and had not been used for a long time, but the remaining 396 were closed after a screening process found that they did not meet the criteria for holding an account at the Vatican bank, which is officially known as the Institute for Religious Works.</p>\n</body>", "brands": [ { "id": http://api.ft.com/things/5c7592a8-1f0c-11e4-b0cb-b2227cce2b54" } ] }'`

Please note that we are only interested in the uuid, title and publishedDate at this time

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
