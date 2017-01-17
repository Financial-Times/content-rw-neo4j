// +build !jenkins

package content

import (
	"fmt"
	"os"
	"testing"


	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	conceptUUID             = "412e4ca3-f8d5-4456-8606-064c1dba3c45"
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
	//cleanDB(db, t, assert)
	deleteThingNodeAndAllRelationships(db, assert)
	checkDbClean(db, t)
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

func cleanDB(db neoutils.CypherRunner, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'}) DETACH DELETE mc", standardContent.UUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fc:Thing {uuid: '%v'}) DETACH DELETE fc", standardContent.UUID),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func writeClassifedByRelationship(db neoutils.NeoConnection, contentId string, conceptId string, lifecycle string, t *testing.T, assert *assert.Assertions) {
	var annotateQuery string
	var qs []*neoism.CypherQuery

	if lifecycle == "" {
		annotateQuery = `
                MERGE (content:Thing{uuid:{contentId}})
                MERGE (upp:Identifier:UPPIdentifier{value:{conceptId}})
                MERGE (upp)-[:IDENTIFIES]->(concept:Thing) ON CREATE SET concept.uuid = {conceptId}
                MERGE (content)-[pred:IS_CLASSIFIED_BY {platformVersion:'v1'}]->(concept)
          `
		qs = []*neoism.CypherQuery{
			{
				Statement:  annotateQuery,
				Parameters: neoism.Props{"contentId": contentId, "conceptId": conceptId},
			},
		}
	} else {
		annotateQuery = `
                MERGE (content:Thing{uuid:{contentId}})
                MERGE (upp:Identifier:UPPIdentifier{value:{conceptId}})
                MERGE (upp)-[:IDENTIFIES]->(concept:Thing) ON CREATE SET concept.uuid = {conceptId}
                MERGE (content)-[pred:IS_CLASSIFIED_BY {platformVersion:'v1', lifecycle: {lifecycle}}]->(concept)
          `
		qs = []*neoism.CypherQuery{
			{
				Statement:  annotateQuery,
				Parameters: neoism.Props{"contentId": contentId, "conceptId": conceptId, "lifecycle": lifecycle},
			},
		}

	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkClassifedByRelationship(db neoutils.NeoConnection, conceptId string, lifecycle string, t *testing.T, assert *assert.Assertions) int {
	countQuery := `	MATCH (t:Thing{uuid:{conceptId}})-[r:IS_CLASSIFIED_BY {platformVersion:'v1', lifecycle: {lifecycle}}]-(x)
			MATCH (t)<-[:IDENTIFIES]-(s:Identifier:UPPIdentifier)
			RETURN count(r) as c`

	results := []struct {
		Count int `json:"c"`
	}{}

	qs := &neoism.CypherQuery{
		Statement:  countQuery,
		Parameters: neoism.Props{"conceptId": conceptId, "lifecycle": lifecycle},
		Result:     &results,
	}

	err := db.CypherBatch([]*neoism.CypherQuery{qs})
	assert.NoError(err)

	return results[0].Count
}


func checkDbClean(db neoutils.CypherRunner, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"org.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (org:Thing) WHERE org.uuid in {uuids} RETURN org.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{standardContent.UUID},
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

