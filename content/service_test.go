// +build integration

package content

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Financial-Times/go-logger/v2"

	annrw "github.com/Financial-Times/annotations-rw-neo4j/v4/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/concepts-rw-neo4j/concepts"
	cnt "github.com/Financial-Times/content-rw-neo4j/v3/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	// Generate uuids so there's no clash with real data
	contentUUID             = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	content2UUID            = "bfa97890-76ff-4a35-a775-b8768f7ea383"
	content3UUID            = "5a9c7429-e76b-4f37-b5d1-842d64a45167"
	content4UUID            = "8e193b84-4697-41aa-a480-065831d1d964"
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
)

const defaultLimit = 10
const defaultPage = 1

// Reusable Neo4J connection
var db neoutils.NeoConnection

func init() {
	log := logger.NewUPPLogger("test-service", "info")
	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, _ = neoutils.Connect(neoURL(), conf)
	if db == nil {
		log.Fatal("Cannot connect to Neo4J")
	}
}

func neoURL() string {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	return url
}

func TestFindMatchingContentForV2Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeAnnotations(assert, db, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")
	writeConcept(assert, db, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver := &ConceptService{conn: db}
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV1Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, db, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver := &ConceptService{conn: db}
	contentList, err := contentByConceptDriver.GetContentForConcept(MetalMickeyConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV2AnnotationWithLimit(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeContent(assert, db, content2UUID)
	writeAnnotations(assert, db, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")
	writeConcept(assert, db, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, content2UUID)

	contentByConceptDriver := &ConceptService{conn: db}
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, 1, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestRetrieveNoContentForV1AnnotationForExclusiveDatePeriod(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, db, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver := &ConceptService{conn: db}
	fromDate, _ := time.Parse("2006-01-02", "2014-03-08")
	toDate, _ := time.Parse("2006-01-02", "2014-03-09")
	contentList, err := contentByConceptDriver.GetContentForConcept(MetalMickeyConceptUUID, RequestParams{0, defaultLimit, fromDate.Unix(), toDate.Unix()})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(contentList), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoContentForThatConcept(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver := &ConceptService{conn: db}
	content, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(content), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeAnnotations(assert, db, contentUUID, "v2", "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")

	defer cleanDB(t, content2UUID, MSJConceptUUID, contentUUID, MetalMickeyConceptUUID, FakebookConceptUUID)

	contentByConceptDriver := &ConceptService{conn: db}
	contentList, err := contentByConceptDriver.GetContentForConcept(MSJConceptUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.Equal(ErrContentNotFound, err, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func TestBrandsDontReturnParentContent(t *testing.T) {
	assert := assert.New(t)
	defer cleanDB(t, content2UUID, content3UUID, content4UUID, OnyxPikeBrandUUID, OnyxPikeParentBrandUUID, OnyPikeyRightBrandUUID)

	writeContent(assert, db, content2UUID)
	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content4UUID)

	writeAnnotations(assert, db, content2UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content2UUID))
	writeAnnotations(assert, db, content3UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content3UUID))
	writeAnnotations(assert, db, content4UUID, "v2", fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content4UUID))

	writeConcept(assert, db, fmt.Sprintf("./fixtures/Brand-OnyxPike-%v.json", OnyxPikeBrandUUID))
	writeConcept(assert, db, fmt.Sprintf("./fixtures/Brand-OnyxPikeParent-%v.json", OnyxPikeParentBrandUUID))

	contentByConceptDriver := &ConceptService{conn: db}
	contentList, err := contentByConceptDriver.GetContentForConcept(OnyxPikeBrandUUID, RequestParams{0, defaultLimit, 0, 0})
	assert.NoError(err, "Unexpected error for concept %s", OnyxPikeBrandUUID)
	assert.Equal(2, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func TestContentIsReturnedFromAllLeafNodesOfConcordance(t *testing.T) {
	assert := assert.New(t)

	defer cleanDB(t, contentUUID, content2UUID, content3UUID, content4UUID, JohnSmithFSUUID, JohnSmithSmartlogicUUID, JohnSmithTMEUUID, JohnSmithOtherTMEUUID)

	writeContent(assert, db, contentUUID)
	writeContent(assert, db, content2UUID)
	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content4UUID)

	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, db, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, db, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, db, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, db, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver := &ConceptService{conn: db}

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

	writeContent(assert, db, contentUUID)
	writeContent(assert, db, content2UUID)
	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content4UUID)

	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, db, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, db, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, db, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, db, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver := &ConceptService{conn: db}

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

	writeContent(assert, db, contentUUID)
	writeContent(assert, db, content2UUID)
	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content4UUID)

	writeAnnotations(assert, db, contentUUID, "v1", "./fixtures/Annotations-JohnSmith1-v1.json")
	writeAnnotations(assert, db, content2UUID, "v1", "./fixtures/Annotations-JohnSmith2-v1.json")
	writeAnnotations(assert, db, content3UUID, "v2", "./fixtures/Annotations-JohnSmith3-v2.json")
	writeAnnotations(assert, db, content4UUID, "v2", "./fixtures/Annotations-JohnSmith4-v2.json")

	writeConcept(assert, db, "./fixtures/Person-JohnSmith-f25b0f71-4cf9-4e3a-8510-14e86d922bfe.json")

	contentByConceptDriver := &ConceptService{conn: db}

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

func TestConceptService_Check(t *testing.T) {
	assert := assert.New(t)
	contentByConceptDriver := &ConceptService{conn: db}
	_, err := contentByConceptDriver.CheckConnection()
	assert.NoError(err, "Test should always pass when connected to db")
}

func writeContent(assert *assert.Assertions, db neoutils.NeoConnection, contentUUID string) {
	contentRW := cnt.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-"+contentUUID+".json", assert)
}

func writeAnnotations(assert *assert.Assertions, db neoutils.NeoConnection, contentUUID string, lifecycle string, fixtureFile string) {
	annotationsRW := annrw.NewCypherAnnotationsService(db)
	assert.NoError(annotationsRW.Initialise())
	f, err := os.Open(fixtureFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	json, errr := annotationsRW.DecodeJSON(dec)
	assert.NoError(errr, "Error parsing file %s", fixtureFile)
	assert.NoError(annotationsRW.Write(contentUUID, lifecycle, "", "", json))
}

func writeConcept(assert *assert.Assertions, db neoutils.NeoConnection, fixture string) concepts.ConceptService {
	conceptsRW := concepts.NewConceptService(db)
	assert.NoError(conceptsRW.Initialise())
	f, err := os.Open(fixture)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := conceptsRW.DecodeJSON(dec)
	assert.NoError(errr)
	_, err = conceptsRW.Write(inst, "TEST_TRANS_ID")
	assert.NoError(err)
	return conceptsRW
}

func cleanDB(t *testing.T, uuids ...string) {
	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf(`
			MATCH (a:Thing {uuid: "%s"})
			OPTIONAL MATCH (a)<-[ii:IDENTIFIES]-(i)
			OPTIONAL MATCH (a)-[annotation]-(c:Content)
			OPTIONAL MATCH (a)-[eq:EQUIVALENT_TO]-(canonical)
			OPTIONAL MATCH (canonical)<-[eq2:EQUIVALENT_TO]-(concepts)
			DETACH DELETE ii, i, annotation, eq, eq2, canonical, a`, uuid)}
	}
	err := db.CypherBatch(qs)
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
