package content

import (

	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"fmt"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
)

var contentWithoutABody = content{
	UUID:  "noBodyContentUuid",
	Title: "Missing Body",
}

var standardContent = content{
	UUID:          "contentUUID",
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
}

var contentDriver baseftrwapp.Service

const (
	testBrandId = "testBrandId"
	FTBrandId = "FTBrandId"
)

func writeClassifedByRelationships(db neoutils.NeoConnection, contentId string, assert *assert.Assertions) {

	var qs []*neoism.CypherQuery

	var statement = `
		MERGE (content:Thing {uuid:{contentId}})
		MERGE (concept:Thing { uuid :{conceptId} })
	`
	var rel_with_lifecycle = `MERGE (content)-[pred:IS_CLASSIFIED_BY { platformVersion:{platformVersion}, lifecycle: {lifecycle}}]->(concept)`

	var rel_without_lifecycle = `MERGE (content)-[pred:IS_CLASSIFIED_BY { platformVersion:"v1" }]->(concept)`

	qs = []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("%s %s", statement, rel_without_lifecycle),
			Parameters: neoism.Props{
				"contentId": contentId,
				"conceptId": testBrandId,
			},
		},
		{
			Statement:  fmt.Sprintf("%s %s", statement, rel_with_lifecycle),
			Parameters: map[string]interface{}{
				"contentId": contentId,
				"conceptId": testBrandId,
				"platformVersion":"v2",
				"lifecycle": "content",
			},

		},
		{
			Statement:  fmt.Sprintf("%s %s", statement, rel_with_lifecycle),
			Parameters: neoism.Props{
				"contentId": contentId,
				"conceptId": FTBrandId,
				"platformVersion":"v2",
				"lifecycle": "content",
			},
		},
		{
			Statement: fmt.Sprintf("%s %s", statement, rel_with_lifecycle),
			Parameters: neoism.Props{
				"contentId": contentId,
				"conceptId": testBrandId,
				"platformVersion":"v1",
				"lifecycle": "annotations-v1",

			},
		},
		{
			Statement: fmt.Sprintf("%s %s", statement, rel_with_lifecycle),
			Parameters: neoism.Props{
				"contentId": contentId,
				"conceptId": FTBrandId,
				"platformVersion":"v1",
				"lifecycle": "annotations-v1",

			},
		},


	}


	writeBrands(db, assert)

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func writeBrands(db neoutils.NeoConnection, assert *assert.Assertions) {
	var statement = `
	merge (brand:Thing:Concept:Brand {aliases: "Test Brand", prefLabel: "Test Brand", uuid: {testBrandId}})
	<-[br:IDENTIFIES]-(bi:Identifier:UPPIdentifier {value: "test-brand-uppIdentifier"})
	merge (parent:Thing:Concept:Brand {aliases: "FT Brand", prefLabel: "FT Brand", uuid: {FTBrandId}})
	<-[pr:IDENTIFIES]-(pi:Identifier:UPPIdentifier {value: "parent-brand-uppIdentifier"})
	merge (brand)-[r:HAS_PARENT]->(parent) `

	var qs = []*neoism.CypherQuery{
		{
			Statement:  statement,
			Parameters: neoism.Props{"testBrandId": testBrandId, "FTBrandId": FTBrandId},
		},
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)

}

func deleteThingNodeAndAllRelationships(db neoutils.NeoConnection, assert *assert.Assertions) {
	var qs []*neoism.CypherQuery

	//match (content:Thing {uuid:"contentUUID"})-[*0..]-(concepts) detach delete content, concepts
	var statement = `match (content:Thing {uuid:{contentId}})-[*0..]-(concepts) detach delete content, concepts`

	qs = []*neoism.CypherQuery{
		{
			Statement:  statement,
			Parameters: map[string]interface{}{
				"contentId": standardContent.UUID,
			},
		},
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkAnyClassifedByRelationship(db neoutils.NeoConnection, conceptId string, lifecycle string, platformVersion string, assert *assert.Assertions) int {

	results := []struct {
		Count int `json:"c"`
	}{}

	//var without_lifecycle = `MATCH (t:Thing{uuid:{conceptId}})
	//			-[r:IS_CLASSIFIED_BY {platformVersion:{platformVersion}}]-(x)
	//		MATCH (t)<-[:IDENTIFIES]-(s:Identifier:UPPIdentifier)
	//		RETURN count(r) as c`

	var countQuery = `	MATCH (t:Thing{uuid:{conceptId}})
				-[r:IS_CLASSIFIED_BY {platformVersion:{platformVersion}, lifecycle: {lifecycle}}]-(x)
			MATCH (t)<-[:IDENTIFIES]-(s:Identifier:UPPIdentifier)
			RETURN count(r) as c`

	qs := &neoism.CypherQuery{
		Statement:  countQuery,
		Parameters: neoism.Props{
			"conceptId": conceptId,
			"platformVersion": platformVersion,
			"lifecycle": lifecycle,
		},
		Result:     &results,
	}

	err := db.CypherBatch([]*neoism.CypherQuery{qs})
	assert.NoError(err)

	return results[0].Count
}

func findThings(uuid string, label string, db neoutils.NeoConnection) (string, error)  {

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

	if len(result) == 0 {
		return "", nil
	}
	fmt.Println(result[0].UUID)
	return result[0].UUID, err
}