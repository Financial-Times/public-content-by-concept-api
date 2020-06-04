package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Financial-Times/api-endpoint"
	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/http-handlers-go/v2/httphandlers"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/public-content-by-concept-api/v2/content"
	st "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
)

type ServerConfig struct {
	Port          string
	APIYMLPath    string
	CacheTime     time.Duration
	RecordMetrics bool
	Log           *logger.UPPLogger

	AppSystemCode  string
	AppName        string
	AppDescription string

	NeoURL    string
	NeoConfig neoutils.ConnectionConfig
}

func StartServer(config ServerConfig) (func(), error) {

	apiEndpoint, err := api.NewAPIEndpointForFile(config.APIYMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to serve the API Endpoint for this service from file %s: %w", config.APIYMLPath, err)
	}
	cbcService, err := content.NewContentByConceptService(config.NeoURL, config.NeoConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create concept service: %w", err)
	}

	handler := Handler{
		ContentService:     cbcService,
		CacheControlHeader: strconv.FormatFloat(config.CacheTime.Seconds(), 'f', 0, 64),
	}

	hs := &HealthcheckService{
		AppSystemCode:  config.AppSystemCode,
		AppName:        config.AppName,
		AppDescription: config.AppDescription,
		ConnChecker:    cbcService.CheckConnection,
	}

	router := mux.NewRouter()
	config.Log.Debug("Registering service handlers")
	monitoredHandler := httphandlers.TransactionAwareRequestLoggingHandler(config.Log, http.HandlerFunc(handler.GetContentByConcept))
	if config.RecordMetrics {
		monitoredHandler = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoredHandler)
	}
	router.Handle("/content", monitoredHandler).Methods(http.MethodGet)

	config.Log.Debug("Registering admin handlers")
	router.HandleFunc("/__health", hs.HealthHandler()).Methods(http.MethodGet)
	router.HandleFunc(st.GTGPath, st.NewGoodToGoHandler(hs.GTG)).Methods(http.MethodGet)
	router.HandleFunc(st.BuildInfoPath, st.BuildInfoHandler).Methods(http.MethodGet)
	router.HandleFunc(api.DefaultPath, apiEndpoint.ServeHTTP).Methods(http.MethodGet)

	srv := http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		config.Log.Debugf("Application started on port %s with args %s", config.Port, os.Args)
		config.Log.Info("Start listening")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			config.Log.WithError(err).Error("Server closed with unexpected error")
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			config.Log.WithError(err).Error("Server shutdown with unexpected error")
		}
	}, nil
}
