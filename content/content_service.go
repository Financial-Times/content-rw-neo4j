package content

import (
	"encoding/json"
	"time"

	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

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
func (cd service) Check() error {
	return neoutils.Check(cd.conn)
}

// Read - reads a content given a UUID
func (cd service) Read(UUID string) (interface{}, bool, error) {
	results := []struct {
		content
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content {uuid:{uuid}})
			OPTIONAL MATCH (sp:Thing)-[rel1:IS_CURATED_FOR]->(n)
			OPTIONAL MATCH (n)-[rel2:CONTAINS]->(cp:Thing)
			WITH n,sp,cp
			return  n.uuid as uuid,
				n.title as title,
				n.publishedDate as publishedDate,
				sp.uuid as storyPackage,
				cp.uuid as contentPackage`,
		Parameters: map[string]interface{}{
			"uuid": UUID,
		},
		Result: &results,
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return content{}, false, err
	}

	if len(results) == 0 {
		return content{}, false, nil
	}

	result := results[0]

	contentItem := content{
		UUID:           result.UUID,
		Title:          result.Title,
		PublishedDate:  result.PublishedDate,
		StoryPackage:   result.StoryPackage,
		ContentPackage: result.ContentPackage,
	}
	return contentItem, true, nil
}

//Write - Writes a content node
func (cd service) Write(thing interface{}) error {
	c := thing.(content)

	// Letting through only articles (which have body), content packages and videos
	if c.Body == "" && c.Type != "Content" && c.Type != "Video" {
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
				OPTIONAL MATCH (c:Thing)-[rel1:IS_CURATED_FOR]->(t)
				OPTIONAL MATCH (cp:Thing)<-[rel2:CONTAINS]-(t)
				DELETE rel1, rel2`,
		Parameters: map[string]interface{}{
			"uuid": c.UUID,
		},
	}

	queries := []*neoism.CypherQuery{deleteEntityRelationshipsQuery}

	labels := `:Content`

	if c.StoryPackage != "" {
		log.Infof("There is a story package with uuid=%v attached to Article with uuid=%v", c.StoryPackage, c.UUID)
		addStoryPackageRelationQuery := addStoryPackageRelationQuery(c.UUID, c.StoryPackage)
		queries = append(queries, addStoryPackageRelationQuery)
	}

	if c.ContentPackage != "" {
		log.Infof("There is a content package with uuid=%v attached to Article with uuid=%v", c.ContentPackage, c.UUID)
		addContentPackageRelationQuery := addContentPackageRelationQuery(c.UUID, c.ContentPackage)
		queries = append(queries, addContentPackageRelationQuery)
		labels = labels + `:ContentPackage`
	}

	statement := `MERGE (n:Thing {uuid: {uuid}})
		      set n={allprops}
		      set n ` + labels

	writeContentQuery := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"uuid":     c.UUID,
			"allprops": params,
		},
	}

	queries = append(queries, writeContentQuery)
	return cd.conn.CypherBatch(queries)
}

func addStoryPackageRelationQuery(articleUUID, packageUUID string) *neoism.CypherQuery {
	statement := `	MERGE(sp:Thing{uuid:{packageUuid}})
			MERGE(c:Thing{uuid:{contentUuid}})
			MERGE(c)<-[rel:IS_CURATED_FOR]-(sp)`

	query := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"packageUuid": packageUUID,
			"contentUuid": articleUUID,
		},
	}
	return query
}

func addContentPackageRelationQuery(articleUUID, packageUUID string) *neoism.CypherQuery {
	statement := `	MERGE(cp:Thing{uuid:{packageUuid}})
			MERGE(c:Thing{uuid:{contentUuid}})
			MERGE(c)-[rel:CONTAINS]->(cp)`

	query := &neoism.CypherQuery{
		Statement: statement,
		Parameters: map[string]interface{}{
			"packageUuid": packageUUID,
			"contentUuid": articleUUID,
		},
	}
	return query
}

//Delete - Deletes a content
func (cd service) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			OPTIONAL MATCH (sp:Thing)-[rel1:IS_CURATED_FOR]->(p)
			OPTIONAL MATCH (p)-[rel2:CONTAINS]->(contained_cp:Thing)
			OPTIONAL MATCH (containing_cp:Thing)-[rel3:CONTAINS]->(p)
			REMOVE p:Content
			DELETE rel1, rel2, rel3
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

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

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
func (cd service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := content{}
	err := dec.Decode(&c)
	return c, c.UUID, err

}

// Count - Returns a count of the number of content in this Neo instance
func (cd service) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content) return count(n) as c`,
		Result:    &results,
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}
