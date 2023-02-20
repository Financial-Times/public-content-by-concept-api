//go:build integration
// +build integration

package content

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	annrw "github.com/Financial-Times/annotations-rw-neo4j/v4/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/v2/baseftrwapp"
	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
	"github.com/Financial-Times/concepts-rw-neo4j/concepts"
	cnt "github.com/Financial-Times/content-rw-neo4j/v3/content"
	"github.com/Financial-Times/go-logger/v2"
)

const (
	// Generate uuids so there's no clash with real data
	contentUUID             = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	content2UUID            = "bfa97890-76ff-4a35-a775-b8768f7ea383"
	content3UUID            = "5a9c7429-e76b-4f37-b5d1-842d64a45167"
	content4UUID            = "8e193b84-4697-41aa-a480-065831d1d964"
	content5UUID            = "8a08dfe3-88c4-47dd-bee6-846ede810448"
	content6UUID            = "27c47a08-6bad-486d-8e06-ce24d583ae2a"
	content7UUID            = "df7e4deb-e048-43d7-9441-f7d152075a91"
	content8UUID            = "4e6a0098-94a9-45c1-835c-7572e1fcc567"
	MSJConceptUUID          = "5d1510f8-2779-4b74-adab-0a5eb138fca6"
	FakebookConceptUUID     = "eac853f5-3859-4c08-8540-55e043719400"
	MetalMickeyConceptUUID  = "0483bef8-5797-40b8-9b25-b12e492f63c6"
	OnyxPikeBrandUUID       = "9a07c16f-def0-457d-a04a-57ba68ba1e00"
	OnyxPikeParentBrandUUID = "0635a44c-2e9e-49b6-b078-be53b0e5301b"
	OnyPikeyRightBrandUUID  = "4c4738cb-45df-43fe-ac7c-bab963b698ea"
	JohnSmithFSUUID         = "bf3c4c55-4ff6-4439-a36c-3a513f563374"
	JohnSmithSmartlogicUUID = "d46c09ce-7861-11e8-b45a-da24cd01f044"
	JohnSmithTMEUUID        = "3af8b4e4-7862-11e8-b45a-da24cd01f044"
	JohnSmithOtherTMEUUID   = "521a2338-2cc7-47dd-8da2-e757b4ceb7ef"
	topic1UUID              = "18e24d65-c8e6-4e23-ab19-206e0d463205"
	topic2UUID              = "64ba2208-0c0d-43e2-a883-beecb55c0d33"
	topic3UUID              = "2e7429bd-7a84-41cb-a619-2c702893e359"
	brand1UUID              = "5c7592a8-1f0c-11e4-b0cb-b2227cce2b54"

	apigURL = "http://api.ft.com"
)

const defaultLimit = 10
const defaultPage = 1

// cmneo4j.Driver is safe to use in different go routines, so it's not that problematic that it is used as global var.
var driver *cmneo4j.Driver

func init() {
	log := logger.NewUPPLogger("test-service", "info")
	driver, _ = cmneo4j.NewDefaultDriver(neoURL(), log)
	if driver == nil {
		log.Fatal("Cannot connect to Neo4J with cmneo4j driver")
	}
}

func neoURL() string {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "bolt://localhost:7687"
	}
	return url
}

