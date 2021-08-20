package content

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
)

var ErrContentNotFound = errors.New("content not found")

// CypherDriver struct
type ConceptService struct {
	driver neo4j.Driver
}

type RequestParams struct {
	Page          int
	ContentLimit  int
	FromDateEpoch int64
	ToDateEpoch   int64
}

func NewContentByConceptService(neoURL string, neoConf func(*neo4j.Config)) (*ConceptService, error) {
	driver, err := neo4j.NewDriver(neoURL, neo4j.NoAuth(), neoConf)
	if err != nil {
		return nil, fmt.Errorf("could not initiate Neo4j driver object: %w", err)
	}
	return &ConceptService{driver}, nil
}

func (cd *ConceptService) Close() error {
	return cd.driver.Close()
}

func (cd *ConceptService) CheckConnection() (string, error) {
	err := checkConnection(cd.driver)
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

	var whereClause string
	if params.FromDateEpoch > 0 && params.ToDateEpoch > 0 {
		whereClause = " WHERE c.publishedDateEpoch > {fromDate} AND c.publishedDateEpoch < {toDate}"
	}

	// skipCount determines how many rows to skip before returning the results
	skipCount := (params.Page - 1) * params.ContentLimit

	parameters := map[string]interface{}{
		"conceptUUID":     conceptUUID,
		"skipCount":       skipCount,
		"maxContentItems": params.ContentLimit,
		"fromDate":        params.FromDateEpoch,
		"toDate":          params.ToDateEpoch,
	}

	session := cd.driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	_, err := session.ReadTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		neoResult, err := transaction.Run(
			`
			MATCH (:Concept{uuid:{conceptUUID}})-[:EQUIVALENT_TO]->(canon:Concept)
			MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[]-(c:Content)`+
				whereClause+
				` WITH DISTINCT c
			ORDER BY c.publishedDateEpoch DESC
			SKIP ({skipCount})
			RETURN c.uuid as uuid, labels(c) as types
			LIMIT({maxContentItems})`,
			parameters,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to run get content transaction: %w", err)
		}

		err = parseTransactionArrResult(neoResult, &results)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Neo content results: %w", err)
		}

		if len(results) == 0 {
			return nil, ErrContentNotFound
		}

		return nil, nil
	})

	if err != nil {
		return nil, err
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

// parseTransactionArrResult iterates trough the records in the given neo4j.Result object and parses them into the given output object.
// The function relies that the neo4j.Result object contains array of records each of which with the same fields.
// TODO: could eventually be moved out in a common library
func parseTransactionArrResult(result neo4j.Result, output interface{}) error {
	var records []*neo4j.Record
	for result.Next() {
		records = append(records, result.Record())
	}

	// It is important to check Err() after Next() returning false to find out whether it is end of result stream or
	// an error that caused the end of result consumption.
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to consume the transaction result stream: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	// Get the keys of the records, we rely on that they are all the same for all the records in the result
	keys := records[0].Keys

	var recordsMaps []map[string]interface{}
	for _, rec := range records {
		recMap := make(map[string]interface{})
		for _, k := range keys {
			val, ok := rec.Get(k)
			if !ok {
				return fmt.Errorf("failed to parse transaction result: unknown key %s", k)
			}
			recMap[k] = val
		}
		recordsMaps = append(recordsMaps, recMap)
	}

	recordsMarshalled, err := json.Marshal(recordsMaps)
	if err != nil {
		return fmt.Errorf("failed to marshall parsed transaction results: %w", err)
	}

	err = json.Unmarshal(recordsMarshalled, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshall parsed transaction results: %w", err)
	}

	return nil
}

// TODO: move to common library
func checkConnection(driver neo4j.Driver) error {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	_, err := session.ReadTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run(`MATCH (n) RETURN id(n) LIMIT 1`, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to run read transaction: %w", err)
		}
		_, err = result.Single()
		if err != nil {
			return nil, fmt.Errorf("failed to get node from db: %w", err)
		}
		return nil, nil
	})

	return err
}
