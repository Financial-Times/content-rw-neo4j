package content
import (

	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestDeletedContentWithNoRelationshipsRemovedCompletely(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver := getCypherDriver(db)
	assert.NoError(contentDriver.Write(standardContent), "Failed to write content")
	deleted, err := contentDriver.Delete(standardContent.UUID)
	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)
	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)

	c, deleted, err := contentDriver.Read(standardContent.UUID)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	id, err := findThings(standardContent.UUID, "Thing", db)
	assert.NoError(err)
	assert.Empty(id, fmt.Sprintf("There should not be a thing with uuid %s after a content node with no relationshipd was deleted.",id) )
}

func TestDeletedContentWithRelationshipsRemainsAThing(t *testing.T) {

	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver := getCypherDriver(db)
	assert.NoError(contentDriver.Write(standardContent))
	writeClassifedByRelationships(db, standardContent.UUID, assert)

	deleted, err := contentDriver.Delete(standardContent.UUID)

	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)

	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)
	c, deleted, err := contentDriver.Read(standardContent.UUID)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	id, err := findThings(standardContent.UUID, "Thing", db)
	assert.NoError(err)
	assert.Equal(id, standardContent.UUID,
	fmt.Sprintf("There should still be a thing with uuid %s after a content node with relationshipd was deleted.",id))
}


func TestDeleteContentWillNotDeleteConcepts(t *testing.T) {

	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver := getCypherDriver(db)
	assert.NoError(contentDriver.Write(standardContent))
	writeClassifedByRelationships(db, standardContent.UUID, assert)

	deleted, err := contentDriver.Delete(standardContent.UUID)
	//
	assert.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)
	//
	assert.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)
	c, deleted, err := contentDriver.Read(standardContent.UUID)

	assert.Equal(content{}, c, "Found content %s who should have been deleted", c)
	assert.False(deleted, "Found content for uuid %s who should have been deleted", standardContent.UUID)
	assert.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	id, err := findThings(testBrandId, "Concept", db)
	assert.NoError(err)
	assert.Equal(id, testBrandId,
		fmt.Sprintf("There should still be a thing with uuid %s after a content node with relationshipd was deleted.",id))
}




