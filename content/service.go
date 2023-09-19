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
		UUID                    string   `json:"uuid"`
		Types                   []string `json:"types"`
		ContentPublication      string   `json:"contentPublication"`
		RelationshipPublication string   `json:"relationshipPublication"`
		ConceptPublication      string   `json:"conceptPublication"`
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
			MATCH (:Thing{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canon:Thing)
			MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[r]-(c:Content)
			WHERE NOT 'LiveEvent' IN labels(c)` +
			dateFilter +
			` WITH DISTINCT c, r, leaves
			ORDER BY c.publishedDateEpoch DESC
			SKIP ($skipCount)
			RETURN c.uuid as uuid, labels(c) as types, r.publication as relationshipPublication, c.publication as contentPublication, leaves.authority as conceptPublication
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
			ID:                      idURL(result.UUID),
			APIURL:                  apiURL(result.UUID, cd.apiURL),
			ContentPublication:      result.ContentPublication,
			RelationshipPublication: result.RelationshipPublication,
			ConceptPublication:      result.ConceptPublication,
		})
	}

	return cntList, nil
}

func (cd *ConceptService) GetContentForConceptImplicitly(conceptUUID string) ([]Content, error) {
	var results []struct {
		UUID                    string   `json:"uuid"`
		Types                   []string `json:"types"`
		ContentPublication      string   `json:"contentPublication"`
		RelationshipPublication string   `json:"relationshipPublication"`
		ConceptPublication      string   `json:"conceptPublication"`
	}

	query := &cmneo4j.Query{
		Cypher: ` 
		MATCH (:Thing{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canonicalConcept:Thing)
		MATCH (canonicalConcept)<-[:EQUIVALENT_TO]-(leaf)
		MATCH (leaf)<-[:HAS_BROADER|HAS_PARENT|IS_PART_OF*0..]-(narrowerLeaf)
		MATCH (narrowerLeaf)-[:EQUIVALENT_TO]->(narrowerCanonical)
		WITH DISTINCT narrowerCanonical
		MATCH (narrowerCanonical)<-[:EQUIVALENT_TO]-(conceptLeaves)
		MATCH (conceptLeaves)-[r]-(content:Content)
		WITH DISTINCT content,conceptLeaves ,r
		RETURN content.uuid as uuid, labels(content) as types, r.publication as relationshipPublication, content.publication as contentPublication, conceptLeaves.authority as conceptPublication
		UNION
		MATCH (:Thing{uuid:$conceptUUID})-[:EQUIVALENT_TO]->(canonicalConcept:Thing)
		MATCH (canonicalConcept)<-[:EQUIVALENT_TO]-(leaf)
		MATCH (leaf)-[:IMPLIED_BY*0..]->(narrowerLeaf)
		MATCH (narrowerLeaf)-[:EQUIVALENT_TO]->(narrowerCanonical)
		WITH DISTINCT narrowerCanonical
		MATCH (narrowerCanonical)<-[:EQUIVALENT_TO]-(conceptLeaves)
		MATCH (conceptLeaves)-[r]-(content:Content)
		WITH DISTINCT content,conceptLeaves,r
		RETURN content.uuid as uuid, labels(content) as types,  r.publication as relationshipPublication, content.publication as contentPublication, conceptLeaves.authority as conceptPublication`,
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
			ID:                      idURL(result.UUID),
			APIURL:                  apiURL(result.UUID, cd.apiURL),
			ContentPublication:      result.ContentPublication,
			RelationshipPublication: result.RelationshipPublication,
			ConceptPublication:      result.ConceptPublication,
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
