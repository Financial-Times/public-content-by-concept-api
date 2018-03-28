package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	annrw "github.com/Financial-Times/annotations-rw-neo4j/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/concepts-rw-neo4j/concepts"
	cnt "github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/jmcvetta/neoism"
	_ "github.com/joho/godotenv/autoload"
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
)

// Reusable Neo4J connection
var db neoutils.NeoConnection

func init() {
	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, _ = neoutils.Connect(neoUrl(), conf)
	if db == nil {
		panic("Cannot connect to Neo4J")
	}
}

func neoUrl() string {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	return url
}

func TestFindMatchingContentForV2Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeOrganisations(assert, db)
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV1Annotation(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeOrganisations(assert, db)
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, db, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MetalMickeyConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV2AnnotationWithLimit(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeContent(assert, db, content2UUID)
	writeOrganisations(assert, db)
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")
	writeAnnotations(assert, db, content2UUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, content2UUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, 1, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestRetrieveNoContentForV1AnnotationForExclusiveDatePeriod(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeOrganisations(assert, db)
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeConcept(assert, db, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json")

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID, MetalMickeyConceptUUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	fromDate, _ := time.Parse("2006-01-02", "2014-03-08")
	toDate, _ := time.Parse("2006-01-02", "2014-03-09")
	contentList, found, err := contentByConceptDriver.read(MetalMickeyConceptUUID, defaultLimit, fromDate.Unix(), toDate.Unix())
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.False(found, "Found matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(0, len(contentList), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoContentForThatConcept(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeOrganisations(assert, db)

	defer cleanDB(t, MSJConceptUUID, contentUUID, FakebookConceptUUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	content, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(content), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)

	writeContent(assert, db, contentUUID)
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json")
	writeAnnotations(assert, db, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json")

	defer cleanDB(t, content2UUID, MSJConceptUUID, contentUUID, MetalMickeyConceptUUID, FakebookConceptUUID)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func TestNewConcordanceModelWithBrands(t *testing.T) {
	assert := assert.New(t)
	defer cleanDB(t, content2UUID, content3UUID, OnyxPikeBrandUUID, OnyPikeyRightBrandUUID)

	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content2UUID)
	writeAnnotations(assert, db, content3UUID, "./fixtures/Annotations-5a9c7429-e76b-4f37-b5d1-842d64a45167-V2.json")
	writeAnnotations(assert, db, content2UUID, "./fixtures/Annotations-bfa97890-76ff-4a35-a775-b8768f7ea383-V2.json")
	writeConcept(assert, db, fmt.Sprintf("./fixtures/Brand-OnyxPike-%v.json", OnyxPikeBrandUUID))

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(OnyxPikeBrandUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", OnyxPikeBrandUUID)
	assert.True(found, "Found annotations for concept %s", OnyxPikeBrandUUID)
	assert.Equal(2, len(contentList), "Didn't get the right number of content items, content=%s", contentList)

}

func TestNewConcordanceModelWithBrandsDoesntReturnParentContent(t *testing.T) {
	assert := assert.New(t)
	defer cleanDB(t, content2UUID, content3UUID, content4UUID, OnyxPikeBrandUUID, OnyxPikeParentBrandUUID, OnyPikeyRightBrandUUID)

	writeContent(assert, db, content2UUID)
	writeContent(assert, db, content3UUID)
	writeContent(assert, db, content4UUID)

	writeAnnotations(assert, db, content2UUID, fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content2UUID))
	writeAnnotations(assert, db, content3UUID, fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content3UUID))
	writeAnnotations(assert, db, content4UUID, fmt.Sprintf("./fixtures/Annotations-%v-V2.json", content4UUID))

	writeConcept(assert, db, fmt.Sprintf("./fixtures/Brand-OnyxPike-%v.json", OnyxPikeBrandUUID))
	writeConcept(assert, db, fmt.Sprintf("./fixtures/Brand-OnyxPikeParent-%v.json", OnyxPikeParentBrandUUID))

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(OnyxPikeBrandUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", OnyxPikeBrandUUID)
	assert.True(found, "Found annotations for concept %s", OnyxPikeBrandUUID)
	assert.Equal(2, len(contentList), "Didn't get the right number of content items, content=%s", contentList)

}

func writeContent(assert *assert.Assertions, db neoutils.NeoConnection, contentUUID string) {
	contentRW := cnt.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-"+contentUUID+".json", assert)
}

func writeOrganisations(assert *assert.Assertions, db neoutils.NeoConnection) {
	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())
	writeJSONToService(organisationRW, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json", assert)
	writeJSONToService(organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json", assert)
}

func writeAnnotations(assert *assert.Assertions, db neoutils.NeoConnection, contentUUID string, fixtures string) annrw.Service {
	annotationsRW := annrw.NewCypherAnnotationsService(db)
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, contentUUID, fixtures, assert)
	return annotationsRW
}

func writeConcept(assert *assert.Assertions, db neoutils.NeoConnection, fixture string) concepts.ConceptService {
	conceptsRW := concepts.NewConceptService(db)
	assert.NoError(conceptsRW.Initialise())
	log.Printf("Logging Concepts: %v", fixture)
	writeJSONToConceptRW(conceptsRW, fixture, assert)
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

func writeJSONToConceptRW(service concepts.ConceptService, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	_, errrr := service.Write(inst, "TEST_TRANS_ID")
	assert.NoError(errrr)
}

func writeJSONToAnnotationsService(service annrw.Service, contentUUID string, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, errr := service.DecodeJSON(dec)
	assert.NoError(errr, "Error parsing file %s", pathToJSONFile)
	errrr := service.Write(contentUUID, "v1", "annotations-v1", "tid_test", inst)
	assert.NoError(errrr)
}

func assertListContainsAll(assert *assert.Assertions, list interface{}, items ...interface{}) {
	assert.Len(list, len(items))
	for _, item := range items {
		assert.Contains(list, item)
	}
}

func getExpectedContent() content {
	return content{
		ID:     "http://www.ft.com/things/" + contentUUID,
		APIURL: "http://api.ft.com/content/" + contentUUID,
	}
}
