package content

import (
	"encoding/json"

	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/jmcvetta/neoism"
)

const (
	fsAuthority = "http://api.ft.com/system/FACTSET"
)

// CypherDriver - CypherDriver
type CypherDriver struct {
	cypherRunner neocypherrunner.CypherRunner
	indexManager neoutils.IndexManager
}

//NewCypherDriver instantiate driver
func NewCypherDriver(cypherRunner neocypherrunner.CypherRunner, indexManager neoutils.IndexManager) CypherDriver {
	return CypherDriver{cypherRunner, indexManager}
}

//Initialise initialisation of the indexes
func (pcd CypherDriver) Initialise() error {
	return neoutils.EnsureConstraints(pcd.indexManager, map[string]string{"Content": "uuid"})
}

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty
func (pcd CypherDriver) Check() error {
	return neoutils.Check(pcd.cypherRunner)
}

// Read - reads a content given a UUID
func (pcd CypherDriver) Read(uuid string) (interface{}, bool, error) {
	results := []struct {
		UUID          string `json:"uuid"`
		Title         string `json:"title"`
		PublishedDate string `json:"publishedDate"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content {uuid:{uuid}}) return n.uuid
		as uuid, n.title as title,
		n.publishedDate as publishedDate`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return content{}, false, err
	}

	if len(results) == 0 {
		return content{}, false, nil
	}

	result := results[0]

	c := content{
		UUID:          result.UUID,
		Title:         result.Title,
		PublishedDate: result.PublishedDate,
	}
	return c, true, nil
}

//Write - Writes a content node
func (pcd CypherDriver) Write(thing interface{}) error {
	c := thing.(content)

	params := map[string]interface{}{
		"uuid": c.UUID,
	}

	if c.Title != "" {
		params["title"] = c.Title
	}

	if c.PublishedDate != "" {
		params["publishedDate"] = c.PublishedDate
	}

	statement := `MERGE (n:Thing {uuid: {uuid}})
				set n={allprops}
				set n :Content`

	query := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"uuid":     c.UUID,
			"allprops": params,
		},
	}

	return pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})
}

//Delete - Deletes a content
func (pcd CypherDriver) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			REMOVE p:Content
			SET p={props}
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
			"props": map[string]interface{}{
				"uuid": uuid,
			},
		},
		IncludeStats: true,
	}

	removeNodeIfUnused := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			OPTIONAL MATCH (p)-[a]-(x)
			WITH p, count(a) AS relCount
			WHERE relCount = 0
			DELETE p
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

	s1, err := clearNode.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.ContainsUpdates && s1.LabelsRemoved > 0 {
		deleted = true
	}

	return deleted, err
}

// DecodeJSON - Decodes JSON into content
func (pcd CypherDriver) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := content{}
	err := dec.Decode(&c)
	return c, c.UUID, err

}

// Count - Returns a count of the number of content in this Neo instance
func (pcd CypherDriver) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content) return count(n) as c`,
		Result:    &results,
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}
