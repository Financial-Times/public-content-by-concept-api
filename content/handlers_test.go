package content

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/stretchr/testify/assert"
)

const (
	testConceptID    = "44129750-7616-11e8-b45a-da24cd01f044"
	testContentUUID  = "e89db5e2-760d-11e8-b45a-da24cd01f044"
	anotherConceptID = "347e2eca-7860-11e8-b45a-da24cd01f044"
)

func TestContentByConceptHandler_GetContentByConcept(t *testing.T) {
	logger.InitDefaultLogger("test-handlers")
	assert := assert.New(t)

	tests := []struct {
		testName           string
		conceptID          string
		contentList        []string
		fromDate           string
		toDate             string
		page               string
		contentLimit       string
		expectedStatusCode int
		expectedBody       string
		backendError       error
	}{
		{
			testName:           "Success for request with full URL",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			page:               "5",
			contentLimit:       "10",
			expectedStatusCode: 200,
		},
		{
			testName:           "Success for request with no page",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			contentLimit:       "10",
			expectedStatusCode: 200,
		},
		{
			testName:           "Success for request with no content limit",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			expectedStatusCode: 200,
		},
		{
			testName:           "Success for request with no content limit, page, fromDate or toDate",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 200,
		},
		{
			testName:           "Bad Request: no isAnnotatedBy parameter",
			conceptID:          "",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`,
		},
		{
			testName:           "Success: isAnnotatedBy has valid UUID",
			conceptID:          anotherConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 200,
			expectedBody:       "",
		},
		{
			testName:           "Bad Request: isAnnotatedBy param has no URI/UUID",
			conceptID:          "NullURI",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "Missing concept URI."}`,
		},
		{
			testName:           "Bad Request: isAnnotatedBy URI has invalid UUID",
			conceptID:          "123456",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "ID extracted from request URL was not valid uuid"}`,
		},
		{
			testName:           "Bad Request: query param 'page' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			page:               "null",
			expectedStatusCode: 400,
			expectedBody:       "{\"message\": provided value for page, null, could not be parsed.}",
		},
		{
			testName:           "Bad Request: query param 'page' is less than defaultPage value",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			page:               "0",
			expectedStatusCode: 400,
		},
		{
			testName:           "Bad Request: query param 'limit' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			contentLimit:       "null",
			expectedStatusCode: 200,
			expectedBody:       "",
		},
		{
			testName:           "Bad Request: query param 'fromDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "null",
			expectedStatusCode: 400,
			expectedBody:       "{\"message\": From date value null could not be parsed}",
		},
		{
			testName:           "Bad Request: query param 'toDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			toDate:             "null",
			expectedStatusCode: 400,
			expectedBody:       "{\"message\": To date value null could not be parsed}",
		},
		{
			testName:           "Backend Error returns 503",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 503,
			expectedBody:       "{\"message\": Backend error returning content for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044}",
			backendError:       errors.New("there was a problem"),
		},
		{
			testName:           "No content for concept returns 404",
			conceptID:          testConceptID,
			expectedStatusCode: 404,
			expectedBody:       "{\"message\": No content found for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044}",
		},
	}

	for _, test := range tests {
		var reqURL string
		ds := dummyService{test.contentList, test.backendError}
		handler := ContentByConceptHandler{&ds, "10", regexp.MustCompile(uuidRegex)}

		rec := httptest.NewRecorder()
		if test.conceptID == "" {
			reqURL = "/content"
		} else if test.conceptID == "NullURI" {
			reqURL = "/content?isAnnotatedBy="
		} else if test.conceptID == anotherConceptID {
			reqURL = "/content?isAnnotatedBy=" + anotherConceptID
		} else {
			reqURL = buildURL(test.conceptID, test.fromDate, test.toDate, test.page, test.contentLimit)
		}
		handler.GetContentByConcept(rec, newRequest("GET", reqURL))
		assert.Equal(test.expectedStatusCode, rec.Code, "There was an error returning the correct status code")
		if test.expectedBody != "" {
			assert.Equal(test.expectedBody, rec.Body.String(), "Wrong body")
		}
	}
}

func buildURL(conceptID, fromDate, toDate, page, contentLimit string) string {
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

func (dS dummyService) GetContentForConcept(conceptUUID string, params RequestParams) (contentList, bool, error) {
	cntList := contentList{}
	for _, contentID := range dS.contentIDList {
		var con = content{}
		con.APIURL = mapper.APIURL(contentID, []string{"Content", "Thing"}, "")
		con.ID = wwwThingsPrefix + contentID
		cntList = append(cntList, con)
	}

	if len(cntList) > 0 {
		return cntList, true, dS.backendErr
	}
	return cntList, false, dS.backendErr
}

func (dS dummyService) CheckConnection() (string, error) {
	return "", nil
}
