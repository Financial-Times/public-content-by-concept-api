package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"regexp"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const thingURIPrefix = "http://api.ft.com/things/"

type Handler struct {
	ContentService     ContentByConceptServicer
	CacheControlHeader string
	UUIDMatcher        *regexp.Regexp
}

func (h *Handler) RegisterHandlers(router *mux.Router) {
	logger.Info("registering handlers")
	gh := handlers.MethodHandler{
		"GET": http.HandlerFunc(h.GetContentByConcept),
	}
	router.Handle("/content", gh)
}

func (h *Handler) GetContentByConcept(w http.ResponseWriter, r *http.Request) {

	transID := transactionidutils.GetTransactionIDFromRequest(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("X-Request-Id", transID)
	logger.WithTransactionID(transID).Debugf("request url is %s", r.URL.RawQuery)

	q := r.URL.Query()

	conceptURI := q.Get("isAnnotatedBy")
	if conceptURI == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`))
		return
	}

	conceptUUID := strings.TrimPrefix(conceptURI, thingURIPrefix)
	if !h.UUIDMatcher.MatchString(conceptUUID) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "ID extracted from request URL was not valid uuid"}`))
		return
	}

	var showImplicit bool
	showImplicitParam := q.Get("showImplicit")
	if showImplicitParam != "" {
		if b, err := strconv.ParseBool(showImplicitParam); err == nil {
			showImplicit = b
		}
	}

	limitParam := q.Get("limit")
	var contentLimit int

	if limitParam == "" {
		logger.WithTransactionID(transID).WithUUID(conceptUUID).Debugf("No contentLimit provided. Using default: %v", defaultLimit)
		contentLimit = defaultLimit
	} else {
		if _, err := strconv.Atoi(limitParam); err != nil {
			logger.Debugf("provided value for contentLimit, %s, could not be parsed. Using default: %d", limitParam, defaultLimit)
			contentLimit = defaultLimit
		}
	}

	fromDateParam := q.Get("fromDate")
	toDateParam := q.Get("toDate")
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
		contentLimit:  contentLimit,
		fromDateEpoch: fromDateEpoch,
		toDateEpoch:   toDateEpoch,
		showImplicit:  showImplicit,
	}

	contentList, found, err := h.ContentService.GetContentForConcept(conceptUUID, requestParams)
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

	w.Header().Set("Cache-Control", h.CacheControlHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(contentList); err != nil {
		msg := fmt.Sprintf("Error parsing returned content list for concept with uuid %s", conceptUUID)
		logger.WithError(err).WithTransactionID(transID).WithUUID(conceptUUID).Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": ` + msg + `}`))
		return
	}
}
