package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cli "github.com/jawher/mow.cli"

	logger "github.com/Financial-Times/go-logger/v2"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
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
	apiYml := app.String(cli.StringOpt{
		Name:   "api-yml",
		Value:  "./api.yml",
		Desc:   "Location of the API Swagger YML file.",
		EnvVar: "API_YML",
	})

	log := logger.NewUPPLogger(*appName, *logLevel)

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
			NeoConfig: func(config *neo4j.Config) {
				config.SocketConnectTimeout = 15 * time.Second
				config.MaxTransactionRetryTime = 1 * time.Minute
				config.Log = neo4j.ConsoleLogger(parseNeoLogLevel(*logLevel))
			},
		}

		stopSrv, err := StartServer(config, log)
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

// parseNeoLogLevel parses logrus log level constants to the neo4j logger equivalent
// TODO: move to separate library
func parseNeoLogLevel(level string) neo4j.LogLevel {
	switch strings.ToLower(level) {
	case "panic":
		return neo4j.ERROR
	case "fatal":
		return neo4j.ERROR
	case "error":
		return neo4j.ERROR
	case "warn", "warning":
		return neo4j.WARNING
	case "info":
		return neo4j.INFO
	case "debug":
		return neo4j.DEBUG
	default:
		return neo4j.ERROR
	}
}
