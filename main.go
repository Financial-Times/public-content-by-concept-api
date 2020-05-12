package main

import (
	"net/http"
	"os"
	"time"

	"os/signal"
	"syscall"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/public-content-by-concept-api/v2/content"
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

		duration, err := time.ParseDuration(*cacheDuration)
		if err != nil {
			logger.WithError(err).Fatal("Failed to parse cache duration value")
		}

		config := content.ServerConfig{
			Port:           *port,
			AppSystemCode:  *appSystemCode,
			AppName:        *appName,
			AppDescription: appDescription,
			NeoURL:         *neoURL,
			APIYMLPath:     *apiYml,
			CacheTime:      duration,
			NeoConfig: neoutils.ConnectionConfig{
				BatchSize:     1024,
				Transactional: false,
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						MaxIdleConnsPerHost: 100,
					},
					Timeout: 1 * time.Minute,
				},
				BackgroundConnect: true,
			},
		}

		stopSrv, err := content.StartServer(config)
		if err != nil {
			logger.WithError(err).Fatal("could not start the server")
		}
		waitForSignal()
		stopSrv()
	}
	app.Run(os.Args)
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
