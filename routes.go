package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"

	"github.com/Financial-Times/api-endpoint"
	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/http-handlers-go/v2/httphandlers"
	"github.com/Financial-Times/public-content-by-concept-api/v2/content"
	st "github.com/Financial-Times/service-status-go/httphandlers"
)

type ServerConfig struct {
	Port          string
	APIYMLPath    string
	CacheTime     time.Duration
	RecordMetrics bool

	AppSystemCode  string
	AppName        string
	AppDescription string

	NeoURL string
}

func StartServer(config ServerConfig, log *logger.UPPLogger, dbLog *logger.UPPLogger) (func(), error) {
	apiEndpoint, err := api.NewAPIEndpointForFile(config.APIYMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to serve the API Endpoint for this service from file %s: %w", config.APIYMLPath, err)
	}

	neoDriver, err := cmneo4j.NewDefaultDriver(config.NeoURL, dbLog)
	if err != nil {
		log.WithError(err).Fatal("Could not initiate cmneo4j driver")
	}

	cbcService := content.NewContentByConceptService(neoDriver)

	handler := Handler{
		ContentService:     cbcService,
		CacheControlHeader: strconv.FormatFloat(config.CacheTime.Seconds(), 'f', 0, 64),
		Log:                log,
	}

	hs := &HealthcheckService{
		AppSystemCode:  config.AppSystemCode,
		AppName:        config.AppName,
		AppDescription: config.AppDescription,
		ConnChecker:    cbcService.CheckConnection,
	}

	router := mux.NewRouter()
	log.Debug("Registering service handlers")
	monitoredHandler := httphandlers.TransactionAwareRequestLoggingHandler(log, http.HandlerFunc(handler.GetContentByConcept))
	if config.RecordMetrics {
		monitoredHandler = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoredHandler)
	}

	monitoredImplicitHandler := httphandlers.TransactionAwareRequestLoggingHandler(log, http.HandlerFunc(handler.GetContentByConceptImplicitly))
	if config.RecordMetrics {
		monitoredImplicitHandler = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoredImplicitHandler)
	}

	router.Handle("/content", monitoredHandler).Methods(http.MethodGet)
	router.Handle("/content/{conceptUUID}/implicitly", monitoredImplicitHandler).Methods(http.MethodGet)

	log.Debug("Registering admin handlers")
	router.HandleFunc("/__health", hs.HealthHandler()).Methods(http.MethodGet)
	router.HandleFunc(st.GTGPath, st.NewGoodToGoHandler(hs.GTG)).Methods(http.MethodGet)
	router.HandleFunc(st.BuildInfoPath, st.BuildInfoHandler).Methods(http.MethodGet)
	router.HandleFunc(api.DefaultPath, apiEndpoint.ServeHTTP).Methods(http.MethodGet)

	srv := http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		log.Debugf("Application started on port %s with args %s", config.Port, os.Args)
		log.Info("Start listening")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Error("Server closed with unexpected error")
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.WithError(err).Error("Server shutdown with unexpected error")
		}

		if err := neoDriver.Close(); err != nil {
			log.WithError(err).Error("Neo4j Driver failed to close")
		}
	}, nil
}
