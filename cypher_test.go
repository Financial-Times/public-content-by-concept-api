package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	annrw "github.com/Financial-Times/annotations-rw-neo4j/annotations"
	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-rw-neo4j/content"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/subjects-rw-neo4j/subjects"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	//Generate uuids so there's no clash with real data
	contentUUID            = "3fc9fe3e-af8c-4f7f-961a-e5065392bb31"
	MSJConceptUUID         = "5d1510f8-2779-4b74-adab-0a5eb138fca6"
	FakebookConceptUUID    = "eac853f5-3859-4c08-8540-55e043719400"
	MetalMickeyConceptUUID = "0483bef8-5797-40b8-9b25-b12e492f63c6"
)

func TestRetrieveMultipleAnnotations(t *testing.T) {
	assert := assert.New(t)
	expectedAnnotations := []annotation{getExpectedFakebookAnnotation(),
		getExpectedMallStreetJournalAnnotation(),
		getExpectedMetalMickeyAnnotation()}
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	// writeContent(assert, db, &batchRunner)
	// writeOrganisations(assert, db, &batchRunner)
	// writeV1Annotations(assert, db, &batchRunner)
	// writeV2Annotations(assert, db, &batchRunner)
	// writeSubjects(assert, db, &batchRunner)

	contentRW := writeContent(assert, db, &batchRunner)
	organisationRW := writeOrganisations(assert, db, &batchRunner)
	annotationsRWV1 := writeV1Annotations(assert, db, &batchRunner)
	annotationsRWV2 := writeV2Annotations(assert, db, &batchRunner)
	subjectsRW := writeSubjects(assert, db, &batchRunner)

	defer cleanDB(db, t, assert)
	defer deleteContent(contentRW)
	defer deleteOrganisations(organisationRW)
	defer deleteAnnotations(annotationsRWV1)
	defer deleteAnnotations(annotationsRWV2)
	defer deleteSubjects(subjectsRW)

	annotationsDriver := newCypherDriver(db, "prod")
	anns, found, err := annotationsDriver.read(contentUUID)
	assert.NoError(err, "Unexpected error for content %s", contentUUID)
	assert.True(found, "Found no annotations for content %s", contentUUID)
	assert.Equal(len(expectedAnnotations), len(anns), "Didn't get the same number of annotations")
	assertListContainsAll(assert, anns, getExpectedFakebookAnnotation(), getExpectedMallStreetJournalAnnotation(), getExpectedMetalMickeyAnnotation())
}

func TestRetrieveNoAnnotationsWhenThereAreNonePresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	organisationRW := writeOrganisations(assert, db, &batchRunner)
	subjectsRW := writeSubjects(assert, db, &batchRunner)

	defer cleanDB(db, t, assert)
	defer deleteContent(contentRW)
	defer deleteOrganisations(organisationRW)
	defer deleteSubjects(subjectsRW)

	annotationsDriver := newCypherDriver(db, "prod")
	anns, found, err := annotationsDriver.read(contentUUID)
	assert.NoError(err, "Unexpected error for content %s", contentUUID)
	assert.False(found, "Found annotations for content %s", contentUUID)
	assert.Equal(0, len(anns), "Didn't get the same number of annotations")
}

func TestRetrieveNoAnnotationsWhenThereAreNoConceptsPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)

	contentRW := writeContent(assert, db, &batchRunner)
	annotationsRWV1 := writeV1Annotations(assert, db, &batchRunner)
	annotationsRWV2 := writeV2Annotations(assert, db, &batchRunner)

	defer cleanDB(db, t, assert)
	defer deleteContent(contentRW)
	defer deleteAnnotations(annotationsRWV1)
	defer deleteAnnotations(annotationsRWV2)

	annotationsDriver := newCypherDriver(db, "prod")
	anns, found, err := annotationsDriver.read(contentUUID)
	assert.NoError(err, "Unexpected error for content %s", contentUUID)
	assert.False(found, "Found annotations for content %s", contentUUID)
	assert.Equal(0, len(anns), "Didn't get the same number of annotations, anns=%s", anns)
}

func writeContent(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	contentRW := content.NewCypherDriver(*batchRunner, db)
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

func writeSubjects(assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) baseftrwapp.Service {
	subjectsRW := subjects.NewCypherSubjectsService(*batchRunner, db)
	assert.NoError(subjectsRW.Initialise())
	writeJSONToService(subjectsRW, "./fixtures/Subject-MetalMickey-0483bef8-5797-40b8-9b25-b12e492f63c6.json", assert)
	return subjectsRW
}

func deleteSubjects(subjectsRW baseftrwapp.Service) {
	subjectsRW.Delete(MetalMickeyConceptUUID)
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
	cleanDB(db, t, assert)
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

func cleanDB(db *neoism.Database, t *testing.T, assert *assert.Assertions) {
	uuids := []string{
		contentUUID,
		MSJConceptUUID,
		FakebookConceptUUID,
		MetalMickeyConceptUUID,
	}

	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%s'}) DETACH DELETE a", uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func getExpectedFakebookAnnotation() annotation {
	return annotation{
		Predicate: "http://www.ft.com/ontology/annotation/mentions",
		ID:        "http://api.ft.com/things/eac853f5-3859-4c08-8540-55e043719400",
		APIURL:    "http://api.ft.com/organisations/eac853f5-3859-4c08-8540-55e043719400",
		Types: []string{
			"http://www.ft.com/ontology/organisation/Organisation",
			"http://www.ft.com/ontology/company/PublicCompany",
			"http://www.ft.com/ontology/company/Company",
		},
		LeiCode:   "BQ4BKCS1HXDV9TTTTTTTT",
		PrefLabel: "Fakebook, Inc.",
	}
}

func getExpectedMallStreetJournalAnnotation() annotation {
	return annotation{
		Predicate: "http://www.ft.com/ontology/annotation/mentions",
		ID:        "http://api.ft.com/things/5d1510f8-2779-4b74-adab-0a5eb138fca6",
		APIURL:    "http://api.ft.com/organisations/5d1510f8-2779-4b74-adab-0a5eb138fca6",
		Types: []string{
			"http://www.ft.com/ontology/organisation/Organisation",
		},
		PrefLabel: "The Mall Street Journal",
	}
}

func getExpectedMetalMickeyAnnotation() annotation {
	return annotation{
		Predicate: "http://www.ft.com/ontology/classification/isClassifiedBy",
		ID:        "http://api.ft.com/things/0483bef8-5797-40b8-9b25-b12e492f63c6",
		APIURL:    "http://api.ft.com/things/0483bef8-5797-40b8-9b25-b12e492f63c6",
		Types: []string{
			"http://www.ft.com/ontology/Subject",
		},
		PrefLabel: "Metal Mickey",
	}
}
