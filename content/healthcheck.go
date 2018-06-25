package content

import (
	"net/http"
	"time"

	"github.com/Financial-Times/api-endpoint"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/service-status-go/gtg"
	st "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type HealthConfig struct {
	AppSystemCode         string
	AppName               string
	AppDescription        string
	RequestLoggingEnabled bool
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

	apiEndpoint, err := api.NewAPIEndpointForFile("./api.yml")
	if err != nil {
		log.WithError(err).WithField("file", "./api.yml").Warn("Failed to serve the API Endpoint for this service. Please validate the Swagger YML and the file location.")
	}

	router.HandleFunc("/__health", fthealth.Handler(hc))
	router.HandleFunc(st.BuildInfoPath, st.BuildInfoHandler)
	router.HandleFunc(api.DefaultPath, apiEndpoint.ServeHTTP)
	router.HandleFunc(st.GTGPath, st.NewGoodToGoHandler(h.gtg))

	var monitoringRouter http.Handler = router
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	if appConf.RequestLoggingEnabled {
		monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)
	}
	return monitoringRouter
}

func (h *ContentByConceptHandler) gtg() gtg.Status {
	var statusChecker []gtg.StatusChecker
	for _, c := range h.checks() {
		checkFunc := func() gtg.Status {
			return gtgCheck(c.Checker)
		}
		statusChecker = append(statusChecker, checkFunc)
	}
	return gtg.FailFastParallelCheck(statusChecker)()
}

func (h *ContentByConceptHandler) checks() []fthealth.Check {
	return []fthealth.Check{h.makeNeo4jAvailabilityCheck()}
}

func (h *ContentByConceptHandler) makeNeo4jAvailabilityCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Cannot respond to API requests",
		Name:             "Check connectivity to Neo4j - neoURL is a parameter in hieradata for this service",
		PanicGuide:       "https://dewey.ft.com/content-by-concept-api.html",
		Severity:         2,
		TechnicalSummary: "Cannot connect to Neo4j instance with at least one concept loaded in it",
		Checker:          h.checkNeo4jAvailability,
	}
}

func (h *ContentByConceptHandler) checkNeo4jAvailability() (string, error) {
	err := h.ContentService.Check()
	if err != nil {
		return "Could not connect to database!", err
	}
	return "", nil
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
