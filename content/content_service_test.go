//go:build integration
// +build integration

package content

import (
	"os"
	"testing"

	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
	"github.com/Financial-Times/go-logger/v2"
	"github.com/stretchr/testify/assert"
)

const (
	contentUUID                  = "ce3f2f5e-33d1-4c36-89e3-51aa00fd5660"
	conceptUUID                  = "412e4ca3-f8d5-4456-8606-064c1dba3c45"
	liveBlogUUID                 = "1520b6b9-d466-49a0-b3ec-894b72338e7d"
	noBodyContentUUID            = "6440aa4a-1298-4a49-9346-78d546bc0229"
	noBodyInvalidTypeContentUUID = "1674d8b6-f3b2-4f18-9f3b-e28bcf5553a0"
	contentPlaceholderUUID       = "ed2d9fc2-b515-4f7d-8b4e-3b0c1fa90986"
	videoContentUUID             = "41bb9444-e3cf-46d4-8182-6702844dc5c1"
	storyPackageUUID             = "3b08c76c-7479-461d-9f0e-a4e92dca56f7"
	contentPackageUUID           = "45163790-eec9-11e6-abbc-ee7d9c5b3b90"
	contentCollectionUUID        = "cc65c43a-fe4e-4315-854b-9b82435be036"
	thingUUID                    = "ebcfe37d-9a70-4c8b-bf01-1feee4dff4b7"
	genericContentPackageUUID    = "27cfe7eb-549d-4d51-9cfd-98ea887a571c"
	graphicUUID                  = "087b42c2-ac7f-40b9-b112-98b3a7f9cd72"
	audioContentUUID             = "128cfcf4-c394-4e71-8c65-198a675acf53"
)

var contentWithoutABody = content{
	UUID:  noBodyContentUUID,
	Title: "Missing Body",
}

var contentPlaceholder = content{
	UUID:  contentPlaceholderUUID,
	Title: "Missing Body",
	Type:  "Content",
}

var liveBlog = content{
	UUID:  liveBlogUUID,
	Title: "Live blog",
	Type:  "Article",
}

var contentWithoutABodyWithType = content{
	UUID:  noBodyInvalidTypeContentUUID,
	Title: "Missing Body",
	Type:  "Image",
}

var videoContent = content{
	UUID:  videoContentUUID,
	Title: "Missing Body",
	Type:  "Video",
}

var graphicContent = content{
	UUID:  graphicUUID,
	Title: "Missing Body",
	Type:  "Graphic",
}

var audioContent = content{
	UUID:  audioContentUUID,
	Title: "Missing Body",
	Type:  "Audio",
}

var standardContent = content{
	UUID:          contentUUID,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
	StoryPackage:  storyPackageUUID,
}

var standardContentPackage = content{
	UUID:           contentUUID,
	Title:          "Content Title",
	PublishedDate:  "1970-01-01T01:00:00.000Z",
	Body:           "Some body",
	StoryPackage:   storyPackageUUID,
	ContentPackage: contentPackageUUID,
}

var genericContentPackage = content{
	UUID:           contentUUID,
	Title:          "Content Title",
	PublishedDate:  "1970-01-01T01:00:00.000Z",
	ContentPackage: genericContentPackageUUID,
	Type:           "ContentPackage",
}

var shorterContent = content{
	UUID: contentUUID,
	Body: "With No Publish Date and No Title",
}

var updatedContent = content{
	UUID:          contentUUID,
	Title:         "New Ttitle",
	PublishedDate: "1999-12-12T01:00:00.000Z",
	Body:          "Doesn't matter",
}

func TestDeleteWithNoRelsIsDeleted(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentService.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write content")

	deleted, err := contentService.Delete(shorterContent.UUID, "TEST_TRANS_ID")
	assert.True(deleted, "Didn't manage to delete content for uuid %s", shorterContent.UUID)
	assert.NoError(err, "Error deleting content for uuid %s", shorterContent.UUID)

	c, deleted, err := contentService.Read(shorterContent.UUID, "TEST_TRANS_ID")

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)
}

