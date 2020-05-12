package content

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/Financial-Times/api-endpoint"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	st "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/gorilla/mux"
)

const uuidRegex = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"

type ServerConfig struct {
	Port           string
	AppSystemCode  string
	AppName        string
	AppDescription string
	NeoURL         string
	APIYMLPath     string
	CacheTime      time.Duration
	NeoConfig      neoutils.ConnectionConfig
}

func StartServer(config ServerConfig) (func(), error) {

	apiEndpoint, err := api.NewAPIEndpointForFile(config.APIYMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to serve the API Endpoint for this service from file %s: %w", config.APIYMLPath, err)
	}
	cbcService, err := NewContentByConceptService(config.NeoURL, config.NeoConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create concept service: %w", err)
	}

	handler := ContentByConceptHandler{
		ContentService:     cbcService,
		CacheControlHeader: strconv.FormatFloat(config.CacheTime.Seconds(), 'f', 0, 64),
		UUIDMatcher:        regexp.MustCompile(uuidRegex),
	}

	hs := &HealthcheckService{
		AppSystemCode:  config.AppSystemCode,
		AppName:        config.AppName,
		AppDescription: config.AppDescription,
		ConnChecker:    cbcService.CheckConnection,
	}

	router := mux.NewRouter()
	logger.Info("registering handlers")
	router.HandleFunc("/content", handler.GetContentByConcept).Methods(http.MethodGet)

	logger.Info("Registering healthcheck handlers")
	router.HandleFunc("/__health", hs.HealthHandler()).Methods(http.MethodGet)
	router.HandleFunc(st.BuildInfoPath, st.BuildInfoHandler).Methods(http.MethodGet)
	router.HandleFunc(api.DefaultPath, apiEndpoint.ServeHTTP).Methods(http.MethodGet)
	router.HandleFunc(st.GTGPath, st.NewGoodToGoHandler(hs.GTG)).Methods(http.MethodGet)

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	srv := http.Server{
		Addr:    ":" + config.Port,
		Handler: monitoringRouter,
	}

	logger.Infof("Application started on port %s with args %s", config.Port, os.Args)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logger.WithError(err).Error("server closed with unexpected error")
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			logger.WithError(err).Error("server shutdown with unexpected error")
		}
	}, nil
}
