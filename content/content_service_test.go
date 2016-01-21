// +build !jenkins

package content

import (
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

var contentDriver baseftrwapp.Service

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"

	contentDriver = getContentCypherDriver(t)

	contentToDelete := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z"}

	assert.NoError(contentDriver.Write(contentToDelete), "Failed to write content")

	found, err := contentDriver.Delete(uuid)
	assert.True(found, "Didn't manage to delete content for uuid %", uuid)
	assert.NoError(err, "Error deleting content for uuid %s", uuid)

	p, found, err := contentDriver.Read(uuid)

	assert.Equal(content{}, p, "Found content %s who should have been deleted", p)
	assert.False(found, "Found content for uuid %s who should have been deleted", uuid)
	assert.NoError(err, "Error trying to find content for uuid %s", uuid)
}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	contentDriver = getContentCypherDriver(t)

	contentToWrite := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z"}

	assert.NoError(contentDriver.Write(contentToWrite), "Failed to write content")

	readContentForUUIDAndCheckFieldsMatch(t, uuid, contentToWrite)

	cleanUp(t, uuid)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	contentDriver = getContentCypherDriver(t)

	contentToWrite := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z"}

	assert.NoError(contentDriver.Write(contentToWrite), "Failed to write content")

	readContentForUUIDAndCheckFieldsMatch(t, uuid, contentToWrite)

	cleanUp(t, uuid)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	contentDriver = getContentCypherDriver(t)

	contentToWrite := content{UUID: uuid, Title: "TestContent", PublishedDate: "1970-01-01T01:00:00.000Z"}

	assert.NoError(contentDriver.Write(contentToWrite), "Failed to write content")

	readContentForUUIDAndCheckFieldsMatch(t, uuid, contentToWrite)

	cleanUp(t, uuid)
}

func readContentForUUIDAndCheckFieldsMatch(t *testing.T, uuid string, expectedContent content) {
	assert := assert.New(t)
	storedContent, found, err := contentDriver.Read(uuid)

	assert.NoError(err, "Error finding content for uuid %s", uuid)
	assert.True(found, "Didn't find content for uuid %s", uuid)
	assert.Equal(expectedContent, storedContent, "content should be the same")
}

func getContentCypherDriver(t *testing.T) CypherDriver {
	assert := assert.New(t)
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return NewCypherDriver(neoutils.StringerDb{db}, db)
}

func cleanUp(t *testing.T, uuid string) {
	assert := assert.New(t)
	found, err := contentDriver.Delete(uuid)
	assert.True(found, "Didn't manage to delete content for uuid %", uuid)
	assert.NoError(err, "Error deleting content for uuid %s", uuid)
}
