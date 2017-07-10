package main

import (
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type driver interface {
	read(id string, limit int, fromDateEpoch int64, toDateEpoch int64) (contentList, bool, error)
	checkConnectivity() error
}

// CypherDriver struct
type cypherDriver struct {
	conn neoutils.NeoConnection
	env  string
}

func newCypherDriver(conn neoutils.NeoConnection, env string) cypherDriver {
	return cypherDriver{conn, env}
}

func (cd cypherDriver) checkConnectivity() error {
	return neoutils.Check(cd.conn)
}

type neoReadStruct struct {
	UUID  string   `json:"uuid"`
	Types []string `json:"types"`
}

func (cd cypherDriver) read(conceptUUID string, limit int, fromDateEpoch int64, toDateEpoch int64) (contentList, bool, error) {
	results := []neoReadStruct{}
	var query *neoism.CypherQuery

	if cd.isNewConcordanceModel(conceptUUID) {
		var whereClause string
		if fromDateEpoch > 0 && toDateEpoch > 0 {
			whereClause = " WHERE c.publishedDateEpoch > {fromDate} AND c.publishedDateEpoch < {toDate} "
		}

		query = &neoism.CypherQuery{
			Statement: `
				MATCH (cc:Concept{uuid:{conceptUUID}})-[r:EQUIVALENT_TO]->(canon:Concept)
				MATCH (canon)<-[:EQUIVALENT_TO]-(leaves)<-[]-(c:Content)` +
				whereClause +
				`RETURN c.uuid as uuid, labels(c) as types
				ORDER BY c.publishedDateEpoch DESC
				LIMIT({maxContentItems})`,
			Parameters: neoism.Props{
				"conceptUUID":     conceptUUID,
				"maxContentItems": limit,
				"fromDate":        fromDateEpoch,
				"toDate":          toDateEpoch,
			},
			Result: &results,
		}
	} else {
		var whereClause string
		if fromDateEpoch > 0 && toDateEpoch > 0 {
			whereClause = " WHERE c.publishedDateEpoch > {fromDate} AND c.publishedDateEpoch < {toDate} "
		}
		query = &neoism.CypherQuery{
			Statement: `
		MATCH (upp:UPPIdentifier{value:{conceptUUID}})-[:IDENTIFIES]->(cc:Concept)
		MATCH (c:Content)-[rel]->(cc)` +
				whereClause +
				`RETURN c.uuid as uuid, labels(c) as types
			ORDER BY c.publishedDateEpoch DESC
			LIMIT({maxContentItems})`,
			Parameters: neoism.Props{
				"conceptUUID":     conceptUUID,
				"maxContentItems": limit,
				"fromDate":        fromDateEpoch,
				"toDate":          toDateEpoch,
			},
			Result: &results,
		}
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil || len(results) == 0 {
		return contentList{}, false, err
	}

	log.Debugf("Found %d pieces of content for uuid: %s", len(results), conceptUUID)

	cntList := neoReadStructToContentList(&results, cd.env)
	return cntList, true, nil
}

func (cd cypherDriver) isNewConcordanceModel(conceptUUID string) bool {
	results := []struct{
		uuid string
	}{}

	query := &neoism.CypherQuery{
		Statement: `
		MATCH (upp:UPPIdentifier{value:{conceptUUID}})-[:IDENTIFIES]->(cc:Concept)
		MATCH (cc)-[:EQUIVALENT_TO]->(canonical:Concept) RETURN cc.uuid as uuid`,
		Parameters: neoism.Props{
			"conceptUUID": conceptUUID,
		},
		Result: &results,
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})
	if err != nil || len(results) == 0 {
		return false
	}

	return true
}

func neoReadStructToContentList(neo *[]neoReadStruct, env string) []content {
	cntList := contentList{}
	for _, neoCon := range *neo {
		var con = content{}
		con.APIURL = mapper.APIURL(neoCon.UUID, neoCon.Types, env)
		con.ID = wwwThingsPrefix + neoCon.UUID //Not using mapper as this has a different prefix (www.ft.com not api.ft.com)
		cntList = append(cntList, con)
	}
	return cntList
}

const (
	defaultLimit    = 50
	wwwThingsPrefix = "http://www.ft.com/things/"
)
