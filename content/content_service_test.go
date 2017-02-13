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
	minimalContentUuid      = "ce3f2f5e-33d1-4c36-89e3-51aa00fd5660"
	fullContentUuid         = "4f21ba89-940c-4708-8959-cc5816afa639"
	noBodyContentUuid       = "6440aa4a-1298-4a49-9346-78d546bc0229"
	financialTimesBrandUuid = "dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
	fastFtBrandUuid         = "5c7592a8-1f0c-11e4-b0cb-b2227cce2b54"
	thingsUriPrefix         = "http://api.ft.com/things/"
	conceptUUID             = "412e4ca3-f8d5-4456-8606-064c1dba3c45"
	storyPackageUUID        = "3b08c76c-7479-461d-9f0e-a4e92dca56f7"
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
}

var fullContent = content{
	UUID:          fullContentUuid,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Fuller body",
	Brands:        []brand{financialTimesBrand, fastFtBrand},
	StoryPackage:  storyPackageUUID,
}

var financialTimesBrand = brand{
	Id: thingsUriPrefix + financialTimesBrandUuid,
}

var fastFtBrand = brand{
	Id: thingsUriPrefix + fastFtBrandUuid,
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(minimalContent), "Failed to write content")

	deleted, err := contentDriver.Delete(minimalContentUuid)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", minimalContentUuid)
	assert.NoError(err, "Error deleting content for uuid %s", minimalContentUuid)

	c, deleted, err := contentDriver.Read(minimalContentUuid)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", minimalContentUuid)
	assert.NoError(err, "Error trying to find content for uuid %s", minimalContentUuid)
}

func TestDeleteWithRelContentLifecycleAndRelIsCuratedFor(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")

	deleted, err := contentDriver.Delete(fullContentUuid)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", fullContentUuid)
	assert.NoError(err, "Error deleting content for uuid %s", fullContentUuid)

	c, found, err := contentDriver.Read(fullContentUuid)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", fullContentUuid)
	assert.NoError(err, "Error trying to find content for uuid %s", fullContentUuid)
	assert.Equal(0, checkIsCuratedForRelationship(db, storyPackageUUID, assert), "incorrect number of isCuratedFor relationships")
}

func TestDeleteWithRelNonContentLifecycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")
	writeClassifiedByRelationship(db, fullContentUuid, conceptUUID, "annotations-v1", assert)

	deleted, err := contentDriver.Delete(fullContentUuid)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", fullContentUuid)
	assert.NoError(err, "Error deleting content for uuid %s", fullContentUuid)

	c, found, err := contentDriver.Read(fullContentUuid)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", fullContentUuid)
	assert.NoError(err, "Error trying to find content for uuid %s", fullContentUuid)
	assert.Equal(1, checkClassifedByRelationship(db, conceptUUID, "annotations-v1", t, assert), "incorrect number, of is classified by relationships")
}

func TestDeleteWithRelNoLifecycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")
	writeClassifiedByRelationship(db, fullContentUuid, conceptUUID, "", assert)

	deleted, err := contentDriver.Delete(fullContentUuid)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", fullContentUuid)
	assert.NoError(err, "Error deleting content for uuid %s", fullContentUuid)

	c, found, err := contentDriver.Read(fullContentUuid)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", fullContentUuid)
	assert.NoError(err, "Error trying to find content for uuid %s", fullContentUuid)
	assert.Equal(0, checkClassifedByRelationship(db, conceptUUID, "", t, assert), "incorrect number, of is classified by relationships")
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

	actualContent := storedContent.(content)

	assert.Equal(fullContent.UUID, actualContent.UUID)
	assert.Equal(fullContent.PublishedDate, actualContent.PublishedDate)
	assert.Equal(fullContent.Title, actualContent.Title)
	assert.Equal(fullContent.StoryPackage, actualContent.StoryPackage)
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

	var shorterFullContent = content{
		UUID: fullContentUuid,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterFullContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
}

func TestUpdateWillRemoveRelsWithNoLifeCycleAndRelIsCuratedFor(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")
	writeClassifiedByRelationship(db, fullContentUuid, conceptUUID, "", assert)
	storedFullContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedFullContent)
	assert.Equal(1, checkIsCuratedForRelationship(db, storyPackageUUID, assert), "incorrect number of isCuratedFor relationships")

	var shorterFullContent = content{
		UUID: fullContentUuid,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterFullContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
	assert.Equal(0, checkClassifedByRelationship(db, conceptUUID, "", t, assert), "incorrect number, of is classified by relationships")
	assert.Equal(0, checkIsCuratedForRelationship(db, storyPackageUUID, assert), "incorrect number of isCuratedFor relationships")
}

func TestUpdateWillNotRemoveRelsWithNonContentLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(fullContent), "Failed to write content")
	writeClassifiedByRelationship(db, fullContentUuid, conceptUUID, "annotations-v1", assert)
	storedFullContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedFullContent)

	var shorterFullContent = content{
		UUID: fullContentUuid,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterFullContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(fullContentUuid)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
	assert.Equal(1, checkClassifedByRelationship(db, conceptUUID, "annotations-v1", t, assert), "incorrect number, of is classified by relationships")
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
	contentReceived := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	contentDriver.Write(contentReceived)

	result := []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}{}

	getEpocQuery := &neoism.CypherQuery{
		Statement: `
			MATCH (t:Content {uuid:"12345"}) RETURN t.publishedDateEpoch
			`,
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
	defer cleanDB(db, t, assert)

	uuid := "12345"
	contentReceived := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	contentDriver.Write(contentReceived)

	result := []struct {
		PrefLabel string `json:"t.prefLabel"`
	}{}

	getPrefLabelQuery := &neoism.CypherQuery{
		Statement: `
				MATCH (t:Content {uuid:"12345"}) RETURN t.prefLabel
				`,
		Result: &result,
	}

	err := contentDriver.conn.CypherBatch([]*neoism.CypherQuery{getPrefLabelQuery})
	assert.NoError(err)
	assert.Equal("TestContent", result[0].PrefLabel, "PrefLabel should be 'TestContent")
}

func TestContentWontBeWrittenIfNoBody(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(contentDriver.Write(contentWithoutABody), "Failed to write content")
	storedFullContent, _, err := contentDriver.Read(noBodyContentUuid)

	assert.NoError(err)
	assert.Equal(content{}, storedFullContent, "No content should be written when the content has no body")
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
			Statement: fmt.Sprintf("MATCH (mc:Thing {uuid: '%v'}) DETACH DELETE mc", minimalContentUuid),
		},
		{
			Statement: fmt.Sprintf("MATCH (fc:Thing {uuid: '%v'}) DETACH DELETE fc", fullContentUuid),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func writeClassifiedByRelationship(db neoutils.NeoConnection, contentId string, conceptId string, lifecycle string, assert *assert.Assertions) {
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

func checkIsCuratedForRelationship(db neoutils.NeoConnection, spID string, assert *assert.Assertions) int {
	countQuery := `	MATCH (t:Thing{uuid:{storyPackageId}})-[r:IS_CURATED_FOR]->(x)
			RETURN count(r) as c`

	results := []struct {
		Count int `json:"c"`
	}{}

	qs := &neoism.CypherQuery{
		Statement:  countQuery,
		Parameters: neoism.Props{"storyPackageId": spID},
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
			"uuids": []string{fullContentUuid, minimalContentUuid},
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
