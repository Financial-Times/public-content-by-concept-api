package main

import (
	"encoding/json"
	"os"
	"testing"

	annrw "github.com/Financial-Times/annotations-rw-neo4j/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	cnt "github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/subjects-rw-neo4j/subjects"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"fmt"
)

const (
	//Generate uuids so there's no clash with real data
	contentUUID            = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	MSJConceptUUID         = "5d1510f8-2779-4b74-adab-0a5eb138fca6"
	FakebookConceptUUID    = "eac853f5-3859-4c08-8540-55e043719400"
	MetalMickeyConceptUUID = "0483bef8-5797-40b8-9b25-b12e492f63c6"
)

func TestFindMatchingContentForV2Annotation(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	organisationRW := writeOrganisations(assert, db, &batchRunner)
	annotationsRWV2 := writeV2Annotations(assert, db, &batchRunner)

	defer deleteContent(contentRW)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRWV2)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MSJConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestFindMatchingContentForV1Annotation(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	organisationRW := writeOrganisations(assert, db, &batchRunner)
	annotationsRWV1 := writeV1Annotations(assert, db, &batchRunner)
	subjectsRW := writeSubjects(assert, db, &batchRunner)

	defer deleteContent(contentRW)
	defer deleteOrganisations(organisationRW)
	defer deleteSubjects(subjectsRW)
	defer deleteAnnotations(annotationsRWV1)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MetalMickeyConceptUUID)
	assert.NoError(err, "Unexpected error for concept %s", MetalMickeyConceptUUID)
	assert.True(found, "Found no matching content for concept %s", MetalMickeyConceptUUID)
	assert.Equal(1, len(contentList), "Didn't get the same list of content")
	assertListContainsAll(assert, contentList, getExpectedContent())
}

func TestRetrieveNoContentWhenThereAreNoContentForThatConcept(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	organisationRW := writeOrganisations(assert, db, &batchRunner)

	defer deleteContent(contentRW)
	defer deleteOrganisations(organisationRW)

	contentByConceptDriver := newCypherDriver(db, "prod")
	content, found, err := contentByConceptDriver.read(MSJConceptUUID)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(content), "Should not get any content items")
}

func TestRetrieveNoContentWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	annotationsRWV1 := writeV1Annotations(assert, db, &batchRunner)
	annotationsRWV2 := writeV2Annotations(assert, db, &batchRunner)

	organisationRW := organisations.NewCypherOrganisationService(batchRunner, db)
	assert.NoError(organisationRW.Initialise())
	subjectsRW := subjects.NewCypherSubjectsService(batchRunner, db)
	assert.NoError(subjectsRW.Initialise())

	defer deleteContent(contentRW)
	defer deleteSubjects(subjectsRW)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRWV2)
	defer deleteAnnotations(annotationsRWV1)

	contentByConceptDriver := newCypherDriver(db, "prod")
	contentList, found, err := contentByConceptDriver.read(MSJConceptUUID)
	assert.NoError(err, "Unexpected error for concept %s", MSJConceptUUID)
	assert.False(found, "Found annotations for concept %s", MSJConceptUUID)
	assert.Equal(0, len(contentList), "Didn't get the right number of content items, content=%s", contentList)
}

func writeContent(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	contentRW := cnt.NewCypherDriver(*batchRunner, db)
	assert.NoError(contentRW.Initialise())
	writeJSONToService(contentRW, "./fixtures/Content-3fc9fe3e-af8c-4f7f-961a-e5065392bb31.json", assert)
	return contentRW
}

func deleteContent(contentRW baseftrwapp.Service) {
	contentRW.Delete(contentUUID)
}

func writeOrganisations(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	organisationRW := organisations.NewCypherOrganisationService(*batchRunner, db)
	assert.NoError(organisationRW.Initialise())
	writeJSONToService(organisationRW, "./fixtures/Organisation-MSJ-5d1510f8-2779-4b74-adab-0a5eb138fca6.json", assert)
	writeJSONToService(organisationRW, "./fixtures/Organisation-Fakebook-eac853f5-3859-4c08-8540-55e043719400.json", assert)
	return organisationRW
}

func deleteOrganisations(organisationRW baseftrwapp.Service) {
	organisationRW.Delete(MSJConceptUUID)
	organisationRW.Delete(FakebookConceptUUID)
}

func writeV1Annotations(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) annrw.Service {
	annotationsRW := annrw.NewAnnotationsService(*batchRunner, db, "v1")
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v1.json", assert)
	return annotationsRW
}

func writeV2Annotations(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) annrw.Service {
	annotationsRW := annrw.NewAnnotationsService(*batchRunner, db, "v2")
	assert.NoError(annotationsRW.Initialise())
	writeJSONToAnnotationsService(annotationsRW, contentUUID, "./fixtures/Annotations-3fc9fe3e-af8c-4f7f-961a-e5065392bb31-v2.json", assert)
	return annotationsRW
}

func writeSubjects(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	subjectsRW := subjects.NewCypherSubjectsService(*batchRunner, db)
	assert.NoError(subjectsRW.Initialise())
	writeJSONToService(subjectsRW, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json", assert)
	return subjectsRW
}

func deleteSubjects(subjectsRW baseftrwapp.Service) {
	fmt.Printf("halihó - próbálkozom")
	subjectsRW.Delete(MetalMickeyConceptUUID)
}

func deleteAnnotations(annotationsRW annrw.Service) {
	annotationsRW.Delete(contentUUID)
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

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) *neoism.Database {
	db := getDatabaseConnection(t, assert)
	return db
}

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) *neoism.Database {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func getExpectedContent() content {
	return content{
		ID:     "http://www.ft.com/things/" + contentUUID,
		APIURL: "http://api.ft.com/content/" + contentUUID,
	}
}
