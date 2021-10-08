package main

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

type ConnectionChecker func() (string, error)

type HealthcheckService struct {
	AppSystemCode  string
	AppName        string
	AppDescription string
	ConnChecker    ConnectionChecker
}

func (h *HealthcheckService) HealthHandler() func(w http.ResponseWriter, r *http.Request) {
	hc := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  h.AppSystemCode,
			Name:        h.AppName,
			Description: h.AppDescription,
			Checks:      h.Checks(),
		},
		Timeout: 10 * time.Second,
	}
	return fthealth.Handler(hc)
}

func (h *HealthcheckService) GTG() gtg.Status {
	var statusChecker []gtg.StatusChecker
	for _, c := range h.Checks() {
		checkFunc := func() gtg.Status {
			return gtgCheck(c.Checker)
		}
		statusChecker = append(statusChecker, checkFunc)
	}
	return gtg.FailFastParallelCheck(statusChecker)()
}

func (h *HealthcheckService) Checks() []fthealth.Check {
	return []fthealth.Check{
		{
			BusinessImpact:   "Cannot respond to API requests",
			Name:             "Check connectivity to Neo4j",
			PanicGuide:       "https://runbooks.ftops.tech/content-by-concept-api",
			Severity:         2,
			TechnicalSummary: "Cannot connect to Neo4j instance",
			Checker:          h.ConnChecker,
		},
	}
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
