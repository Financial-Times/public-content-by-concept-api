package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Financial-Times/go-fthealth/v1a"
	log "github.com/Sirupsen/logrus"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpHandlers struct {
	contentDriver      driver
	cacheControlHeader string
}

//var maxAge = 24 * time.Hour

func (hh *httpHandlers) healthCheck() v1a.Check {
	return v1a.Check{
		BusinessImpact:   "Unable to respond to Public Content By Concept api requests",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/public-content-by-concept-api",
		Severity:         1,
		TechnicalSummary: `Cannot connect to Neo4j. If this check fails, check that Neo4j instance is up and running. You can find the neoUrl as a parameter in hieradata for this service.`,
		Checker:          hh.checker,
	}
}

func (hh *httpHandlers) checker() (string, error) {
	err := hh.contentDriver.checkConnectivity()
	if err == nil {
		return "Connectivity to neo4j is ok", err
	}
	return "Error connecting to neo4j", err
}

func (hh *httpHandlers) ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

//goodToGo returns a 503 if the healthcheck fails - suitable for use from varnish to check availability of a node
func (hh *httpHandlers) goodToGo(writer http.ResponseWriter, req *http.Request) {
	if _, err := hh.checker(); err != nil {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}

}

// buildInfoHandler - This is a stop gap and will be added to when we can define what we should display here
func (hh *httpHandlers) buildInfoHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "build-info")
}

// methodNotAllowedHandler handles 405
func (hh *httpHandlers) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

func (hh *httpHandlers) getContentByConcept(w http.ResponseWriter, r *http.Request) {

	m, _ := url.ParseQuery(r.URL.RawQuery)

	_, isAnnotatedByPresent := m["isAnnotatedBy"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if !isAnnotatedByPresent {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(
			`{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`))
		return
	}

	if len(m["isAnnotatedBy"]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(
			`{"message": "Only one concept uri should be provided"}`))
		return
	}

	conceptUri := m["isAnnotatedBy"][0]

	if conceptUri == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(
			`{"message": "Missing concept URI."}`))
		return
	}

	conceptUuid := strings.TrimPrefix(conceptUri, thingURIPrefix)

	limitParam := m.Get("limit")
	var limit int
	var err error

	if limitParam == "" {
		log.Infof("No limit provided. Using default: %v", defaultLimit)
		limit = defaultLimit
	} else {
		limit, err = strconv.Atoi(limitParam)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(`{"message":"Error limit is not a number: %s."}`, limitParam)
		w.Write([]byte(msg))
		return
	}

	fromDateParam := m.Get("fromDate")
	toDateParam := m.Get("toDate")
	var fromDateEpoch, toDateEpoch int64

	if fromDateParam == "" {
		log.Infof("No fromDate supplied.")
	} else {
		fromDateEpoch = convertStringToDateTimeEpoch(fromDateParam)
	}

	if toDateParam == "" {
		log.Infof("No toDate supplied")
	} else {
		toDateEpoch = convertStringToDateTimeEpoch(toDateParam)
	}

	contentList, found, err := hh.contentDriver.read(conceptUuid, limit, fromDateEpoch, toDateEpoch)

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := fmt.Sprintf(`{"message":"Error getting content for concept with uuid %s, err=%s"}`, conceptUuid, err.Error())
		w.Write([]byte(msg))
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf(`{"message":"No content found for concept with uuid %s."}`, conceptUuid)
		w.Write([]byte(msg))
		return
	}

	w.Header().Set("Cache-Control", hh.cacheControlHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(contentList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf(`{"message":"Error parsing content for concept with uuid %s, err=%s"}`, conceptUuid, err.Error())
		w.Write([]byte(msg))
	}
}

func convertStringToDateTimeEpoch(dateString string) int64 {
	datetime, err := time.Parse("2006-01-02", dateString)

	if err != nil {
		log.Warnf("Date can't be parsed: %v\n", dateString)
		return 0
	}

	return datetime.Unix()
}

const (
	thingURIPrefix = "http://api.ft.com/things/"
)
