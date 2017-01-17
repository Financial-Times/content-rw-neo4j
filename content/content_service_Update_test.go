package content

import (

	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)
	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	storedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent)

	var shorterContent = content{
		UUID: standardContent.UUID,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
}

func TestUpdateWillRemoveRelsWithNoLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeClassifedByRelationship(db, standardContent.UUID, conceptUUID, "", t, assert)
	storedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent)

	var shorterContent = content{
		UUID: standardContent.UUID,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
	assert.Equal(0, checkClassifedByRelationship(db, conceptUUID, "", t, assert), "incorrect number, of is classified by relationships")
}

func TestUpdateWillNotRemoveRelsWithNonContentLifeCycle(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	writeClassifedByRelationship(db, standardContent.UUID, conceptUUID, "annotations-v1", t, assert)
	storedContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedContent)

	var shorterContent = content{
		UUID: standardContent.UUID,
		Body: "Shorter body",
	}

	assert.NoError(contentDriver.Write(shorterContent), "Failed to write updated content")
	storedMinimalContent, _, err := contentDriver.Read(standardContent.UUID)

	assert.NoError(err)
	assert.NotEmpty(storedMinimalContent)
	assert.Equal(1, checkClassifedByRelationship(db, conceptUUID, "annotations-v1", t, assert), "incorrect number, of is classified by relationships")
}