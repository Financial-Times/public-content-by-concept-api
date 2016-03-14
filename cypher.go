package main

import (
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type driver interface {
	read(id string) (cntList contentList, found bool, err error)
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

func (cd cypherDriver) read(conceptUUID string) (cntList contentList, found bool, err error) {
	results := []neoReadStruct{}
	query := &neoism.CypherQuery{
		Statement: `
		MATCH (c:Content)-[rel]->(cc:Concept{uuid:{conceptUUID}})
    	RETURN c.uuid as uuid, labels(c) as types
		ORDER BY c.publishedDateEpoch
		LIMIT({maxContentItems})`,
		Parameters: neoism.Props{
			"conceptUUID":     conceptUUID,
			"maxContentItems": maxContentItems,
		},
		Result: &results,
	}

	err = cd.db.Cypher(query)
	if err != nil {
		log.Errorf("Error looking up uuid %s with query %s from neoism: %+v", conceptUUID, query.Statement, err)
		return contentList{}, false, fmt.Errorf("Error accessing content datastore for concept with uuid: %s", conceptUUID)
	}

	log.Debugf("Found %d pieces of content for uuid: %s", len(results), conceptUUID)

	if (len(results)) == 0 {
		return contentList{}, false, nil
	}

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