func TestDeleteWithRelsIsDeleted(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	s := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(s.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")
	writeRelationship(d, standardContent.UUID, conceptUUID, t, assert)

	deleted, err := s.Delete(standardContent.UUID, "TEST_TRANS_ID")
	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)

	c, found, err := s.Read(standardContent.UUID, "TEST_TRANS_ID")

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(found, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	exists, err := doesThingExist(standardContent.UUID, d)
	assert.NoError(err)
	assert.False(exists, "Thing should not exist for deleted content with relations")
}

func TestDeleteContentPackageIsDeletedAttachedContentCollectionRemains(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentService.Write(genericContentPackage, "TEST_TRANS_ID"), "Failed to write content package")
	writeNodeWithLabels(d, contentCollectionUUID, "Thing:Content:ContentCollection", t, assert)
	writeContentPackageContainsRelation(d, genericContentPackage.UUID, contentCollectionUUID, assert)

	deleted, err := contentService.Delete(genericContentPackage.UUID, "TEST_TRANS_ID")
	assert.NoError(err, "Error deleting Content Package for uuid %contentService", genericContentPackage.UUID)
	assert.True(deleted, "Didn't manage to delete Content Package for uuid %contentService", genericContentPackage.UUID)

	c, found, err := contentService.Read(genericContentPackage.UUID, "TEST_TRANS_ID")

	assert.Equal(content{}, c, "Found Content Package %contentService who should have been deleted", c)
	assert.False(found, "Found Content Package for uuid %contentService who should have been deleted", genericContentPackage.UUID)
	assert.NoError(err, "Error trying to find Content Package for uuid %contentService", genericContentPackage.UUID)

	exists, err := doesThingExist(genericContentPackage.UUID, d)
	assert.NoError(err)
	assert.False(exists, "Thing should not exist for deleted Content Package")

	existsCC, err := doesThingExist(contentCollectionUUID, d)
	assert.NoError(err)
	assert.True(existsCC, "Content Collection should exist")
}

func TestDeleteContentPackageIsDeletedAttachedNodeIsAlsoDeleted(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentService.Write(genericContentPackage, "TEST_TRANS_ID"), "Failed to write content package")
	writeNodeWithLabels(d, thingUUID, "Thing", t, assert)
	writeContentPackageContainsRelation(d, genericContentPackage.UUID, thingUUID, assert)

	deleted, err := contentService.Delete(genericContentPackage.UUID, "TEST_TRANS_ID")
	assert.NoError(err, "Error deleting Content Package for uuid %contentService", genericContentPackage.UUID)
	assert.True(deleted, "Didn't manage to delete Content Package for uuid %contentService", genericContentPackage.UUID)

	c, found, err := contentService.Read(genericContentPackage.UUID, "TEST_TRANS_ID")

	assert.Equal(content{}, c, "Found Content Package %contentService who should have been deleted", c)
	assert.False(found, "Found Content Package for uuid %contentService who should have been deleted", genericContentPackage.UUID)
	assert.NoError(err, "Error trying to find Content Package for uuid %contentService", genericContentPackage.UUID)

	exists, err := doesThingExist(genericContentPackage.UUID, d)
	assert.NoError(err)
	assert.False(exists, "Thing should not exist for deleted Content Package")

	existsThing, err := doesThingExist(thingUUID, d)
	assert.NoError(err)
	assert.False(existsThing, "Thing related to deleted Content Package should not exist")
}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentDriver := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentDriver.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := contentDriver.Read(standardContent.UUID, "TEST_TRANS_ID")
	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to retireve stored content")
	actualContent := storedContent.(content)

	assert.Equal(standardContent.UUID, actualContent.UUID, "Failed to match UUID")
	assert.Equal(standardContent.Title, actualContent.Title, "Failed to match Title")
	assert.Equal(standardContent.PublishedDate, actualContent.PublishedDate, "Failed to match PublishedDate")
	assert.Empty(actualContent.Body, "Body should not have been stored")
	assert.Equal(standardContent.StoryPackage, actualContent.StoryPackage)
	assert.Equal(standardContent.ContentPackage, actualContent.ContentPackage)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentDriver := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentDriver.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := contentDriver.Read(shorterContent.UUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.Empty(storedContent.(content).PublishedDate)
	assert.Empty(storedContent.(content).Title)
}

