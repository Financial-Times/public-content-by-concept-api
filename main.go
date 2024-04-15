package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	opa "github.com/Financial-Times/opa-client-go"
	"github.com/Financial-Times/public-content-by-concept-api/v2/policy"
	cli "github.com/jawher/mow.cli"
)

const (
	appDescription = "An API for returning content related to a given concept"
	serviceName    = "public-content-by-concept-api"
)

func main() {
	app := cli.App(serviceName, appDescription)
	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "public-content-by-concept-api",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})
	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "Public Content by Concept API",
		Desc:   "Application name",
		EnvVar: "APP_NAME",
	})
	neoURL := app.String(cli.StringOpt{
		Name:   "neo-url",
		Value:  "bolt://localhost:7687",
		Desc:   "neo4j endpoint URL",
		EnvVar: "NEO_URL",
	})
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})
	cacheDuration := app.String(cli.StringOpt{
		Name:   "cache-duration",
		Value:  "30s",
		Desc:   "Duration Get requests should be cached for. e.g. 2h45m would set the max-age value to '7440' seconds",
		EnvVar: "CACHE_DURATION",
	})
	recordMetrics := app.Bool(cli.BoolOpt{
		Name:   "record-http-metrics",
		Desc:   "enable recording of http handler metrics",
		EnvVar: "RECORD_HTTP_METRICS",
		Value:  false,
	})
	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Level of logging in the service",
		EnvVar: "LOG_LEVEL",
	})
	dbDriverLogLevel := app.String(cli.StringOpt{
		Name:   "dbDriverLogLevel",
		Value:  "WARNING",
		Desc:   "Level of logging in the service",
		EnvVar: "DB_DRIVER_LOG_LEVEL",
	})
	apiYml := app.String(cli.StringOpt{
		Name:   "api-yml",
		Value:  "api/api.yml",
		Desc:   "Location of the API Swagger YML file.",
		EnvVar: "API_YML",
	})
	apiURL := app.String(cli.StringOpt{
		Name:   "publicAPIURL",
		Value:  "http://api.ft.com",
		Desc:   "API Gateway URL used when building the thing ID url in the response, in the format scheme://host",
		EnvVar: "PUBLIC_API_URL",
	})

	openPolicyAgentURL := app.String(cli.StringOpt{
		Name:   "openPolicyAgentURL",
		Value:  "http://localhost:8181",
		Desc:   "URL of the open policy angent",
		EnvVar: "OPA_URL",
	})

	log := logger.NewUPPLogger(*appName, *logLevel)
	dbLog := logger.NewUPPLogger(fmt.Sprintf("%s %s", *appName, "cmneo4j-driver"), *dbDriverLogLevel)

	app.Action = func() {

		duration, err := time.ParseDuration(*cacheDuration)
		if err != nil {
			log.WithError(err).Fatal("Failed to parse cache duration value")
		}

		config := ServerConfig{
			Port:           *port,
			APIYMLPath:     *apiYml,
			CacheTime:      duration,
			RecordMetrics:  *recordMetrics,
			AppSystemCode:  *appSystemCode,
			AppName:        *appName,
			AppDescription: appDescription,
			NeoURL:         *neoURL,
		}

		paths := map[string]string{
			policy.PublicationPolicyKey: policy.OpaPolicyPath,
		}
		opaClient := opa.NewOpenPolicyAgentClient(*openPolicyAgentURL, paths, opa.WithLogger(log))

		stopSrv, err := StartServer(config, log, dbLog, *apiURL, opaClient)
		if err != nil {
			log.WithError(err).Fatal("Could not start the server")
		}
		waitForSignal()
		stopSrv()
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
