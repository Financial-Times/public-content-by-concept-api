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

```shell
Options:
  --app-system-code       System Code of the application (env $APP_SYSTEM_CODE) (default "public-content-by-concept-api")
  --app-name              Application name (env $APP_NAME) (default "Public Content by Concept API")
  --neo-url               neo4j endpoint URL (env $NEO_URL) (default "bolt://localhost:7687")
  --port                  Port to listen on (env $APP_PORT) (default "8080")
  --cache-duration        Duration Get requests should be cached for. e.g. 2h45m would set the max-age value to '7440' seconds (env $CACHE_DURATION) (default "30s")
  --record-http-metrics   enable recording of http handler metrics (env $RECORD_HTTP_METRICS)
  --logLevel              Level of logging in the service (env $LOG_LEVEL) (default "INFO")
  --dbDriverLogLevel      Level of logging in the service (env $DB_DRIVER_LOG_LEVEL) (default "WARNING")
  --api-yml               Location of the API Swagger YML file. (env $API_YML) (default "./api.yml")
  --publicAPIURL          API Gateway URL used when building the thing ID url in the response, in the format scheme://host (env $PUBLIC_API_URL) (default "http://api.ft.com")
  --ftURL                 FT's URL used when building the ID url in the response, in the format scheme://host (env $FT_URL) (default "http://www.ft.com")
```

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
Full API definition and description of supported endpoints can be found in the [Open API specification](./api/api.yml).

## Admin Enpoints
Healthcheck: [http://localhost:8080/__health](http://localhost:8080/__health)
Gtg: [http://localhost:8080/__gtg](http://localhost:8080/__gtg)
Build-Info: [http://localhost:8080/__build-info](http://localhost:8080/__build-info)