func TestWillUpdateProperties(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentDriver := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentDriver.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentUUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.Equal(storedContent.(content).Title, standardContent.Title)
	assert.Equal(storedContent.(content).PublishedDate, standardContent.PublishedDate)

	assert.NoError(contentDriver.Write(updatedContent, "TEST_TRANS_ID"), "Failed to write updated content")
	storedContent, _, err = contentDriver.Read(contentUUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.Equal(storedContent.(content).Title, updatedContent.Title, "Should have updated Title but it is still present")
	assert.Equal(storedContent.(content).PublishedDate, updatedContent.PublishedDate, "Should have updated PublishedDate but it is still present")
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentDriver := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentDriver.Write(standardContentPackage, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentUUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.NotEmpty(storedContent.(content).Title)
	assert.NotEmpty(storedContent.(content).PublishedDate)
	assert.Equal(1, checkIsCuratedForRelationship(d, storyPackageUUID, assert), "incorrect number of isCuratedFor relationships")
	assert.Equal(1, checkContainsRelationship(d, contentPackageUUID, assert), "incorrect number of contains relationships")

	assert.NoError(contentDriver.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write updated content")
	storedContent, _, err = contentDriver.Read(contentUUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to rеtriеve updated content")
	assert.Empty(storedContent.(content).Title, "Update should have removed Title but it is still present")
	assert.Empty(storedContent.(content).PublishedDate, "Update should have removed PublishedDate but it is still present")
	assert.Equal(0, checkIsCuratedForRelationship(d, storyPackageUUID, assert), "incorrect number of isCuratedFor relationships")
	assert.Equal(0, checkContainsRelationship(d, contentPackageUUID, assert), "incorrect number of contains relationships")
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	uuid := standardContent.UUID
	contentReceived := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z", Body: "Some Test text"}
	err := contentService.Write(contentReceived, "TEST_TRANS_ID")
	assert.NoError(err)

	var result []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}

	getEpochQuery := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Content {uuid:{uuid}}) RETURN t.publishedDateEpoch
			`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(getEpochQuery)
	assert.NoError(err)
	assert.Equal(3600, result[0].PublishedDateEpoc, "Epoc of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	err := contentService.Write(standardContent, "TEST_TRANS_ID")
	assert.NoError(err)

	var result []struct {
		PrefLabel string `json:"t.prefLabel"`
	}

	getPrefLabelQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid:{uuid}}) RETURN t.prefLabel
				`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(getPrefLabelQuery)
	assert.NoError(err)
	assert.Equal(standardContent.Title, result[0].PrefLabel, "PrefLabel should be 'Content Title'")
}

func TestWriteNodeLabelsAreWrittenForContent(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	err := contentService.Write(standardContent, "TEST_TRANS_ID")
	assert.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}

	getNodeLabelsQuery := []*cmneo4j.Query{
		{
			Cypher: `
				MATCH (t:Content {uuid:{uuid}}) RETURN labels(t)
				`,
			Params: map[string]interface{}{
				"uuid": standardContent.UUID,
			},
			Result: &result,
		},
	}

	err = d.Write(getNodeLabelsQuery...)
	assert.NoError(err)
	assert.Len(result[0].NodeLabels, 2, "There should be 2 node labels: Thing, Content")
	assert.Equal("Thing", result[0].NodeLabels[0], "Thing should be the parent label")
	assert.Equal("Content", result[0].NodeLabels[1], "Content should be the child label")
}

func TestWriteNodeLabelsAreWrittenForContentPackage(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	err := contentService.Write(standardContentPackage, "TEST_TRANS_ID")
	assert.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}

	getNodeLabelsQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid:{uuid}}) RETURN labels(t)
				`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(getNodeLabelsQuery)
	assert.NoError(err)
	assert.Len(result[0].NodeLabels, 3, "There should be 3 node labels: Thing, Content, ContentPackage")
	assert.Equal("Thing", result[0].NodeLabels[0], "Thing should be the grandparent label")
	assert.Equal("Content", result[0].NodeLabels[1], "Content should be the parent label")
	assert.Equal("ContentPackage", result[0].NodeLabels[2], "ContentPackage should be the child label")
}

func TestWriteNodeLabelsAreWrittenForGenericContentPackage(t *testing.T) {
	assert := assert.New(t)

	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	err := contentService.Write(genericContentPackage, "TEST_TRANS_ID")
	assert.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}
	getNodeLabelsQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid:{uuid}}) RETURN labels(t)
				`,
		Params: map[string]interface{}{
			"uuid": genericContentPackage.UUID,
		},
		Result: &result,
	}

	err = d.Write(getNodeLabelsQuery)
	assert.NoError(err)
	assert.Len(result[0].NodeLabels, 3, "There should be 3 node labels: Thing, Content, ContentPackage")
	assert.Equal("Thing", result[0].NodeLabels[0], "Thing should be the grandparent label")
	assert.Equal("Content", result[0].NodeLabels[1], "Content should be the parent label")
	assert.Equal("ContentPackage", result[0].NodeLabels[2], "ContentPackage should be the child label")
}

