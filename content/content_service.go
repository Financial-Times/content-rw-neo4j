package content

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

var uuidExtractRegex = regexp.MustCompile(".*/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$")

// CypherDriver - CypherDriver
type service struct {
	conn neoutils.NeoConnection
}

//NewCypherDriver instantiate driver
func NewCypherContentService(cypherRunner neoutils.NeoConnection) service {
	return service{cypherRunner}
}

//Initialise ensure constraint on content uuid
func (cd service) Initialise() error {

	return cd.conn.EnsureConstraints(map[string]string{
		"Content": "uuid"})
}

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty
func (pcd service) Check() error {
	return neoutils.Check(pcd.conn)
}

// Read - reads a content given a UUID
func (pcd service) Read(uuid string) (interface{}, bool, error) {
	results := []struct {
		content
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content {uuid:{uuid}})
			OPTIONAL MATCH (n)-[rel:IS_CLASSIFIED_BY]->(b:Thing)
				WHERE rel.lifecycle IS NULL
				OR rel.lifecycle = "content"
			OPTIONAL MATCH (sp:Thing)-[rel2:IS_CURATED_FOR]->(n)
			WITH n,collect({id:b.uuid}) as brands, sp
			return n.uuid as uuid, n.title as title, n.publishedDate as publishedDate, sp.uuid as storyPackage`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return content{}, false, err
	}

	if len(results) == 0 {
		return content{}, false, nil
	}

	result := results[0]

	contentItem := content{
		UUID:          result.UUID,
		Title:         result.Title,
		PublishedDate: result.PublishedDate,
		StoryPackage:  result.StoryPackage,
	}
	return contentItem, true, nil
}

//Write - Writes a content node
func (pcd service) Write(thing interface{}) error {
	c := thing.(content)

	// Only Articles have a body
	if c.Body == "" {
		log.Infof("There is no body with this content item therefore assuming is it not an Article: %v", c.UUID)
		return nil
	}

	params := map[string]interface{}{
		"uuid": c.UUID,
	}

	if c.Title != "" {
		params["title"] = c.Title
		params["prefLabel"] = c.Title
	}

	if c.PublishedDate != "" {
		params["publishedDate"] = c.PublishedDate
		datetimeEpoch, err := time.Parse(time.RFC3339, c.PublishedDate)

		if err != nil {
			return err
		}

		params["publishedDateEpoch"] = datetimeEpoch.Unix()
	}

	deleteEntityRelationshipsQuery := &neoism.CypherQuery{
		Statement: `MATCH (t:Thing {uuid:{uuid}})
				OPTIONAL MATCH (c:Thing)-[rel2:IS_CURATED_FOR]->(t)
				DELETE rel2`,
		Parameters: map[string]interface{}{
			"uuid": c.UUID,
		},
	}

	queries := []*neoism.CypherQuery{deleteEntityRelationshipsQuery}

	if c.StoryPackage != "" {
		log.Infof("There is a story package with uuid=%v attached to Article with uuid=%v", c.StoryPackage, c.UUID)
		addStoryPackageRelationQuery := addPackageRelationQuery(c.UUID, c.StoryPackage)
		queries = append(queries, addStoryPackageRelationQuery)
	}

	statement := `MERGE (n:Thing {uuid: {uuid}})
		      set n={allprops}
		      set n :Content`

	writeContentQuery := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"uuid":     c.UUID,
			"allprops": params,
		},
	}

	queries = append(queries, writeContentQuery)
	return pcd.conn.CypherBatch(queries)
}


func addPackageRelationQuery(articleUuid, packageUuid string) *neoism.CypherQuery {
	statement := `	MERGE(sp:Thing{uuid:{packageUuid}})
			MERGE(c:Thing{uuid:{contentUuid}})
			MERGE(c)<-[rel:IS_CURATED_FOR]-(sp)`

	query := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"packageUuid": packageUuid,
			"contentUuid": articleUuid,
		},
	}
	return query
}

//Delete - Deletes a content
func (pcd service) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			OPTIONAL MATCH (sp:Thing)-[rel:IS_CURATED_FOR]->(p)
			REMOVE p:Content
			DELETE rel
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

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

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
func (pcd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := content{}
	err := dec.Decode(&c)
	return c, c.UUID, err

}

// Count - Returns a count of the number of content in this Neo instance
func (pcd service) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content) return count(n) as c`,
		Result:    &results,
	}

	err := pcd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}

