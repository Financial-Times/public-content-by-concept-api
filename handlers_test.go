package main

import (
	"errors"
	"fmt"
	"github.com/Financial-Times/public-content-by-concept-api/v2/policy"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/public-content-by-concept-api/v2/content"
	"github.com/stretchr/testify/assert"
)

const (
	testConceptID    = "44129750-7616-11e8-b45a-da24cd01f044"
	testContentUUID  = "e89db5e2-760d-11e8-b45a-da24cd01f044"
	anotherConceptID = "347e2eca-7860-11e8-b45a-da24cd01f044"
)

var (
	isAuthorized           = policy.Result{IsAuthorizedForPublication: true, AddFilterByPublication: false, Publications: []string{}}
	isNotAuthorized        = policy.Result{IsAuthorizedForPublication: false, AddFilterByPublication: false, Publications: []string{}}
	addFilterByPublication = policy.Result{IsAuthorizedForPublication: false, AddFilterByPublication: true, Publications: []string{"88fdde6c-2aa4-4f78-af02-9f680097cfd6", "8e6c705e-1132-42a2-8db0-c295e29e8658"}}
)

func TestContentByConceptHandler_GetContentByConcept(t *testing.T) {
	log := logger.NewUPPLogger("test-service", "info")

	assert := assert.New(t)

	tests := []struct {
		testName           string
		conceptID          string
		contentList        []string
		fromDate           string
		toDate             string
		page               string
		contentLimit       string
		publication        []string
		expectedStatusCode int
		expectedBody       string
		backendError       error
		opaPolicyResult    policy.Result
	}{
		{
			testName:           "Success for request with full URL",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			page:               "5",
			contentLimit:       "10",
			publication:        []string{"88fdde6c-2aa4-4f78-af02-9f680097cfd6", "8e6c705e-1132-42a2-8db0-c295e29e8658"},
			expectedStatusCode: http.StatusOK,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Success for request with filter added by the middleware",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			page:               "5",
			contentLimit:       "10",
			expectedStatusCode: http.StatusOK,
			opaPolicyResult:    addFilterByPublication,
		},
		{
			testName:           "Forbidden by policy resultg",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			page:               "5",
			contentLimit:       "10",
			publication:        []string{"88fdde6c-2aa4-4f78-af02-9f680097cfd6", "8e6c705e-1132-42a2-8db0-c295e29e8658"},
			expectedStatusCode: http.StatusForbidden,
			opaPolicyResult:    isNotAuthorized,
		},
		{
			testName:           "Success for request with no page",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			contentLimit:       "10",
			expectedStatusCode: http.StatusOK,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Success for request with no content limit",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			expectedStatusCode: http.StatusOK,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Success for request with no content limit, page, fromDate or toDate",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusOK,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: no isAnnotatedBy parameter",
			conceptID:          "",
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Success: isAnnotatedBy has valid UUID",
			conceptID:          anotherConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusOK,
			expectedBody:       "",
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: isAnnotatedBy param has no URI/UUID",
			conceptID:          "NullURI",
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: isAnnotatedBy URI has invalid UUID",
			conceptID:          "123456",
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "123456 extracted from request URL was not valid uuid"}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'page' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			page:               "null",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "provided value for page, null, could not be parsed."}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'page' is less than defaultPage value",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			page:               "0",
			expectedStatusCode: http.StatusBadRequest,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'limit' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			contentLimit:       "null",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "",
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'fromDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "null",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "From date value null could not be parsed"}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'toDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			toDate:             "null",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "To date value null could not be parsed"}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Backend Error returns 503",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectedBody:       `{"message": "Backend error returning content for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044"}`,
			backendError:       errors.New("there was a problem"),
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "No content for concept returns 404",
			conceptID:          testConceptID,
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       `{"message": "No content found for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044"}`,
			opaPolicyResult:    isAuthorized,
		},
		{
			testName:           "Bad Request: query param 'publication' has invalid uuid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			publication:        []string{"88fdde6c-2aa4-4f78-af02-9f680097cfd"},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "Publication array param contains value 88fdde6c-2aa4-4f78-af02-9f680097cfd which is not valid uuid"}`,
			opaPolicyResult:    isAuthorized,
		},
	}

	for _, test := range tests {
		var reqURL string
		ds := dummyService{test.contentList, test.backendError}
		handler := Handler{ContentService: &ds, CacheControlHeader: "10", Log: log}

		rec := httptest.NewRecorder()

		r := mux.NewRouter()
		r.HandleFunc("/content", handler.GetContentByConcept).Methods("GET")

		if test.conceptID == "" {
			reqURL = "/content"
		} else if test.conceptID == "NullURI" {
			reqURL = "/content?isAnnotatedBy="
		} else if test.conceptID == anotherConceptID {
			reqURL = "/content?isAnnotatedBy=" + anotherConceptID
		} else {
			reqURL = buildURL(test.conceptID, test.fromDate, test.toDate, test.page, test.contentLimit, test.publication)
		}

		policy.IsAuthorizedPublication(r, rec, newRequest("GET", reqURL), log, test.opaPolicyResult)
		assert.Equal(test.expectedStatusCode, rec.Code, "There was an error returning the correct status code")
		if test.expectedBody != "" {
			assert.Equal(test.expectedBody, rec.Body.String(), "Wrong body")
		}
	}
}

