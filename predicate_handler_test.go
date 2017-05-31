package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetWithPredicateHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []test{
		{"Success", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"SuccessWithLimit", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=http://api.ft.com/things/%s&limit=2&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"SuccessWithToDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&toDate=2006-01-02&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"SuccessWithFromDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&fromDate=2006-01-02&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"SuccessDespiteInvalidFromDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&fromDate=0&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"SuccessDespiteInvalidToDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&toDate=0&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"NotFound", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", "99999"), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusNotFound, "", message("No content found for concept with uuid 99999.")},
		{"ReadError", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusServiceUnavailable, "", message("Error getting content for concept with uuid 12345, err=TEST failing to READ")},
		{"LimitShouldBeNumber", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=http://api.ft.com/things/%s&limit=huh&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message("Error limit is not a number: huh.")},
		{"ExactlyOneLimit", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=http://api.ft.com/things/%s&limit=10&limit=50&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`Please provide one value for \"limit\".`)},
		{"MissingAnnotatedBy", newRequest("GET", "/content?withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`Missing query parameter \"isAnnotatedBy\". Expecting exactly one valid absolute Concept URI.`)},
		{"BlankAnnotatedBy", newRequest("GET", "/content?isAnnotatedBy=&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`No value specified for Concept URI.`)},
		{"MoreThanOneAnnotatedBy", newRequest("GET", "/content?isAnnotatedBy=blah&isAnnotatedBy=blah-again&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor", "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`More than one value found for query parameter \"isAnnotatedBy\". Expecting exactly one valid absolute Concept URI.`)},
		{"MoreThanOneToDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&toDate=0&toDate=1&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`More than one value for \"toDate\" supplied. Please provide exactly one value.`)},
		{"MoreThanOneFromDate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&fromDate=0&fromDate=1&withPredicate=http://www.ft.com/ontology/annotation/hasAuthor`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`More than one value for \"fromDate\" supplied. Please provide exactly one value.`)},
		{"UnsupportedPredicate", newRequest("GET", fmt.Sprintf(`/content?isAnnotatedBy=%s&withPredicate=http://www.ft.com/ontology/annotation/hasSomethingElse`, knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusNotImplemented, "", message(`Provided \"withPredicate\" value is currently unsupported.`)},
		{"TooManyPredicates", newRequest("GET", `/content?withPredicate=http://www.ft.com/ontology/annotation/hasSomething&withPredicate=http://www.ft.com/ontology/annotation/hasSomethingElse`, "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusBadRequest, "", message(`More than one value found for query parameter \"withPredicate\". Expecting exactly one valid absolute predicate URI.`)},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(httpHandlers{test.dummyService, "max-age=360, public"}).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.JSONEq(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}
