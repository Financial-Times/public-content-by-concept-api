package content

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	testConceptID    = "44129750-7616-11e8-b45a-da24cd01f044"
	testContentUUID  = "e89db5e2-760d-11e8-b45a-da24cd01f044"
	anotherConceptID = "347e2eca-7860-11e8-b45a-da24cd01f044"
	uuidRegex        = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"
)

func TestContentByConceptHandler_GetContentByConcept(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		testName           string
		conceptID          string
		contentList        []string
		fromDate           string
		toDate             string
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
			contentLimit:       "10",
			expectedStatusCode: 200,
			expectedBody:       "",
			backendError:       nil,
		},
		{
			testName:           "Success for request with no content limit",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "2018-01-01",
			toDate:             "2018-06-20",
			expectedStatusCode: 200,
			expectedBody:       "",
			backendError:       nil,
		},
		{
			testName:           "Success for request with no content limit, fromDate or toDate",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 200,
			expectedBody:       "",
			backendError:       nil,
		},
		{
			testName:           "Bad Request: no isAnnotatedBy parameter",
			conceptID:          "",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`,
			backendError:       nil,
		},
		{
			testName:           "Success: isAnnotatedBy has valid UUID",
			conceptID:          anotherConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 200,
			expectedBody:       "",
			backendError:       nil,
		},
		{
			testName:           "Bad Request: isAnnotatedBy param has no URI/UUID",
			conceptID:          "NullURI",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "Missing or empty query parameter isAnnotatedBy. Expecting valid absolute concept URI."}`,
			backendError:       nil,
		},
		{
			testName:           "Bad Request: isAnnotatedBy URI has invalid UUID",
			conceptID:          "123456",
			contentList:        []string{testContentUUID},
			expectedStatusCode: 400,
			expectedBody:       `{"message": "ID extracted from request URL was not valid uuid"}`,
			backendError:       nil,
		},
		{
			testName:           "Bad Request: query param 'limit' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			contentLimit:       "null",
			expectedStatusCode: 200,
			expectedBody:       "",
			backendError:       nil,
		},
		{
			testName:           "Bad Request: query param 'fromDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			fromDate:           "null",
			expectedStatusCode: 400,
			expectedBody:       "{\"message\": From date value null could not be parsed}",
			backendError:       nil,
		},
		{
			testName:           "Bad Request: query param 'toDate' is invalid",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			toDate:             "null",
			expectedStatusCode: 400,
			expectedBody:       "{\"message\": To date value null could not be parsed}",
			backendError:       nil,
		},
		{
			testName:           "Backend Error returns 503",
			conceptID:          testConceptID,
			contentList:        []string{testContentUUID},
			expectedStatusCode: 503,
			expectedBody:       "{\"message\": Backend error returning content for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044}",
			backendError:       errors.New("There was a problem"),
		},
		{
			testName:           "No content for concept returns 404",
			conceptID:          testConceptID,
			expectedStatusCode: 404,
			expectedBody:       "{\"message\": No content found for concept with uuid 44129750-7616-11e8-b45a-da24cd01f044}",
			backendError:       nil,
		},
	}

	for _, test := range tests {
		var reqURL string
		ds := dummyService{test.contentList, test.backendError}
		handler := Handler{&ds, "10", regexp.MustCompile(uuidRegex)}
		router := mux.NewRouter()
		handler.RegisterHandlers(router)
		rec := httptest.NewRecorder()
		if test.conceptID == "" {
			reqURL = "/content"
		} else if test.conceptID == "NullURI" {
			reqURL = "/content?isAnnotatedBy="
		} else if test.conceptID == anotherConceptID {
			reqURL = "/content?isAnnotatedBy=" + anotherConceptID
		} else {
			reqURL = buildURL(test.conceptID, test.fromDate, test.toDate, test.contentLimit)
		}
		router.ServeHTTP(rec, newRequest("GET", reqURL))
		assert.Equal(test.expectedStatusCode, rec.Code, "THere was an error returning the correct status code")
		if test.expectedBody != "" {
			assert.Equal(test.expectedBody, rec.Body.String(), "Wrong body")
		}
	}
}

func buildURL(conceptID, fromDate, toDate, contentLimit string) string {
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

func (dS dummyService) Check() error {
	return nil
}
