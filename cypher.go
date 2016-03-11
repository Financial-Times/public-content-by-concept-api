package main

import (
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type driver interface {
	read(id string) (cntList ContentList, found bool, err error)
	checkConnectivity() error
}

// CypherDriver struct
type cypherDriver struct {
	db  *neoism.Database
	env string
}

func newCypherDriver(db *neoism.Database, env string) cypherDriver {
	return cypherDriver{db, env}
}

func (cd cypherDriver) checkConnectivity() error {
	results := []struct {
		ID int
	}{}
	query := &neoism.CypherQuery{
		Statement: "MATCH (x) RETURN ID(x) LIMIT 1",
		Result:    &results,
	}
	err := cd.db.Cypher(query)
	log.Debugf("CheckConnectivity results:%+v  err: %+v", results, err)
	return err
}

type neoReadStruct struct {
	UUID  string   `json:"uuid"`
	Types []string `json:"types"`
}

func (cd cypherDriver) read(conceptUUID string) (cntList ContentList, found bool, err error) {
	results := []neoReadStruct{}
	query := &neoism.CypherQuery{
		Statement: `
		MATCH (c:Content)-[rel]->(cc:Concept{uuid:"2384fa7a-d514-3d6a-a0ea-3a711f66d0d8"})
    RETURN c.uuid as uuid, labels(c) as types`,
		Parameters: neoism.Props{"conceptUUID": conceptUUID},
		Result:     &results,
	}

	err = cd.db.Cypher(query)
	if err != nil {
		log.Errorf("Error looking up uuid %s with query %s from neoism: %+v", conceptUUID, query.Statement, err)
		return ContentList{}, false, fmt.Errorf("Error accessing Content datastore for concept with uuid: %s", conceptUUID)
	}
	log.Debugf("Found %d pieces of content for uuid: %s", len(results), conceptUUID)

	if (len(results)) == 0 {
		return ContentList{}, false, nil
	}

	contentList, err := neoReadStructToContentList(&results, cd.env)
	return contentList, true, nil
}

func neoReadStructToContentList(neo *[]neoReadStruct, env string) (cntList []Content, err error) {
	cntList = make([]Content, len(*neo))
	for _, neoCon := range *neo {
		var con = Content{}
		con.APIURL = mapper.APIURL(neoCon.UUID, neoCon.Types, env)
		con.ID = "http://www.ft.com/things/" + con.ID
		cntList = append(cntList, con)
	}
	return cntList, nil
}

const (
	contentType = "content"
)
