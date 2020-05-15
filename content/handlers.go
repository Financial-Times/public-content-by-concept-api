package content

import (
	"encoding/json"
	"errors"
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

	requestParams, err := extractRequestParams(m, logger.WithTransactionID(transID).WithUUID(conceptUUID))
	if err != nil {
		writeJSONMessage(w, http.StatusBadRequest, err.Error())
		return
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

func extractRequestParams(val url.Values, log logger.LogEntry) (RequestParams, error) {

	var (
		page          = defaultPage
		contentLimit  = defaultLimit
		fromDateEpoch = int64(0)
		toDateEpoch   = int64(0)
		err           error
	)

	pageParam := val.Get("page")
	if pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			msg := fmt.Sprintf("provided value for page, %s, could not be parsed.", pageParam)
			log.WithError(err).Error(msg)
			return RequestParams{}, errors.New(msg)
		}

		if page < defaultPage {
			msg := fmt.Sprintf("provided value for page should be greater than: %v", defaultPage)
			log.Debugf(msg)
			return RequestParams{}, errors.New(msg)
		}
	}

	limitParam := val.Get("limit")

	if limitParam == "" {
		log.Debugf("No contentLimit provided. Using default: %d", defaultLimit)
	} else {
		limit, err := strconv.Atoi(limitParam)
		if err != nil {
			log.Debugf("provided value for contentLimit, %s, could not be parsed. Using default: %d", limitParam, defaultLimit)
		} else {
			contentLimit = limit
		}
	}

	fromDateParam := val.Get("fromDate")
	toDateParam := val.Get("toDate")

	if fromDateParam == "" {
		log.Debug("no fromDate url param supplied")
	} else {
		fromDateTime, err := time.Parse(dateTimeLayout, fromDateParam)
		if err != nil {
			msg := fmt.Sprintf("From date value %s could not be parsed", fromDateParam)
			log.WithError(err).Error(msg)
			return RequestParams{}, errors.New(msg)
		}
		fromDateEpoch = fromDateTime.Unix()
	}

	if toDateParam == "" {
		log.Debug("no toDate url param supplied")
	} else {
		toDateTime, err := time.Parse(dateTimeLayout, toDateParam)
		if err != nil {
			msg := fmt.Sprintf("To date value %s could not be parsed", toDateParam)
			log.WithError(err).Error(msg)
			return RequestParams{}, errors.New(msg)
		}
		toDateEpoch = toDateTime.Unix()
	}

	return RequestParams{
		page:          page,
		contentLimit:  contentLimit,
		fromDateEpoch: fromDateEpoch,
		toDateEpoch:   toDateEpoch,
	}, nil
}

func writeJSONMessage(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"message": "` + msg + `"}`))
}
