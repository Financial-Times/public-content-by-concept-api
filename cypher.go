package main

import (
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
	"github.com/Financial-Times/neo-utils-go/neoutils"
)

// Driver interface
type driver interface {
	read(id string) (cntList contentList, found bool, err error)
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

func (cd cypherDriver) read(conceptUUID string) (cntList contentList, found bool, err error) {
	results := []neoReadStruct{}
	query := &neoism.CypherQuery{
		Statement: `
		MATCH (upp:UPPIdentifier{value:{conceptUUID}})-[:IDENTIFIES]->(cc:Concept)
		MATCH (c:Content)-[rel]->(cc)
    		RETURN c.uuid as uuid, labels(c) as types
		ORDER BY c.publishedDateEpoch DESC
		LIMIT({maxContentItems})`,
		Parameters: neoism.Props{
			"conceptUUID":     conceptUUID,
			"maxContentItems": maxContentItems,
		},
		Result: &results,
	}

	if err := cd.conn.CypherBatch([]*neoism.CypherQuery{query}); err != nil || len(results) == 0 {
		return contentList{}, false, err
	}

	log.Debugf("Found %d pieces of content for uuid: %s", len(results), conceptUUID)

	contentList, err := neoReadStructToContentList(&results, cd.env)
	return contentList, true, nil
}

func neoReadStructToContentList(neo *[]neoReadStruct, env string) (cntList []content, err error) {
	cntList = contentList{}
	for _, neoCon := range *neo {
		var con = content{}
		con.APIURL = mapper.APIURL(neoCon.UUID, neoCon.Types, env)
		con.ID = wwwThingsPrefix + neoCon.UUID //Not using mapper as this has a different prefix (www.ft.com not api.ft.com)
		cntList = append(cntList, con)
	}
	return cntList, nil
}

const (
	contentType     = "content"
	maxContentItems = 50
	wwwThingsPrefix = "http://www.ft.com/things/"
)
