package content

import (
	"encoding/json"
	"time"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	tid "github.com/Financial-Times/transactionid-utils-go"
	"github.com/jmcvetta/neoism"
)

const LiveBlogPackage = "LiveBlogPackage"
const LiveBlogPost = "LiveBlogPost"

var contentTypesWithNoBody = map[string]bool{
	"Content":        true,
	"Article":        true,
	"Video":          true,
	"Graphic":        true,
	"Audio":          true,
	"ContentPackage": true,
	LiveBlogPackage:  true,
	LiveBlogPost:     true,
}

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

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty and
// also checks if we are connected to leader/writable node
func (cd service) Check() error {
	writableErr := neoutils.CheckWritable(cd.conn)
	if writableErr != nil {
		return writableErr
	}

	return neoutils.Check(cd.conn)
}

// Read - reads a content given a UUID
func (cd service) Read(uuid string, transId string) (interface{}, bool, error) {
	var results []struct {
		content
	}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Content {uuid:{uuid}})
			OPTIONAL MATCH (sp:Thing)-[rel1:IS_CURATED_FOR]->(n)
			OPTIONAL MATCH (n)-[rel2:CONTAINS]->(cp:Thing)
			WITH n,sp,cp
			RETURN n.uuid as uuid,
				n.title as title,
				n.publishedDate as publishedDate,
				sp.uuid as storyPackage,
				cp.uuid as contentPackage`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
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
func (cd service) Write(thing interface{}, transId string) error {
	c := thing.(content)

	// Letting through only articles (which have body), live blogs, content packages, graphics, videos and audios (which don't have a body)
	if c.Body == "" && !contentTypesWithNoBody[c.Type] {
		logger.WithField(tid.TransactionIDKey, transId).
			Infof("There is no body with this content item therefore assuming is it not an Article: %v", c.UUID)
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
		logger.WithField(tid.TransactionIDKey, transId).
			Infof("There is a story package with uuid=%v attached to Article with uuid=%v", c.StoryPackage, c.UUID)
		addStoryPackageRelationQuery := addStoryPackageRelationQuery(c.UUID, c.StoryPackage)
		queries = append(queries, addStoryPackageRelationQuery)
	}

	if c.ContentPackage != "" {
		logger.WithField(tid.TransactionIDKey, transId).
			Infof("There is a content package with uuid=%v attached to Article with uuid=%v", c.ContentPackage, c.UUID)
		addContentPackageRelationQuery := addContentPackageRelationQuery(c.UUID, c.ContentPackage)
		queries = append(queries, addContentPackageRelationQuery)

		labels = labels + `:ContentPackage`
		if c.Type == LiveBlogPackage {
			labels = labels + `:` + LiveBlogPackage
		}
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
	err := cd.conn.CypherBatch(queries)
	if err != nil {
		logger.WithMonitoringEvent("SaveNeo4j", transId, c.Type).WithField("uuid", c.UUID).WithError(err).Errorf("error: the query could not be executed")
	} else {
		logger.WithMonitoringEvent("SaveNeo4j", transId, c.Type).WithField("uuid", c.UUID).Info("the query was successfully executed")
	}
	return err
}

func addStoryPackageRelationQuery(articleUUID, packageUUID string) *neoism.CypherQuery {
	statement := `MERGE(sp:Thing{uuid:{packageUuid}})
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
	statement := `MERGE(cp:Thing{uuid:{packageUuid}})
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
func (cd service) Delete(uuid string, transId string) (bool, error) {
	// "clearCollectionNode" query handles a specific case when
	// a Content Collection was deleted, which means its contents are removed
	// and the "ContentCollection" label was removed, but the node remains in Neo4j
	// with the label "Thing" only and still has a relation to a Content Package.
	// When a delete request occurs for the very same Content Package,
	// the related hanging node gets deleted by this query.

	// Check "content-collection-rw-neo4j" service for the the Content Collection deletion query.
	clearCollectionNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:ContentPackage {uuid: {uuid}})-[rel:CONTAINS]->(cc:Thing)
			OPTIONAL MATCH (cc)-[rel]-()
			WITH cc, count(rel) AS relCount
			WHERE relCount = 1 AND NOT cc:ContentCollection
			DETACH DELETE cc
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	removeNode := &neoism.CypherQuery{
		Statement: `
			MATCH (p:Thing {uuid: {uuid}})
			DETACH DELETE p
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		IncludeStats: true,
	}

	err := cd.conn.CypherBatch([]*neoism.CypherQuery{clearCollectionNode})
	if err != nil {
		logger.WithMonitoringEvent("SaveNeo4j", transId, "").WithField("uuid", uuid).WithError(err).Error("error: the extra delete query could not be executed")
		return false, err
	}
	// The queries should be executed in the specified order but `CypherBatch` does not guarantee order,
	// so we execute them in separate batches
	// dependency: if a CP is deleted before the first query is executed, there is no way to find the related node
	// left after a ContentCollections is deleted
	err = cd.conn.CypherBatch([]*neoism.CypherQuery{removeNode})
	if err != nil {
		logger.WithMonitoringEvent("SaveNeo4j", transId, "").WithField("uuid", uuid).WithError(err).Error("error: the delete query could not be executed")
		return false, err
	}

	s1, err := removeNode.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.NodesDeleted > 0 {
		deleted = true
	}

	logger.WithMonitoringEvent("SaveNeo4j", transId, "").WithField("uuid", uuid).Info("the delete query was successfully executed")
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

	var results []struct {
		Count int `json:"c"`
	}

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
