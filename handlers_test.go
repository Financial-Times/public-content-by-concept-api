package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const knownUUID = "12345"

type test struct {
	name         string
	req          *http.Request
	dummyService driver
	statusCode   int
	contentType  string // Contents of the Content-Type header
	body         string
}

func TestGetHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []test{
		{"Success", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s", knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusOK, "", "[]"},
		{"NotFound", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s", "99999"), "application/json", nil), dummyService{contentUUID: knownUUID}, http.StatusNotFound, "", message("No content found for concept with uuid 99999.")},
		{"ReadError", newRequest("GET", fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s", knownUUID), "application/json", nil), dummyService{contentUUID: knownUUID, failRead: true}, http.StatusServiceUnavailable, "", message("Error getting content for concept with uuid 12345, err=TEST failing to READ")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(httpHandlers{test.dummyService, "max-age=360, public"}).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.JSONEq(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func newRequest(method, url, contentType string, body []byte) *http.Request {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", contentType)
	return req
}

func message(errMsg string) string {
	return fmt.Sprintf("{\"message\": \"%s\"}\n", errMsg)
}

type dummyService struct {
	contentUUID string
	failRead    bool
}

func (dS dummyService) read(contentUUID string) (ContentList, bool, error) {
	if dS.failRead {
		return nil, false, errors.New("TEST failing to READ")
	}
	if contentUUID == dS.contentUUID {
		return ContentList{}, true, nil
	}
	return nil, false, nil
}

func (dS dummyService) checkConnectivity() error {
	return nil
}