func TestContentByConceptHandler_GetContentByConceptImplicitly(t *testing.T) {
	log := logger.NewUPPLogger("test-service", "info")

	assert := assert.New(t)

	tests := []struct {
		testName           string
		conceptID          string
		contentList        []string
		expectedStatusCode int
		expectedBody       string
		backendError       error
	}{
		{
			testName:           "Successful request",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusOK,
		},
		{
			testName:           "Bad Request: conceptUUID param has invalid URI/UUID",
			conceptID:          "NullURI",
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message": "NullURI extracted from request URL was not valid uuid"}`,
		},
		{
			testName:           "Backend Error returns 503",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectedBody:       `{"message": "Backend error returning content for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044"}`,
			backendError:       errors.New("there was a problem"),
		},
		{
			testName:           "No content for concept returns 404",
			conceptID:          testConceptID,
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       `{"message": "No content found for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044"}`,
		},
	}

	for _, test := range tests {
		ds := dummyService{test.contentList, test.backendError}
		handler := Handler{ContentService: &ds, CacheControlHeader: "10", Log: log}

		rec := httptest.NewRecorder()
		r := mux.NewRouter()
		r.HandleFunc("/content/{conceptUUID}/implicitly", handler.GetContentByConceptImplicitly).Methods("GET")
		r.ServeHTTP(rec, newRequest("GET", fmt.Sprintf("/content/%s/implicitly", test.conceptID)))
		assert.Equal(test.expectedStatusCode, rec.Code, "There was an error returning the correct status code")
		if test.expectedBody != "" {
			assert.Equal(test.expectedBody, rec.Body.String(), "Wrong body")
		}
	}
}

func buildURL(conceptID, fromDate, toDate, page, contentLimit string, publication []string) string {
	var URL = fmt.Sprintf("/content?isAnnotatedBy=http://api.ft.com/things/%s", conceptID)
	if fromDate != "" {
		URL = URL + fmt.Sprintf("&fromDate=%s", fromDate)
	}
	if toDate != "" {
		URL = URL + fmt.Sprintf("&toDate=%s", toDate)
	}
	if contentLimit != "" {
		URL = URL + fmt.Sprintf("&limit=%s", contentLimit)
	}
	if page != "" {
		URL = URL + fmt.Sprintf("&page=%s", page)
	}
	if len(publication) != 0 {
		URL = URL + fmt.Sprintf("&publication=%s", strings.Join(publication, ","))
	}
	return URL
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

type dummyService struct {
	contentIDList []string
	backendErr    error
}

func (dS dummyService) GetContentForConcept(conceptUUID string, params content.RequestParams) ([]content.Content, error) {
	if dS.backendErr != nil {
		return nil, dS.backendErr
	}
	if len(dS.contentIDList) == 0 && dS.backendErr == nil {
		return nil, content.ErrContentNotFound
	}

	cntList := make([]content.Content, 0)
	for _, contentID := range dS.contentIDList {
		var con = content.Content{}
		con.APIURL = apiURL(contentID)
		con.ID = idURL(contentID)
		cntList = append(cntList, con)
	}

	return cntList, nil
}

func (dS dummyService) GetContentForConceptImplicitly(conceptUUID string) ([]content.Content, error) {
	if dS.backendErr != nil {
		return nil, dS.backendErr
	}
	if len(dS.contentIDList) == 0 && dS.backendErr == nil {
		return nil, content.ErrContentNotFound
	}

	cntList := make([]content.Content, 0)
	for _, contentID := range dS.contentIDList {
		var con = content.Content{}
		con.APIURL = apiURL(contentID)
		con.ID = idURL(contentID)
		cntList = append(cntList, con)
	}

	return cntList, nil
}

func (dS dummyService) CheckConnection() (string, error) {
	return "", nil
}

func apiURL(uuid string) string {
	return "http://api.ft.com/content/" + uuid
}

func idURL(uuid string) string {
	return "http://www.ft.com/content/" + uuid
}
