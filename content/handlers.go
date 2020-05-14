package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

const (
	defaultPage    = 1
	thingURIPrefix = "http://api.ft.com/things/"
	dateTimeLayout = "2006-01-02"
)

type dbService interface {
	GetContentForConcept(conceptUUID string, params RequestParams) ([]Content, error)
}

type Handler struct {
	ContentService     dbService
	CacheControlHeader string
}

func (h *Handler) GetContentByConcept(w http.ResponseWriter, r *http.Request) {
	transID := transactionidutils.GetTransactionIDFromRequest(r)

	m, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		logger.WithError(err).WithTransactionID(transID).Error("Could not parse request url")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set(transactionidutils.TransactionIDHeader, transID)
	logger.WithTransactionID(transID).Infof("request url is %s", r.URL.RawQuery)

	conceptURI := m.Get("isAnnotatedBy")
	if conceptURI == "" {
		writeJSONMessage(w, http.StatusBadRequest, "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI.")
		return
	}

	conceptUUID := strings.TrimPrefix(conceptURI, thingURIPrefix)
	if !UUIDRegex.MatchString(conceptUUID) {
		writeJSONMessage(w, http.StatusBadRequest, fmt.Sprintf("%s extracted from request URL was not valid uuid", conceptUUID))
		return
	}

	page := defaultPage
	pageParam := m.Get("page")
	if pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			msg := fmt.Sprintf("provided value for page, %s, could not be parsed.", pageParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			writeJSONMessage(w, http.StatusBadRequest, msg)
			return
		}

		if page < defaultPage {
			msg := fmt.Sprintf("provided value for page should be greater than: %v", defaultPage)
			logger.WithTransactionID(transID).WithUUID(conceptUUID).Debugf(msg)
			writeJSONMessage(w, http.StatusBadRequest, msg)
			return
		}
	}

	limitParam := m.Get("limit")
	var contentLimit int

	if limitParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debugf("No contentLimit provided. Using default: %d", defaultLimit)
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
		fromDateTime, err := time.Parse(dateTimeLayout, fromDateParam)
		if err != nil {
			msg := fmt.Sprintf("From date value %s could not be parsed", fromDateParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			writeJSONMessage(w, http.StatusBadRequest, msg)
			return
		}
		fromDateEpoch = fromDateTime.Unix()
	}

	if toDateParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debug("no toDate url param supplied")
	} else {
		toDateTime, err := time.Parse(dateTimeLayout, toDateParam)
		if err != nil {
			msg := fmt.Sprintf("To date value %s could not be parsed", toDateParam)
			logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
			writeJSONMessage(w, http.StatusBadRequest, msg)
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

	contentList, err := h.ContentService.GetContentForConcept(conceptUUID, requestParams)
	if err != nil {

		if err == ErrContentNotFound {
			msg := fmt.Sprintf("No content found for concept with uuid %s", conceptUUID)
			logger.WithTransactionID(transID).WithUUID(conceptUUID).Info(msg)
			writeJSONMessage(w, http.StatusNotFound, msg)
			return
		}

		msg := fmt.Sprintf("Backend error returning content for concept with uuid %s", conceptUUID)
		logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
		writeJSONMessage(w, http.StatusServiceUnavailable, msg)
		return
	}

	w.Header().Set("Cache-Control", h.CacheControlHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(contentList); err != nil {
		msg := fmt.Sprintf("Error parsing returned content list for concept with uuid %s", conceptUUID)
		logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
		writeJSONMessage(w, http.StatusInternalServerError, msg)
		return
	}
}

func writeJSONMessage(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"message": "` + msg + `"}`))
}