func TestFindMatchingContentForV2Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)
	writeAnnotations(assert, driver, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")
	writeConcept(assert, driver, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV1Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)
	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, driver, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	contentList, err := contentByConceptDriver.GetContentForConcept(MetalMickeyConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV2AnnotationWithLimit(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)
	writeContent(assert, driver, content2UUID)
	writeAnnotations(assert, driver, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")
	writeConcept(assert, driver, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, content2UUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, 1, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestRetrieveNoContentForV1AnnotationForExclusiveDatePeriod(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)
	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, driver, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	fromDate, _ := time.Parse("2006-01-02", "2014-03-08")
	toDate, _ := time.Parse("2006-01-02", "2014-03-09")
	contentList, err := contentByConceptDriver.GetContentForConcept(MetalMickeyConceptUUID, RequestParams{0, defaultLimit, fromDate.Unix(), toDate.Unix()})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(contentList), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoContentForThatConcept(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	content, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(content), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, driver, contentUUID)
	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeAnnotations(assert, driver, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")

	defer cleanDB(t, content2UUID, MSJConceptUUID, contentUUID, MetalMickeyConceptUUID, FakebookConceptUUID)

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func TestBrandsDontReturnParentContent(t *testing.T) {
	assert := assert.New(t)
	defer cleanDB(t, content2UUID, content3UUID, content4UUID, OnyxPikeBrandUUID, OnyxPikeParentBrandUUID, OnyPikeyRightBrandUUID)

	writeContent(assert, driver, content2UUID)
	writeContent(assert, driver, content3UUID)
	writeContent(assert, driver, content4UUID)

	writeAnnotations(assert, driver, content2UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content2UUID))
	writeAnnotations(assert, driver, content3UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content3UUID))
	writeAnnotations(assert, driver, content4UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content4UUID))

	writeConcept(assert, driver, fmt.Sprintf("./fixtures/Brand-OnyxPike-%v.json", OnyxPikeBrandUUID))
	writeConcept(assert, driver, fmt.Sprintf("./fixtures/Brand-OnyxPikeParent-%v.json", OnyxPikeParentBrandUUID))

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	contentList, err := contentByConceptDriver.GetContentForConcept(OnyxPikeBrandUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", OnyxPikeBrandUUID)
	assert.Equal(2, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func TestContentIsReturnedFromAllLeafNodesOfConcordance(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, contentUUID, content2UUID, content3UUID, content4UUID, JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID)

	writeContent(assert, driver, contentUUID)
	writeContent(assert, driver, content2UUID)
	writeContent(assert, driver, content3UUID)
	writeContent(assert, driver, content4UUID)

	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, driver, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, driver, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, driver, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, driver, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)

	idsToCheck := []string{JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID}

	for _, uuid := range idsToCheck {
		contentList, err := contentByConceptDriver.GetContentForConcept(uuid, RequestParams{0, defaultLimit, 0, 0})
		assert.NoError(err, "Unexpected error for concept %s", uuid)
		assert.Equal(4, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
	}
}

func TestContentIsReturnedFromAllLeafNodesOfConcordanceWithDateRestrictions(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, contentUUID, content2UUID, content3UUID, content4UUID, JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID)

	writeContent(assert, driver, contentUUID)
	writeContent(assert, driver, content2UUID)
	writeContent(assert, driver, content3UUID)
	writeContent(assert, driver, content4UUID)

	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, driver, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, driver, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, driver, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, driver, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)

	idsToCheck := []string{JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID}

	for _, uuid := range idsToCheck {
		contentList, err := contentByConceptDriver.GetContentForConcept(uuid, RequestParams{0, defaultLimit, 1372550400, 1388448000})
		//From July 1st 2013 - January 1st 2014
		assert.NoError(err, "Unexpected error for concept %s", uuid)
		assert.Equal(1, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
	}
}

func TestContentIsReturnedFromAllLeafNodesOfConcordanceWithPagination(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, contentUUID, content2UUID, content3UUID, content4UUID, JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID)

	writeContent(assert, driver, contentUUID)
	writeContent(assert, driver, content2UUID)
	writeContent(assert, driver, content3UUID)
	writeContent(assert, driver, content4UUID)

	writeAnnotations(assert, driver, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, driver, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, driver, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, driver, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, driver, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)

	idsToCheck := []string{JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID}

	for _, uuid := range idsToCheck {
		page := defaultPage
		pageSize := 2
		allContent := make([]Content, 0)
		for {
			requestParams := RequestParams{
				Page:         page,
				ContentLimit: pageSize,
			}

			pageContents, err := contentByConceptDriver.GetContentForConcept(uuid, requestParams)
			if err == ErrContentNotFound {
				break
			}

			assert.NoError(err, "Unexpected error for concept %s", uuid)
			assert.Equal(pageSize, len(pageContents), "Didn't get the right number of page items, content=%s", pageContents)

			page++
			allContent = append(allContent, pageContents...)
		}

		assert.Equal(4, len(allContent), "Didn't get the right number of content items, content=%s", allContent)
	}
}

func TestContentIsReturnedImplicitlyForHasBroaderOrHasParentOrIsPartOfRelationship(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, content5UUID, content6UUID, topic1UUID, topic2UUID)

	writeContent(assert, driver, content5UUID)
	writeContent(assert, driver, content6UUID)

	writeAnnotations(assert, driver, content5UUID, "v2", "./fixtures/Annotations-8a08dfe3-88c4-47dd-bee6-846ede810448-V2.json")
	writeAnnotations(assert, driver, content6UUID, "v2", "./fixtures/Annotations-27c47a08-6bad-486d-8e06-ce24d583ae2a-V2.json")

	writeConcept(assert, driver, "./fixtures/Topic-18e24d65-c8e6-4e23-ab19-206e0d463205.json")
	writeConcept(assert, driver, "./fixtures/Topic-64ba2208-0c0d-43e2-a883-beecb55c0d33.json")

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)

	contentList1, err := contentByConceptDriver.GetContentForConcept(topic1UUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", topic1UUID)
	assert.Equal(1, len(contentList1), "Didn't get the right number of content items, content=%s", contentList1)

	contentList2, err := contentByConceptDriver.GetContentForConcept(topic2UUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", topic2UUID)
	assert.Equal(1, len(contentList2), "Didn't get the right number of content items, content=%s", contentList2)

	contentList3, err := contentByConceptDriver.GetContentForConceptImplicitly(topic2UUID)
	assert.NoError(err, "Unexpected error for concept %s", topic2UUID)
	assert.Equal(2, len(contentList3), "Didn't get the right number of content items, content=%s", contentList3)
}

func TestContentIsReturnedImplicitlyForImpliedByRelationship(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, content7UUID, content8UUID, brand1UUID, topic3UUID)

	writeContent(assert, driver, content7UUID)
	writeContent(assert, driver, content8UUID)

	writeAnnotations(assert, driver, content7UUID, "v2", "./fixtures/Annotations-df7e4deb-e048-43d7-9441-f7d152075a91-V2.json")
	writeAnnotations(assert, driver, content8UUID, "v2", "./fixtures/Annotations-4e6a0098-94a9-45c1-835c-7572e1fcc567-V2.json")

	writeConcept(assert, driver, "./fixtures/Brand-5c7592a8-1f0c-11e4-b0cb-b2227cce2b54.json")
	writeConcept(assert, driver, "./fixtures/Topic-2e7429bd-7a84-41cb-a619-2c702893e359.json")

	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)

	contentList1, err := contentByConceptDriver.GetContentForConcept(brand1UUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", brand1UUID)
	assert.Equal(1, len(contentList1), "Didn't get the right number of content items, content=%s", contentList1)

	contentList2, err := contentByConceptDriver.GetContentForConcept(topic3UUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", topic3UUID)
	assert.Equal(1, len(contentList2), "Didn't get the right number of content items, content=%s", contentList2)

	contentList3, err := contentByConceptDriver.GetContentForConceptImplicitly(brand1UUID)
	assert.NoError(err, "Unexpected error for concept %s", brand1UUID)
	assert.Equal(2, len(contentList3), "Didn't get the right number of content items, content=%s", contentList3)
}

func TestConceptService_Check(t *testing.T) {
	assert := assert.New(t)
	contentByConceptDriver, err := NewContentByConceptService(driver, apigURL)
	assert.NoError(err)
	_, err = contentByConceptDriver.CheckConnection()
	assert.NoError(err, "Test should always pass when connected to db")
}

func writeContent(assert *assert.Assertions, driver *cmneo4j.Driver, contentUUID string) {
	contentRW := cnt.NewContentService(driver)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-"+contentUUID+".json", assert)
}

func writeAnnotations(assert *assert.Assertions, driver *cmneo4j.Driver, contentUUID string, lifecycle string, fixtureFile string) {
	annotationsRW := annrw.NewCypherAnnotationsService(driver)
	assert.NoError(annotationsRW.Initialise())
	f, err := os.Open(fixtureFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	json, errr := annotationsRW.DecodeJSON(dec)
	assert.NoError(errr, "Error parsing file %s", fixtureFile)
	assert.NoError(annotationsRW.Write(contentUUID, lifecycle, "", "", json))
}

func writeConcept(assert *assert.Assertions, driver *cmneo4j.Driver, fixture string) {
	log := logger.NewUPPLogger("test-service", "warning")
	conceptsRW := concepts.NewConceptService(driver, log)
	assert.NoError(conceptsRW.Initialise())
	f, err := os.Open(fixture)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := conceptsRW.DecodeJSON(dec)
	assert.NoError(errr)
	_, err = conceptsRW.Write(inst, "TEST_TRANS_ID")
	assert.NoError(err)
}

func cleanDB(t *testing.T, uuids ...string) {
	qs := make([]*cmneo4j.Query, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &cmneo4j.Query{
			Cypher: `
			MATCH (a:Thing {uuid: $uuid})
			OPTIONAL MATCH (a)-[annotation]-(c:Content)
			OPTIONAL MATCH (a)-[eq:EQUIVALENT_TO]-(canonical)
			OPTIONAL MATCH (canonical)<-[eq2:EQUIVALENT_TO]-(concepts)
			DETACH DELETE annotation, eq, eq2, canonical, a`,
			Params: map[string]interface{}{
				"uuid": uuid,
			},
		}
	}
	err := driver.Write(qs...)
	assert.NoError(t, err, fmt.Sprintf("Error executing clean up cypher. Error: %v", err))
}

func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst, "TEST_TRANS_ID")
	assert.NoError(errrr)
}

func assertListContainsAll(assert *assert.Assertions, list interface{}, items ...interface{}) {
	assert.Len(list, len(items))
	for _, item := range items {
		assert.Contains(list, item)
	}
}

func getExpectedContent() Content {
	return Content{
		ID:     "http://www.ft.com/things/" + contentUUID,
		APIURL: "http://api.ft.com/content/" + contentUUID,
	}
}
