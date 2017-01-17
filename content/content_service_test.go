// +build !jenkins

package content

import (
	"os"
	"testing"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")

	storedContent, _, err := contentDriver.Read(standardContent.UUID)
	expectedContent := content {
		UUID:          standardContent.UUID,
		Title:         standardContent.Title,
		PublishedDate: standardContent.PublishedDate,
	}
	assert.NoError(err)
	assert.Equal(expectedContent, storedContent.(content), "Not all expected content properties were present.")
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver.Write(standardContent)

	result := []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}{}

	getEpocQuery := &neoism.CypherQuery{
		Statement: `
			MATCH (t:Content {uuid:{uuid}}) RETURN t.publishedDateEpoch
			`,
		Parameters: neoism.Props{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err := contentDriver.conn.CypherBatch([]*neoism.CypherQuery{getEpocQuery})
	assert.NoError(err)
	assert.Equal(3600, result[0].PublishedDateEpoc, "Epoc of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver.Write(standardContent)

	result := []struct {
		PrefLabel string `json:"t.prefLabel"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Content {uuid:{uuid}}) RETURN t.prefLabel
				`,
		Parameters: neoism.Props{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err := contentDriver.conn.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)

	assert.Equal(standardContent.Title, result[0].PrefLabel, "PrefLabel should be 'Content Title'")
}

func TestContentWontBeWrittenIfNoBody(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(contentWithoutABody), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentWithoutABody.UUID)

	assert.NoError(err)
	assert.Equal(content{}, storedContent, "No content should be written when the content has no body")
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(assert)
	deleteThingNodeAndAllRelationships(db, assert)
	checkDbClean(db, assert)
	return db
}

func getDatabaseConnection(assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func checkDbClean(db neoutils.CypherRunner, assert *assert.Assertions) {

	result := []struct {
		Uuid string `json:"t.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (t:Thing) WHERE t.uuid in {uuids} RETURN t.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{standardContent.UUID, testBrandId, FTBrandId},
		},
		Result: &result,
	}
	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)
	assert.Empty(result)
}

func getCypherDriver(db neoutils.NeoConnection) service {
	cr := NewCypherContentService(db)
	cr.Initialise()
	return cr
}

