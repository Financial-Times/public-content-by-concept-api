package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"os/signal"
	"regexp"
	"syscall"

	"github.com/Financial-Times/api-endpoint"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/public-content-by-concept-api/content"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	_ "github.com/joho/godotenv/autoload"
)

const (
	appDescription = "An API for returning content related to a given concept"
	serviceName    = "public-content-by-concept-api"
	uuidRegex      = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"
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
		Value:  "http://localhost:7474/db/data",
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
	requestLoggingEnabled := app.Bool(cli.BoolOpt{
		Name:   "requestLoggingOn",
		Value:  false,
		Desc:   "Whether to log requests or not",
		EnvVar: "REQUEST_LOGGING_ENABLED",
	})
	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Level of logging in the service",
		EnvVar: "LOG_LEVEL",
	})
	apiYml := app.String(cli.StringOpt{
		Name:   "api-yml",
		Value:  "./api.yml",
		Desc:   "Location of the API Swagger YML file.",
		EnvVar: "API_YML",
	})

	logger.InitLogger(*appName, *logLevel)
	app.Action = func() {
		conf := neoutils.ConnectionConfig{
			BatchSize:     1024,
			Transactional: false,
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 100,
				},
				Timeout: 1 * time.Minute,
			},
			BackgroundConnect: true,
		}

		db, err := neoutils.Connect(*neoURL, &conf)
		if err != nil {
			logger.WithError(err).Fatal("Could not connect to Neo4j")
		}

		duration, err := time.ParseDuration(*cacheDuration)
		if err != nil {
			logger.WithError(err).Fatal("Failed to parse cache duration value")
		}

		apiEndpoint, err := api.NewAPIEndpointForFile(*apiYml)
		if err != nil {
			logger.WithError(err).WithField("file", *apiYml).Warn("Failed to serve the API Endpoint for this service. Please validate the Swagger YML and the file location.")
		}

		cbcService := content.NewContentByConceptService(db)

		handler := content.Handler{
			ContentService:     cbcService,
			CacheControlHeader: strconv.FormatFloat(duration.Seconds(), 'f', 0, 64),
			UUIDMatcher:        regexp.MustCompile(uuidRegex),
		}

		router := mux.NewRouter()
		handler.RegisterHandlers(router)
		appConf := content.HealthConfig{
			AppSystemCode:         *appSystemCode,
			AppName:               *appName,
			AppDescription:        appDescription,
			RequestLoggingEnabled: *requestLoggingEnabled,
			ApiEndpoint:           apiEndpoint,
		}

		monitoringRouter := handler.RegisterAdminHandlers(router, appConf)
		http.Handle("/", monitoringRouter)

		logger.Infof("Application started on port %s with args %s", *port, os.Args)
		if err := http.ListenAndServe(":"+*port, monitoringRouter); err != nil {
			logger.Fatalf("Unable to start server: %v", err)
		}
		waitForSignal()
	}
	app.Run(os.Args)
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
