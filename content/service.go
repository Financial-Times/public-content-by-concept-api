package content

import (
	"errors"
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
)

var ErrContentNotFound = errors.New("content not found")

// CypherDriver struct
type ConceptService struct {
	conn neoutils.NeoConnection
}

type RequestParams struct {
	Page          int
	ContentLimit  int
	FromDateEpoch int64
	ToDateEpoch   int64
}

func NewContentByConceptService(neoURL string, neoConf neoutils.ConnectionConfig) (*ConceptService, error) {
	conn, err := neoutils.Connect(neoURL, &neoConf)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Neo4j: %w", err)
	}
	return &ConceptService{conn}, nil
}

func (cd *ConceptService) CheckConnection() (string, error) {
	err := neoutils.Check(cd.conn)
	if err != nil {
		return "Could not connect to database!", err
	}
	return "Database connection is OK", nil
}

func (cd *ConceptService) GetContentForConcept(conceptUUID string, params RequestParams) ([]Content, error) {
	var results []struct {
		UUID  string   `json:"uuid"`
		Types []string `json:"types"`
	}
	var query *neoism.CypherQuery

	var whereClause string
	if params.FromDateEpoch > 0 && params.ToDateEpoch > 0 {
		whereClause = " WHERE c.publishedDateEpoch > {fromDate} AND c.publishedDateEpoch < {toDate}"
	}

	// skipCount determines how many rows to skip before returning the results
	skipCount := (params.Page - 1) * params.ContentLimit

	parameters := neoism.Props{
		"conceptUUID":     conceptUUID,
		"skipCount":       skipCount,
		"maxContentItems": params.ContentLimit,
		"fromDate":        params.FromDateEpoch,
		"toDate":          params.ToDateEpoch}

	// New concordance model
	query = &neoism.CypherQuery{
		Statement: `
			MATCH (:Concept{uuid:{conceptUUID}})-[:EQUIVALENT_TO]->(canon:Concept)
			MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[]-(c:Content)` +
			whereClause +
			` WITH DISTINCT c
			ORDER BY c.publishedDateEpoch DESC
			SKIP ({skipCount})
			RETURN c.uuid as uuid, labels(c) as types
			LIMIT({maxContentItems})`,
		Parameters: parameters,
		Result:     &results,
	}
	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, ErrContentNotFound
	}

	cntList := make([]Content, 0)
	for _, result := range results {
		cntList = append(cntList, Content{
			ID:     ThingsPrefix + result.UUID, //Not using mapper as this has a different prefix (www.ft.com not api.ft.com)
			APIURL: mapper.APIURL(result.UUID, result.Types, ""),
		})
	}

	return cntList, nil
}
