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
	contentUUID 		= "ce3f2f5e-33d1-4c36-89e3-51aa00fd5660"
	conceptUUID             = "412e4ca3-f8d5-4456-8606-064c1dba3c45"
	noBodyContentUuid	= "6440aa4a-1298-4a49-9346-78d546bc0229"
)

var contentWithoutABody = content{
	UUID:  noBodyContentUuid,
	Title: "Missing Body",
}

var standardContent = content{
	UUID:          contentUUID,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
}

var shorterContent = content{
	UUID: contentUUID,
	Body: "With No Publish Date and No Title",
}

var updatedContent = content{
	UUID: contentUUID,
	Title: "New Ttitle",
	PublishedDate: "1999-12-12T01:00:00.000Z",
	Body: "Doesn't matter",
}

func TestDeleteWithNoRelsIsDeleted(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")

	deleted, err := contentDriver.Delete(standardContent.UUID)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)
	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)

	c, deleted, err := contentDriver.Read(standardContent.UUID)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)
}

func TestDeleteWithRelsBecomesThing(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeRelationship(db, standardContent.UUID, conceptUUID, t, assert)

	deleted, err := contentDriver.Delete(standardContent.UUID)
	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)

	c, found, err := contentDriver.Read(standardContent.UUID)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	exists, err := findConcept(standardContent.UUID, "Thing", db)
	assert.NoError(err)
	assert.True(exists, "Failed to find Thing for deleted content with relationships")
}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")

	storedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to retireve stored content")
	assert.Equal(storedContent.(content).UUID, standardContent.UUID, "Failed to match UUID")
	assert.Equal(storedContent.(content).Title, standardContent.Title, "Failed to match Title")
	assert.Equal(storedContent.(content).PublishedDate, standardContent.PublishedDate, "Failed to match PublishedDate")
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(shorterContent), "Failed to write content")

	storedContent, _, err := contentDriver.Read(shorterContent.UUID)

	assert.NoError(err)
	assert.Empty(storedContent.(content).PublishedDate)
	assert.Empty(storedContent.(content).Title)

}

func TestCreateDoesNotStoreBody(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	storedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to retrive stored content")
	assert.Empty(storedContent.(content).Body, "Body should not have been stored")

}

func TestWillUpdateProperties(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentUUID)

	assert.NoError(err)
	assert.Equal(storedContent.(content).Title, standardContent.Title)
	assert.Equal(storedContent.(content).PublishedDate, standardContent.PublishedDate)



	assert.NoError(contentDriver.Write(updatedContent), "Failed to write updated content")
	storedContent, _, err = contentDriver.Read(contentUUID)

	assert.NoError(err)
	assert.Equal(storedContent.(content).Title, updatedContent.Title, "Should have updated Title but it is still present")
	assert.Equal(storedContent.(content).PublishedDate, updatedContent.PublishedDate, "Should have updated PublishedDate but it is still present")
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentUUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent.(content).Title)
	assert.NotEmpty(storedContent.(content).PublishedDate)

	assert.NoError(contentDriver.Write(shorterContent), "Failed to write updated content")
	storedContent, _, err = contentDriver.Read(contentUUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to rtreive updated content")
	assert.Empty(storedContent.(content).Title, "Update should have removed Title but it is still present")
	assert.Empty(storedContent.(content).PublishedDate, "Update should have removed PublishedDate but it is still present")

}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	uuid := standardContent.UUID
	contentReceived := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	contentDriver.Write(contentReceived)

	result := []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}{}

	getEpochQuery := &neoism.CypherQuery{
		Statement: `
			MATCH (t:Content {uuid:{uuid}}) RETURN t.publishedDateEpoch
			`,
		Parameters: neoism.Props{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err := contentDriver.conn.CypherBatch([]*neoism.CypherQuery{getEpochQuery})
	assert.NoError(err)
	assert.Equal(3600, result[0].PublishedDateEpoc, "Epoc of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

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
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(contentWithoutABody), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentWithoutABody.UUID)

	assert.NoError(err)
	assert.Equal(content{}, storedContent, "No content should be written when the content has no body")
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(assert)
	cleanDB(db, t, assert)
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
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'}) DETACH DELETE mc", conceptUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fc:Thing {uuid: '%v'}) DETACH DELETE fc", contentUUID),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func writeRelationship(db neoutils.NeoConnection, contentId string, conceptId string, t *testing.T, assert *assert.Assertions) {
	var annotateQuery string
	var qs []*neoism.CypherQuery

	annotateQuery = `
                MERGE (content:Thing{uuid:{contentId}})
                MERGE (concept:Thing) ON CREATE SET concept.uuid = {conceptId}
                MERGE (content)-[pred:SOME_PPREDICATE]->(concept)
          `

	qs = []*neoism.CypherQuery{
		{
			Statement:  annotateQuery,
			Parameters: neoism.Props{"contentId": contentId, "conceptId": conceptId},
		},
	}


	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func findConcept(uuid string, label string, db neoutils.NeoConnection) (bool, error) {

	type  thing struct{
		UUID  string  `json:"uuid,omitempty"`
	}

	result := []struct{
		thing
	}{}

	var statement = fmt.Sprintf("MATCH (t:%s {uuid:'%s'}) RETURN t.uuid as uuid", label, uuid)

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: statement,
		Result: &result,
	}
	err := db.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})

	return len(result) > 0, err
}

func checkDbClean(db neoutils.CypherRunner, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"t.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (t:Thing) WHERE t.uuid in {uuids} RETURN t.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{standardContent.UUID, conceptUUID },
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
