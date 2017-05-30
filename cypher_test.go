package main

import (
	"encoding/json"
	"fmt"
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
	"github.com/stretchr/testify/assert"
)

const (
	//Generate uuids so there's no clash with real data
	contentUUID            = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	content2UUID           = "bfa97890-76ff-4a35-a775-b8768f7ea383"
	MSJConceptUUID         = "5d1510f8-2779-4b74-adab-0a5eb138fca6"
	FakebookConceptUUID    = "eac853f5-3859-4c08-8540-55e043719400"
	MetalMickeyConceptUUID = "0483bef8-5797-40b8-9b25-b12e492f63c6"
)

func TestFindMatchingContentForV2Annotation(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	organisationRW := writeOrganisations(assert, db)
	annotationsRWV2 := writeV2Annotations(assert, db, contentUUID)

	defer deleteContent(contentRW, contentUUID)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRWV2)
	defer cleanUpBrandAndIdentifier(db, t, assert)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV1Annotation(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	organisationRW := writeOrganisations(assert, db)
	annotationsRWV1 := writeV1Annotations(assert, db)
	subjectsRW := writeSubjects(assert, db)

	defer deleteContent(contentRW, contentUUID)
	defer deleteOrganisations(organisationRW)
	defer deleteSubjects(subjectsRW)
	defer deleteAnnotations(annotationsRWV1)
	defer cleanUpBrandAndIdentifier(db, t, assert)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MetalMickeyConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV2AnnotationWithLimit(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	contentRW2 := writeContent(assert, db, content2UUID)
	organisationRW := writeOrganisations(assert, db)
	annotationsRW1 := writeV2Annotations(assert, db, contentUUID)
	annotationsRW2 := writeV2Annotations(assert, db, content2UUID)

	defer deleteContent(contentRW, contentUUID)
	defer deleteContent(contentRW2, content2UUID)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRW1)
	defer deleteAnnotations(annotationsRW2)
	defer cleanUpBrandAndIdentifier(db, t, assert)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, 1, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestRetrieveNoContentForV1AnnotationForExclusiveDatePeriod(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	organisationRW := writeOrganisations(assert, db)
	annotationsRWV1 := writeV1Annotations(assert, db)
	subjectsRW := writeSubjects(assert, db)

	defer deleteContent(contentRW, contentUUID)
	defer deleteOrganisations(organisationRW)
	defer deleteSubjects(subjectsRW)
	defer deleteAnnotations(annotationsRWV1)
	defer cleanUpBrandAndIdentifier(db, t, assert)

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
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	organisationRW := writeOrganisations(assert, db)

	defer deleteContent(contentRW, contentUUID)
	defer deleteOrganisations(organisationRW)
	defer cleanUpBrandAndIdentifier(db, t, assert)

	contentByConceptDriver := newCypherDriver(db, "prod")
	content, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(content), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnection(t, assert)

	contentRW := writeContent(assert, db, contentUUID)
	annotationsRWV1 := writeV1Annotations(assert, db)
	annotationsRWV2 := writeV2Annotations(assert, db, contentUUID)

	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())
	subjectsRW := concepts.NewConceptService(db)
	assert.NoError(subjectsRW.Initialise())

	defer deleteContent(contentRW, contentUUID)
	defer deleteSubjects(subjectsRW)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRWV2)
	defer deleteAnnotations(annotationsRWV1)
	defer cleanUpBrandAndIdentifier(db, t, assert)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID, defaultLimit, 0, 0)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func writeContent(assert *assert.Assertions, db neoutils.NeoConnection, contentUUID string) baseftrwapp.Service {
	contentRW := cnt.NewCypherContentService(db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-"+contentUUID+".json", assert)
	return contentRW
}

func deleteContent(contentRW baseftrwapp.Service, UUID string) {
	contentRW.Delete(UUID)
}

func writeOrganisations(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())
	writeJSONToService(organisationRW, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json", assert)
	writeJSONToService(organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json", assert)
	return organisationRW
}

func deleteOrganisations(organisationRW baseftrwapp.Service) {
	organisationRW.Delete(MSJConceptUUID)
	organisationRW.Delete(FakebookConceptUUID)
}

func writeV1Annotations(assert *assert.Assertions, db neoutils.NeoConnection) annrw.Service {
	annotationsRW := annrw.NewCypherAnnotationsService(db, "v1", "annotations-v1")
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json", assert)
	return annotationsRW
}

func writeV2Annotations(assert *assert.Assertions, db neoutils.NeoConnection, id string) annrw.Service {
	annotationsRW := annrw.NewCypherAnnotationsService(db, "v2", "annotations-v2")
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, id, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json", assert)
	return annotationsRW
}

func writeSubjects(assert *assert.Assertions, db neoutils.NeoConnection) baseftrwapp.Service {
	subjectsRW := concepts.NewConceptService(db)
	assert.NoError(subjectsRW.Initialise())
	writeJSONToService(subjectsRW, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json", assert)
	return subjectsRW
}

func deleteSubjects(subjectsRW baseftrwapp.Service) {
	subjectsRW.Delete(MetalMickeyConceptUUID)
}

func deleteAnnotations(annotationsRW annrw.Service) {
	annotationsRW.Delete(contentUUID)
	annotationsRW.Delete(content2UUID)
}

func cleanUpBrandAndIdentifier(db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			//deletes 'brand' which only has type Thing
			Statement: fmt.Sprintf("MATCH (j:Thing {uuid: '%v'}) DETACH DELETE j", "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"),
		},
		{
			//deletes upp identifier for the above parent 'org'
			Statement: fmt.Sprintf("MATCH (k:Identifier {value: '%v'}) DETACH DELETE k", "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}

func writeJSONToAnnotationsService(service annrw.Service, contentUUID string, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, errr := service.DecodeJSON(dec)
	assert.NoError(errr, "Error parsing file %s", pathToJSONFile)
	errrr := service.Write(contentUUID, inst)
	assert.NoError(errrr)
}

func assertListContainsAll(assert *assert.Assertions, list interface{}, items ...interface{}) {
	assert.Len(list, len(items))
	for _, item := range items {
		assert.Contains(list, item)
	}
}

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	if testing.Short() {
		t.Skip("Short flag set - skipping Neo4j integration test.")
	}

	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func getExpectedContent() content {
	return content{
		ID:     "http://www.ft.com/things/" + contentUUID,
		APIURL: "http://api.ft.com/content/" + contentUUID,
	}
}
