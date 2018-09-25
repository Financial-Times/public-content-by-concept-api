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
	showImplicit  bool
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

	var whereClause string
	if params.fromDateEpoch > 0 && params.toDateEpoch > 0 {
		whereClause = " WHERE content.publishedDateEpoch > {fromDate} AND content.publishedDateEpoch < {toDate}"
	}

	maxDepth := "0"
	if params.showImplicit {
		maxDepth = "10"
	}

	parameters := neoism.Props{
		"conceptUUID":     conceptUUID,
		"maxContentItems": params.contentLimit,
		"fromDate":        params.fromDateEpoch,
		"toDate":          params.toDateEpoch,
	}

	query = &neoism.CypherQuery{
		Statement: `
			MATCH (c:Thing{uuid:{conceptUUID}})-[:EQUIVALENT_TO]->(canonicalConcept:Concept)
			MATCH (canonicalConcept)<-[:EQUIVALENT_TO]-(leaf)
			MATCH (leaf)<-[:HAS_BROADER*0..` +
			maxDepth +
			`]-(narrowerLeaf)
			MATCH (narrowerLeaf)-[:EQUIVALENT_TO]->(narrowerCanonical)
			WITH DISTINCT narrowerCanonical
			MATCH (narrowerCanonical)<-[:EQUIVALENT_TO]-(conceptLeaves)
			MATCH (conceptLeaves)-[]-(content:Content)` +
			whereClause +
			` WITH DISTINCT content
			ORDER BY content.publishedDateEpoch DESC
			RETURN content.uuid as uuid, labels(content) as types
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
