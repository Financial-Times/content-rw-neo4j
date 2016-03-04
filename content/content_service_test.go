// +build !jenkins

package content

import (
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	minimalContentUuid      = "ce3f2f5e-33d1-4c36-89e3-51aa00fd5660"
	fullContentUuid         = "4f21ba89-940c-4708-8959-cc5816afa639"
	noBodyContentUuid       = "6440aa4a-1298-4a49-9346-78d546bc0229"
	financialTimesBrandUuid = "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
	fastFtBrandUuid         = "5c7592a8-1f0c-11e4-b0cb-b2227cce2b54"
	thingsUriPrefix         = "http://api.ft.com/things/"
)

var contentWithoutABody = content{
	UUID:  noBodyContentUuid,
	Title: "Missing Body",
}

var minimalContent = content{
	UUID:          minimalContentUuid,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
	Brands:        []brand{financialTimesBrand},
}

var fullContent = content{
	UUID:          minimalContentUuid,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Fuller body",
	Brands:        []brand{financialTimesBrand, fastFtBrand},
}

var financialTimesBrand = brand{
	Id: thingsUriPrefix + financialTimesBrandUuid,
}

var fastFtBrand = brand{
	Id: thingsUriPrefix + fastFtBrandUuid,
}

var contentDriver baseftrwapp.Service

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(minimalContent), "Failed to write content")

	found, err := contentDriver.Delete(minimalContentUuid)
	assert.True(found, "Didn't manage to delete content for uuid %", minimalContentUuid)
	assert.NoError(err, "Error deleting content for uuid %s", minimalContentUuid)

	c, found, err := contentDriver.Read(minimalContentUuid)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", minimalContentUuid)
	assert.NoError(err, "Error trying to find content for uuid %s", minimalContentUuid)
}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")

	storedContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedContent)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")
	storedFullContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedFullContent)

	assert.NoError(contentDriver.Write(minimalContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(minimalContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(minimalContent), "Failed to write content")

	storedContent, _, err := contentDriver.Read(minimalContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedContent)
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	uuid := "12345"
	contentRecieved := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	contentDriver.Write(contentRecieved)

	result := []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}{}

	getEpocQuery := &neoism.CypherQuery{
		Statement: `
			MATCH (t:Content {uuid:"12345"}) RETURN t.publishedDateEpoch
			`,
		Result: &result,
	}

	err := contentDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getEpocQuery})
	assert.NoError(err)
	assert.Equal(3600, result[0].PublishedDateEpoc, "Epoc of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	uuid := "12345"
	contentRecieved := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	contentDriver.Write(contentRecieved)

	result := []struct {
		PrefLabel string `json:"t.prefLabel"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Content {uuid:"12345"}) RETURN t.prefLabel
				`,
		Result: &result,
	}

	err := contentDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal("TestContent", result[0].PrefLabel, "PrefLabel should be 'TestContent")
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) *neoism.Database {
	db := getDatabaseConnection(assert)
	cleanDB(db, t, assert)
	checkDbClean(db, t)
	return db
}

func getDatabaseConnection(assert *assert.Assertions) *neoism.Database {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db *neoism.Database, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'})-[rel:IS_CLASSIFIED_BY]->(b:Brand) DELETE mc, rel", minimalContentUuid),
		},
		{
			Statement: fmt.Sprintf("MATCH (fc:Thing {uuid: '%v'})-[rel:IS_CLASSIFIED_BY]->(b:Brand) DELETE fc, rel", fullContentUuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db *neoism.Database, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"org.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (org:Thing) WHERE org.uuid in {uuids} RETURN org.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{fullContentUuid, minimalContentUuid},
		},
		Result: &result,
	}
	err := db.Cypher(&checkGraph)
	assert.NoError(err)
	assert.Empty(result)
}

func getCypherDriver(db *neoism.Database) CypherDriver {
	cr := NewCypherDriver(neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 3), db)
	cr.Initialise()
	return cr
}
