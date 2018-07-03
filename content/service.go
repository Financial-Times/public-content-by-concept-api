package content

import (
	log "github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
)

const (
	defaultLimit    = 50
	wwwThingsPrefix = "http://www.ft.com/things/"
)

// Driver interface
type ContentByConceptServicer interface {
	Check() error
	GetContentForConcept(conceptUUID string, params RequestParams) (contentList, bool, error)
}

// CypherDriver struct
type ConceptService struct {
	conn neoutils.NeoConnection
}

type RequestParams struct {
	contentLimit  int
	fromDateEpoch int64
	toDateEpoch   int64
}

func NewContentByConceptService(conn neoutils.NeoConnection) ContentByConceptServicer {
	return ConceptService{conn}
}

func (cd ConceptService) Check() error {
	return neoutils.Check(cd.conn)
}

type neoReadStruct struct {
	UUID  string   `json:"uuid"`
	Types []string `json:"types"`
}

func (cd ConceptService) GetContentForConcept(conceptUUID string, params RequestParams) (contentList, bool, error) {
	var results []neoReadStruct
	var query *neoism.CypherQuery

	log.Debugf("Query params are %v", params)

	var whereClause string
	if params.fromDateEpoch > 0 && params.toDateEpoch > 0 {
		whereClause = " WHERE c.publishedDateEpoch > {fromDate} AND c.publishedDateEpoch < {toDate}"
	}

	parameters := neoism.Props{
		"conceptUUID":     conceptUUID,
		"maxContentItems": params.contentLimit,
		"fromDate":        params.fromDateEpoch,
		"toDate":          params.toDateEpoch}

	// New concordance model
	query = &neoism.CypherQuery{
		Statement: `
			MATCH (:Concept{uuid:{conceptUUID}})-[:EQUIVALENT_TO]->(canon:Concept)
			MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[]-(c:Content)` +
			whereClause +
			` WITH DISTINCT c
			ORDER BY c.publishedDateEpoch DESC
			RETURN c.uuid as uuid, labels(c) as types
			LIMIT({maxContentItems})`,
		Parameters: parameters,
		Result:     &results,
	}
	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil || len(results) == 0 {
		return contentList{}, false, err
	}
	log.Debugf("Found the following content for uuid %s: %v", conceptUUID, results)

	return neoReadStructToContentList(&results), true, nil
}

func neoReadStructToContentList(results *[]neoReadStruct) []content {
	cntList := contentList{}
	for _, result := range *results {
		var con = content{}
		con.APIURL = mapper.APIURL(result.UUID, result.Types, "")
		con.ID = wwwThingsPrefix + result.UUID //Not using mapper as this has a different prefix (www.ft.com not api.ft.com)
		cntList = append(cntList, con)
	}
	return cntList
}
