package content

import (

	"testing"
	"github.com/stretchr/testify/assert"
)


var expectedUpdatedContent = content{
	UUID: standardContent.UUID,
	Title:         "Another Title",
	PublishedDate: "1971-01-01T01:00:00.000Z",
	Body: "Ignored Body",
}

var shorterContent = content{
	UUID: standardContent.UUID,
	Body: "Shorter Content",
}

func TestWillUpdateProperties(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	assert.NoError(contentDriver.Write(expectedUpdatedContent), "Failed to write updated content")
	actualUpdatedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(actualUpdatedContent)
	assert.Equal(actualUpdatedContent.(content).Title, expectedUpdatedContent.Title, "Faiiled to update properties of the content node.")
	assert.Equal(actualUpdatedContent.(content).PublishedDate, expectedUpdatedContent.PublishedDate, "Faiiled to update properties of the content node.")
}

func TestWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	assert.NoError(contentDriver.Write(shorterContent), "Failed to write updated content")
	updatedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(updatedContent)
	assert.Empty(updatedContent.(content).Title,  "Faiiled to update properties of the content node.")
	assert.Empty(updatedContent.(content).PublishedDate,  "Faiiled to update properties of the content node.")
}

func TestUpdateWillRemoveRelsWithNoLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeClassifedByRelationships(db, standardContent.UUID, assert)
	assert.NoError(contentDriver.Write(expectedUpdatedContent), "Failed to write updated content")

	assert.Equal(0,
		checkAnyClassifedByRelationship(db, testBrandId, "", "v1",assert),
		"incorrect number, of is classified by relationships")
}

func TestUpdateWillRemoveRelsWithContnetLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeClassifedByRelationships(db, standardContent.UUID, assert)
	assert.NoError(contentDriver.Write(expectedUpdatedContent), "Failed to write updated content")

	assert.Equal(0,
		checkAnyClassifedByRelationship(db, testBrandId, "content", "v2",assert),
		"incorrect number, of is classified by relationships")
}

func TestUpdateWillNotRemoveRelsWithAnnotationsLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeClassifedByRelationships(db, standardContent.UUID, assert)
	assert.NoError(contentDriver.Write(expectedUpdatedContent), "Failed to write updated content")

	assert.Equal(1, checkAnyClassifedByRelationship(db, testBrandId, "annotations-v1", "v1", assert), "incorrect number, of is classified by relationships")
}


