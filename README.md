# Public API for Content By Concept (public-content-by-concept-api)

[![Circle CI](https://circleci.com/gh/Financial-Times/public-content-by-concept-api.svg?style=shield)](https://circleci.com/gh/Financial-Times/public-content-by-concept-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/public-content-by-concept-api)](https://goreportcard.com/report/github.com/Financial-Times/public-content-by-concept-api)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/public-content-by-concept-api/badge.svg)](https://coveralls.io/github/Financial-Times/public-content-by-concept-api)

__A public API which returns an ordered list of the most recently annotated content about a given concept__


## Installation & running locally
* `go get -u github.com/Financial-Times/public-content-by-concept-api`
* `cd $GOPATH/src/github.com/Financial-Times/public-content-by-concept-api`
* `go install`
* `$GOPATH/bin/public-content-by-concept-api --neo-url={neo4jUrl} --port={port} --log-level={DEBUG|INFO|WARN|ERROR}--cache-duration{e.g. 22h10m3s} --requestLoggingEnabled=false`

  Optional arguments are:
  * --neo-url defaults to bolt://neo4j:7687, which is the default url to connect to single Neo4j instance using bolt protocol.
  * --port defaults to 8080.
  * --cache-duration defaults to 1 hour
  * --logLevel set level of app logging, request critical logs are info level with more helpful logs found at debug
  * --requestLoggingEnabled when true will toggle logging of both admin endpoints(health/gtg) as well as http endpoints_

## Testing
* Unit tests only: `go test -race ./...`
* Unit and integration tests:
  
  In order for the integration tests to execute you must provide GITHUB_USERNAME and GITHUB_TOKEN values, because the service is depending on internal repositories.
    ```
    GITHUB_USERNAME="<user-name>" GITHUB_TOKEN="<personal-access-token>" \
    docker-compose -f docker-compose-tests.yml up -d --build && \
    docker logs -f test-runner && \
    docker-compose -f docker-compose-tests.yml down
    ```

## Examples for the endpoint that does not return implicitly annotated content: 
* `curl http://localhost:8080/content?isAnnotatedBy=http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54 `
* `curl http://localhost:8080/content?isAnnotatedBy=http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54&fromDate=2016-01-02&toDate=2016-01-05&limit=200`
* `curl http://localhost:8080/content?isAnnotatedBy=http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54&fromDate=2016-01-02&toDate=2016-01-05&page=3&limit=200`

*Note: Optional request params: limit (number of items to return), page, toDate, fromDate. isAnnotatedBy param accepts both full concept URI or just the UUID*

## Examples for the endpoint that returns implicitly annotated content:
* `curl http://localhost:8080/content/http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54/implicitly `

## API definition
Based on the following [google doc](https://docs.google.com/a/ft.com/document/d/1YjqNYEXkc0Ip-6bGttwnPcAh2XKG6tgzmojTdq8gM2s)

## Admin Enpoints
Healthcheck: [http://localhost:8080/__health](http://localhost:8080/__health)
Gtg: [http://localhost:8080/__gtg](http://localhost:8080/__gtg)
Build-Info: [http://localhost:8080/__build-info](http://localhost:8080/__build-info)
