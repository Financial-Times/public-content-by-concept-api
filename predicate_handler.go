package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/coreos/fleet/log"
)

var predicateToLabelMap = map[string]string{
	"http://www.ft.com/ontology/annotation/hasAuthor": "HAS_AUTHOR",
}

func (hh *httpHandlers) getContentByConceptWithPredicate(w http.ResponseWriter, r *http.Request, predicate string) {
	label, ok := predicateToLabelMap[predicate]
	if !ok {
		writeHTTPMessage(w, http.StatusNotImplemented, `Provided "withPredicate" value is currently unsupported.`)
		return
	}

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

	contentByConcept, found, err := hh.contentDriver.readWithPredicate(conceptUUID, label, limit, fromDateEpoch, toDateEpoch)
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
