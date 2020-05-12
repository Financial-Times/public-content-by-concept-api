package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"regexp"

	"github.com/Financial-Times/go-logger"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

const (
	defaultPage    = 1
	thingURIPrefix = "http://api.ft.com/things/"
)

type dbService interface {
	CheckConnection() (string, error)
	GetContentForConcept(conceptUUID string, params RequestParams) (contentList, bool, error)
}
type ContentByConceptHandler struct {
	ContentService     dbService
	CacheControlHeader string
	UUIDMatcher        *regexp.Regexp
}

func (ch *ContentByConceptHandler) GetContentByConcept(w http.ResponseWriter, r *http.Request) {
	transID := transactionidutils.GetTransactionIDFromRequest(r)
	m, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		logger.WithError(err).WithTransactionID(transID).Error("Could not parse request url")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("X-Request-Id", transID)
	logger.WithTransactionID(transID).Infof("request url is %s", r.URL.RawQuery)

	_, isAnnotatedByPresent := m["isAnnotatedBy"]
	if !isAnnotatedByPresent {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`))
		return
	}

	conceptURI := m["isAnnotatedBy"][0]
	if conceptURI == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Missing concept URI."}`))
		return
	}

	conceptUUID := strings.TrimPrefix(conceptURI, thingURIPrefix)
	if !ch.UUIDMatcher.MatchString(conceptUUID) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "ID extracted from request URL was not valid uuid"}`))
		return
	}

	page := defaultPage
	pageParam := m.Get("page")
	if pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			msg := fmt.Sprintf("provided value for page, %s, could not be parsed.", pageParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": ` + msg + `}`))
			return
		}

		if page < defaultPage {
			msg := fmt.Sprintf("provided value for page should be greater than: %v", defaultPage)
			logger.WithTransactionID(transID).WithUUID(conceptUUID).Debugf(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": ` + msg + `}`))
			return
		}
	}

	limitParam := m.Get("limit")
	var contentLimit int

	if limitParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debugf("No contentLimit provided. Using default: %v", defaultLimit)
		contentLimit = defaultLimit
	} else {
		contentLimit, err = strconv.Atoi(limitParam)
		if err != nil {
			logger.Debugf("provided value for contentLimit, %s, could not be parsed. Using default: %d", limitParam, defaultLimit)
			contentLimit = defaultLimit
		}
	}

	fromDateParam := m.Get("fromDate")
	toDateParam := m.Get("toDate")
	var fromDateEpoch, toDateEpoch int64

	if fromDateParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debug("no fromDate url param supplied")
	} else {
		fromDateTime, err := time.Parse("2006-01-02", fromDateParam)
		if err != nil {
			msg := fmt.Sprintf("From date value %s could not be parsed", fromDateParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": ` + msg + `}`))
			return
		}
		fromDateEpoch = fromDateTime.Unix()
	}

	if toDateParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debug("no toDate url param supplied")
	} else {
		toDateTime, err := time.Parse("2006-01-02", toDateParam)
		if err != nil {
			msg := fmt.Sprintf("To date value %s could not be parsed", toDateParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": ` + msg + `}`))
			return
		}
		toDateEpoch = toDateTime.Unix()
	}

	requestParams := RequestParams{
		page:          page,
		contentLimit:  contentLimit,
		fromDateEpoch: fromDateEpoch,
		toDateEpoch:   toDateEpoch,
	}

	contentList, found, err := ch.ContentService.GetContentForConcept(conceptUUID, requestParams)
	if err != nil {
		msg := fmt.Sprintf("Backend error returning content for concept with uuid %s", conceptUUID)
		logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"message": ` + msg + `}`))
		return
	}
	if !found {
		msg := fmt.Sprintf("No content found for concept with uuid %s", conceptUUID)
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Info(msg)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": ` + msg + `}`))
		return
	}

	w.Header().Set("Cache-Control", ch.CacheControlHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(contentList); err != nil {
		msg := fmt.Sprintf("Error parsing returned content list for concept with uuid %s", conceptUUID)
		logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": ` + msg + `}`))
		return
	}
}
