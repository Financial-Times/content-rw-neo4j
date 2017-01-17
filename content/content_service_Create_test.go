// +build !jenkins

package content

import (
	"testing"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(assert)
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

func TestWriteCalculateEpochCorrectly(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	contentDriver.Write(standardContent)

	result := []struct {
		PublishedDateEpoch int `json:"t.publishedDateEpoch"`
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
	assert.Equal(3600, result[0].PublishedDateEpoch, "Epoch of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(assert)
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
	db := getDatabaseConnectionAndCheckClean(assert)
	contentDriver := getCypherDriver(db)
	defer deleteThingNodeAndAllRelationships(db, assert)

	assert.NoError(contentDriver.Write(contentWithoutABody), "Failed to write content")
	storedContent, _, err := contentDriver.Read(contentWithoutABody.UUID)

	assert.NoError(err)
	assert.Equal(content{}, storedContent, "No content should be written when the content has no body")
}