func TestContentWontBeWrittenIfNoBody(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	err := contentService.Write(contentWithoutABody, "TEST_TRANS_ID")
	assert.NoError(err, "Failed to write content")

	storedContent, _, err := contentService.Read(contentWithoutABody.UUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.Equal(content{}, storedContent, "No content should be written when the content has no body")
}

func TestContentWontBeWrittenIfNoBodyWithInvalidType(t *testing.T) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	contentService := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(contentService.Write(contentWithoutABodyWithType, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := contentService.Read(contentWithoutABodyWithType.UUID, "TEST_TRANS_ID")

	assert.NoError(err)
	assert.Equal(content{}, storedContent, "No content should be written when the content has no body")
}

func TestLiveBlogsWillBeWrittenDespiteNoBody(t *testing.T) {
	testContentWillBeWritten(t, liveBlog)
}

func TestContentPlaceholderWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, contentPlaceholder)
}

func TestVideoContentWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, videoContent)
}

func TestGraphicWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, graphicContent)
}

func TestAudioWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, audioContent)
}

func testContentWillBeWritten(t *testing.T, c content) {
	assert := assert.New(t)
	d := getDriverAndCheckClean(t, assert)
	s := getContentService(d)
	defer cleanDB(d, t, assert)

	assert.NoError(s.Write(c, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := s.Read(c.UUID, "TEST_TRANS_ID")
	assert.NoError(err)
	assert.NotEmpty(storedContent, "Failed to retrieve stored content")
	actualContent := storedContent.(content)

	assert.Equal(c.UUID, actualContent.UUID, "Failed to match UUID")
	assert.Equal(c.Title, actualContent.Title, "Failed to match Title")
}

func getDriverAndCheckClean(t *testing.T, assert *assert.Assertions) *cmneo4j.Driver {
	d := getNeoDriver(assert)
	cleanDB(d, t, assert)
	checkDbClean(d, t)
	return d
}

func getNeoDriver(assert *assert.Assertions) *cmneo4j.Driver {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	log := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	d, err := cmneo4j.NewDefaultDriver(url, log)
	assert.NoError(err, "Failed to connect to Neo4j")
	return d
}

func cleanDB(d *cmneo4j.Driver, t *testing.T, assert *assert.Assertions) {
	uuids := []string{
		contentUUID,
		conceptUUID,
		liveBlogUUID,
		noBodyContentUUID,
		noBodyInvalidTypeContentUUID,
		contentPlaceholderUUID,
		videoContentUUID,
		storyPackageUUID,
		contentPackageUUID,
		contentCollectionUUID,
		thingUUID,
		genericContentPackageUUID,
		graphicUUID,
		audioContentUUID,
	}

	qs := []*cmneo4j.Query{}
	for _, uuid := range uuids {
		qs = append(qs, &cmneo4j.Query{
			Cypher: `MATCH (t:Thing {uuid:{uuid}}) DETACH DELETE t`,
			Params: map[string]interface{}{
				"uuid": uuid,
			},
		})
	}

	err := d.Write(qs...)
	assert.NoError(err)
}

func writeContentPackageContainsRelation(d *cmneo4j.Driver, cpUUID string, UUID string, assert *assert.Assertions) {
	writeRelation := `
	MATCH (cp:ContentPackage {uuid:{cpUUID}}), (t {uuid:{UUID}})
	CREATE (cp)-[pred:CONTAINS]->(t)
	`

	qs := []*cmneo4j.Query{
		{
			Cypher: writeRelation,
			Params: map[string]interface{}{
				"cpUUID": cpUUID,
				"UUID":   UUID,
			},
		},
	}

	err := d.Write(qs...)
	assert.NoError(err)
}

func writeNodeWithLabels(d *cmneo4j.Driver, UUID string, labels string, t *testing.T, assert *assert.Assertions) {
	writeThingWithLabelsQuery := `CREATE (n:` + labels + `{uuid: {uuid}})`

	qs := []*cmneo4j.Query{
		{
			Cypher: writeThingWithLabelsQuery,
			Params: map[string]interface{}{
				"uuid": UUID,
			},
		},
	}

	err := d.Write(qs...)
	assert.NoError(err)
}

func writeRelationship(d *cmneo4j.Driver, contentID string, conceptID string, t *testing.T, assert *assert.Assertions) {
	annotateQuery := `
		MERGE (content:Thing{uuid:{contentId}})
		MERGE (concept:Thing) ON CREATE SET concept.uuid = {conceptId}
		MERGE (content)-[pred:SOME_PPREDICATE]->(concept)
		`

	qs := []*cmneo4j.Query{
		{
			Cypher: annotateQuery,
			Params: map[string]interface{}{
				"contentId": contentID,
				"conceptId": conceptID,
			},
		},
	}

	err := d.Write(qs...)
	assert.NoError(err)
}

func doesThingExist(uuid string, d *cmneo4j.Driver) (bool, error) {
	var result []struct {
		UUID string `json:"uuid,omitempty"`
	}
	query := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Thing {uuid:{uuid}})
			RETURN t.uuid as uuid`,
		Params: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &result,
	}
	err := d.Write(query)

	return len(result) > 0, err
}

func checkIsCuratedForRelationship(d *cmneo4j.Driver, spID string, assert *assert.Assertions) int {
	countQuery := `
		MATCH (t:Thing{uuid:{storyPackageId}})-[r:IS_CURATED_FOR]->(x)
		RETURN count(r) as c`

	var results []struct {
		Count int `json:"c"`
	}

	qs := &cmneo4j.Query{
		Cypher: countQuery,
		Params: map[string]interface{}{
			"storyPackageId": spID,
		},
		Result: &results,
	}

	err := d.Write(qs)
	assert.NoError(err)

	return results[0].Count
}

func checkContainsRelationship(d *cmneo4j.Driver, cpID string, assert *assert.Assertions) int {
	countQuery := `
		MATCH (t:Thing{uuid:{contentPackageId}})<-[r:CONTAINS]-(x)
		RETURN count(r) as c`

	var results []struct {
		Count int `json:"c"`
	}

	qs := &cmneo4j.Query{
		Cypher: countQuery,
		Params: map[string]interface{}{"contentPackageId": cpID},
		Result: &results,
	}

	err := d.Write(qs)
	assert.NoError(err)

	return results[0].Count
}

func checkDbClean(d *cmneo4j.Driver, t *testing.T) {
	assert := assert.New(t)

	var result []struct {
		UUID string `json:"t.uuid"`
	}

	checkGraph := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Thing) WHERE t.uuid in {uuids}
			RETURN t.uuid
		`,
		Params: map[string]interface{}{
			"uuids": []string{standardContent.UUID, conceptUUID},
		},
		Result: &result,
	}
	err := d.Write(checkGraph)
	assert.NoError(err)
	assert.Empty(result)
}

func getContentService(d *cmneo4j.Driver) Service {
	cs := NewContentService(d)
	cs.Initialise()
	return cs
}
