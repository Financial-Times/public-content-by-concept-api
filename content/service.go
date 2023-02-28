package content

import (
	"errors"
	"net/url"
	"strings"

	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
)

var ErrContentNotFound = errors.New("content not found")

// ConceptService interacts with Neo4j db to extract content by concept information
type ConceptService struct {
	driver *cmneo4j.Driver
	apiURL string
}

type RequestParams struct {
	Page          int
	ContentLimit  int
	FromDateEpoch int64
	ToDateEpoch   int64
}

func NewContentByConceptService(driver *cmneo4j.Driver, apiURL string) (*ConceptService, error) {
	_, err := url.ParseRequestURI(apiURL)
	if err != nil {
		return nil, err
	}

	return &ConceptService{
		driver: driver,
		apiURL: apiURL,
	}, nil
}

func (cd *ConceptService) CheckConnection() (string, error) {
	err := cd.driver.VerifyConnectivity()
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

	var dateFilter string
	if params.FromDateEpoch > 0 && params.ToDateEpoch > 0 {
		dateFilter = " AND c.publishedDateEpoch > $fromDate AND c.publishedDateEpoch < $toDate"
	}

	// skipCount determines how many rows to skip before returning the results
	skipCount := 0
	if params.Page > 1 {
		skipCount = (params.Page - 1) * params.ContentLimit
	}

	parameters := map[string]interface{}{
		"conceptUUID":     conceptUUID,
		"skipCount":       skipCount,
		"maxContentItems": params.ContentLimit,
		"fromDate":        params.FromDateEpoch,
		"toDate":          params.ToDateEpoch,
	}

	// New concordance model
	query := &cmneo4j.Query{
		Cypher: `
			MATCH (:Concept{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canon:Concept)
			MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[]-(c:Content)
			WHERE NOT 'LiveEvent' IN labels(c)` +
			dateFilter +
			` WITH DISTINCT c
			ORDER BY c.publishedDateEpoch DESC
			SKIP ($skipCount)
			RETURN c.uuid as uuid, labels(c) as types
			LIMIT($maxContentItems)`,
		Params: parameters,
		Result: &results,
	}

	err := cd.driver.Read(query)
	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		return nil, ErrContentNotFound
	}
	if err != nil {
		return nil, err
	}

	cntList := make([]Content, 0)
	for _, result := range results {
		cntList = append(cntList, Content{
			ID:     idURL(result.UUID),
			APIURL: apiURL(result.UUID, cd.apiURL),
		})
	}

	return cntList, nil
}

func (cd *ConceptService) GetContentForConceptImplicitly(conceptUUID string) ([]Content, error) {
	var results []struct {
		UUID  string   `json:"uuid"`
		Types []string `json:"types"`
	}

	query := &cmneo4j.Query{
		Cypher: ` 
		MATCH (:Thing{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canonicalConcept:Concept)
		MATCH (canonicalConcept)<-[:EQUIVALENT_TO]-(leaf)
		MATCH (leaf)<-[:HAS_BROADER|HAS_PARENT|IS_PART_OF*0..]-(narrowerLeaf)
		MATCH (narrowerLeaf)-[:EQUIVALENT_TO]->(narrowerCanonical)
		WITH DISTINCT narrowerCanonical
		MATCH (narrowerCanonical)<-[:EQUIVALENT_TO]-(conceptLeaves)
		MATCH (conceptLeaves)-[]-(content:Content)
		WITH DISTINCT content
		RETURN content.uuid as uuid, labels(content) as types
		UNION
		MATCH (:Thing{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canonicalConcept:Concept)
		MATCH (canonicalConcept)<-[:EQUIVALENT_TO]-(leaf)
		MATCH (leaf)-[:IMPLIED_BY*0..]->(narrowerLeaf)
		MATCH (narrowerLeaf)-[:EQUIVALENT_TO]->(narrowerCanonical)
		WITH DISTINCT narrowerCanonical
		MATCH (narrowerCanonical)<-[:EQUIVALENT_TO]-(conceptLeaves)
		MATCH (conceptLeaves)-[]-(content:Content)
		WITH DISTINCT content
		RETURN content.uuid as uuid, labels(content) as types`,
		Params: map[string]interface{}{"conceptUUID": conceptUUID},
		Result: &results,
	}

	err := cd.driver.Read(query)
	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		return nil, ErrContentNotFound
	}
	if err != nil {
		return nil, err
	}

	cntList := make([]Content, 0)
	for _, result := range results {
		cntList = append(cntList, Content{
			ID:     idURL(result.UUID),
			APIURL: apiURL(result.UUID, cd.apiURL),
		})
	}

	return cntList, nil
}

func idURL(uuid string) string {
	return ThingsPrefix + uuid
}

func apiURL(uuid, baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/content/" + uuid
}
