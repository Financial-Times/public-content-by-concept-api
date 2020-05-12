package content

import (
	"github.com/Financial-Times/api-endpoint"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

type HealthConfig struct {
	AppSystemCode         string
	AppName               string
	AppDescription        string
	RequestLoggingEnabled bool
	ApiEndpoint           api.Endpoint
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
	return []fthealth.Check{
		{
			BusinessImpact:   "Cannot respond to API requests",
			Name:             "Check connectivity to Neo4j",
			PanicGuide:       "https://runbooks.ftops.tech/content-by-concept-api",
			Severity:         2,
			TechnicalSummary: "Cannot connect to Neo4j instance with at least one concept loaded in it",
			Checker:          h.ContentService.CheckConnection,
		},
	}
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
