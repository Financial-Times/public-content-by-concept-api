package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/go-fthealth/v1a"
	log "github.com/Sirupsen/logrus"
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

func (hh *httpHandlers) selectContentByConceptHandler(w http.ResponseWriter, r *http.Request) {
	predicate, found, err := getSingleValueQueryParameter(r, "withPredicate")
	if err != nil {
		writeHTTPMessage(w, http.StatusBadRequest, `More than one value found for query parameter "withPredicate". Expecting exactly one valid absolute predicate URI.`)
		return
	}

	if found {
		hh.getContentByConceptWithPredicate(w, r, predicate)
	} else {
		hh.getContentByConcept(w, r)
	}
}

func (hh *httpHandlers) getContentByConcept(w http.ResponseWriter, r *http.Request) {
	conceptURI, found, err := getSingleValueQueryParameter(r, "isAnnotatedBy")
	if !found {
		writeHTTPMessage(w, http.StatusBadRequest, `Missing query parameter "isAnnotatedBy". Expecting exactly one valid absolute Concept URI.`)
		return
	}

	if err != nil {
		writeHTTPMessage(w, http.StatusBadRequest, `More than one value found for query parameter "isAnnotatedBy". Expecting exactly one valid absolute Concept URI.`)
		return
	}

	if strings.TrimSpace(conceptURI) == "" {
		writeHTTPMessage(w, http.StatusBadRequest, `No value specified for Concept URI.`)
		return
	}

	limitText, found, err := getSingleValueQueryParameter(r, "limit")
	if err != nil {
		writeHTTPMessage(w, http.StatusBadRequest, `Please provide one value for "limit".`)
		return
	}

	var limit int
	if !found {
		log.Infof("No limit provided. Using default: %v", defaultLimit)
		limit = defaultLimit
	} else {
		limit, err = strconv.Atoi(limitText)
		if err != nil {
			writeHTTPMessage(w, http.StatusBadRequest, fmt.Sprintf("Error limit is not a number: %s.", limitText))
			return
		}
	}

	conceptUUID := strings.TrimPrefix(conceptURI, thingURIPrefix)

	toDateEpoch, found, err := getDateParam(r, "toDate")
	if !found {
		log.Infof("No toDate supplied.")
	} else if err != nil {
		writeHTTPMessage(w, http.StatusBadRequest, `More than one value for "toDate" supplied. Please provide exactly one value.`)
		return
	}

	fromDateEpoch, found, err := getDateParam(r, "fromDate")
	if !found {
		log.Infof("No fromDate supplied.")
	} else if err != nil {
		writeHTTPMessage(w, http.StatusBadRequest, `More than one value for "fromDate" supplied. Please provide exactly one value.`)
		return
	}

	contentByConcept, found, err := hh.contentDriver.read(conceptUUID, limit, fromDateEpoch, toDateEpoch)
	if err != nil {
		writeHTTPMessage(w, http.StatusServiceUnavailable, fmt.Sprintf("Error getting content for concept with uuid %s, err=%s", conceptUUID, err.Error()))
		return
	}

	if !found {
		writeHTTPMessage(w, http.StatusNotFound, fmt.Sprintf("No content found for concept with uuid %s.", conceptUUID))
		return
	}

	responseJSON, err := json.Marshal(contentByConcept)
	if err != nil {
		writeHTTPMessage(w, http.StatusInternalServerError, fmt.Sprintf(`Error parsing content for concept with uuid %s, err=%s`, conceptUUID, err.Error()))
		return
	}

	w.Header().Set("Cache-Control", hh.cacheControlHeader)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func getDateParam(req *http.Request, name string) (int64, bool, error) {
	dateParam, found, err := getSingleValueQueryParameter(req, name)
	if err != nil || !found {
		return 0, found, err
	}

	return convertStringToDateTimeEpoch(dateParam), found, nil
}

func getSingleValueQueryParameter(req *http.Request, param string) (string, bool, error) {
	query := req.URL.Query()
	values, found := query[param]
	if len(values) > 1 {
		return "", found, fmt.Errorf("specified multiple %v query parameters in the URL", param)
	}

	if len(values) < 1 {
		return "", found, nil
	}

	return values[0], found, nil
}

func writeHTTPMessage(w http.ResponseWriter, status int, message string) {
	resp := make(map[string]string)
	resp["message"] = message

	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.Encode(&resp)
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
