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
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	st "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/gorilla/mux"
)

const uuidRegex = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"

func StartServer(addr string, heathConfig HealthConfig, neoURL string, neoConf neoutils.ConnectionConfig, cacheTime time.Duration) (func(), error) {

	cbcService, err := NewContentByConceptService(neoURL, neoConf)
	if err != nil {
		return nil, fmt.Errorf("could not create concept service: %w", err)
	}

	handler := ContentByConceptHandler{
		ContentService:     cbcService,
		CacheControlHeader: strconv.FormatFloat(cacheTime.Seconds(), 'f', 0, 64),
		UUIDMatcher:        regexp.MustCompile(uuidRegex),
	}

	router := mux.NewRouter()
	logger.Info("registering handlers")
	router.HandleFunc("/content", handler.GetContentByConcept).Methods(http.MethodGet)

	monitoringRouter := handler.RegisterAdminHandlers(router, heathConfig)

	srv := http.Server{
		Addr:    addr,
		Handler: monitoringRouter,
	}

	logger.Infof("Application started on address %s with args %s", addr, os.Args)
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

func (h *ContentByConceptHandler) RegisterAdminHandlers(router *mux.Router, appConf HealthConfig) http.Handler {
	logger.Info("Registering healthcheck handlers")

	hc := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  appConf.AppSystemCode,
			Name:        appConf.AppName,
			Description: appConf.AppDescription,
			Checks:      h.checks(),
		},
		Timeout: 10 * time.Second,
	}

	router.HandleFunc("/__health", fthealth.Handler(hc))
	router.HandleFunc(st.BuildInfoPath, st.BuildInfoHandler)
	router.HandleFunc(api.DefaultPath, appConf.ApiEndpoint.ServeHTTP)
	router.HandleFunc(st.GTGPath, st.NewGoodToGoHandler(h.gtg))

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	if appConf.RequestLoggingEnabled {
		monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)
	}
	return monitoringRouter
}
