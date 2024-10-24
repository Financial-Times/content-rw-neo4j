package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger/v2"

	"github.com/Financial-Times/content-rw-neo4j/v3/policy"

	cmneo4j "github.com/Financial-Times/cm-neo4j-driver/v2"
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
	"LiveEvent":      true,
}

type Service struct {
	driver *cmneo4j.Driver
	agent  policy.Agent
	log    *logger.UPPLogger
}

// NewCypherDriver instantiate driver
func NewContentService(d *cmneo4j.Driver, a policy.Agent, l *logger.UPPLogger) Service {
	return Service{
		driver: d,
		agent:  a,
		log:    l,
	}
}

// Initialise ensures constraints on content uuid
func (cd Service) Initialise() error {
	err := cd.driver.EnsureConstraints(context.Background(), map[string]string{
		"Content": "uuid"})
	return err
}

// Check - Feeds into the Healthcheck and checks whether we can connect to Neo and that the datastore isn't empty and
// also checks if we are connected to leader/writable node
func (cd Service) Check() error {
	return cd.driver.VerifyConnectivity(context.Background())
}

// Read - reads a content given a UUID
func (cd Service) Read(uuid string, transID string) (interface{}, bool, error) {
	var results []struct {
		content
	}

	query := &cmneo4j.Query{
		Cypher: `OPTIONAL MATCH (n:Content {uuid: $uuid})
			OPTIONAL MATCH (sp:Thing)-[rel1:IS_CURATED_FOR]->(n)
			OPTIONAL MATCH (n)-[rel2:CONTAINS]->(cp:Thing)
			WITH n,sp,cp
			RETURN n.uuid as uuid,
				n.title as title,
				n.publishedDate as publishedDate,
				n.publication as publication,
				sp.uuid as storyPackage,
				cp.uuid as contentPackage`,
		Params: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}

	err := cd.driver.Read(context.Background(), query)

	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		return content{}, false, nil
	}

	if err != nil {
		return content{}, false, err
	}

	result := results[0]

	// At this point in time if the UUID is empty, it means that the content was not found
	if result.UUID == "" {
		return content{}, false, nil
	}

	contentItem := content{
		UUID:           result.UUID,
		Title:          result.Title,
		PublishedDate:  result.PublishedDate,
		Publication:    result.Publication,
		StoryPackage:   result.StoryPackage,
		ContentPackage: result.ContentPackage,
	}
	return contentItem, true, nil
}

// Write - Writes a content node
func (cd Service) Write(thing interface{}, transID string) error {
	c := thing.(content)

	// Letting through only articles (which have body), live blogs, content packages, graphics, videos and audios (which don't have a body)
	if c.Body == "" && !contentTypesWithNoBody[c.Type] {
		return nil
	}

	result, err := cd.agent.EvaluateSpecialContentPolicy(
		map[string]interface{}{
			"editorialDesk": c.EditorialDesk,
		},
	)
	if err != nil {
		return err
	}
	if result.IsSpecialContent {
		cd.log.Infof("Content with ID %s was marked as special content, it would not be persisted.", c.UUID)
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

	if len(c.Publication) != 0 {
		params["publication"] = c.Publication
	}

	deleteEntityRelationshipsQuery := &cmneo4j.Query{
		Cypher: `MATCH (t:Thing {uuid: $uuid})
				OPTIONAL MATCH (c:Thing)-[rel1:IS_CURATED_FOR]->(t)
				OPTIONAL MATCH (cp:Thing)<-[rel2:CONTAINS]-(t)
				DELETE rel1, rel2`,
		Params: map[string]interface{}{
			"uuid": c.UUID,
		},
	}

	queries := []*cmneo4j.Query{deleteEntityRelationshipsQuery}

	labels := getContentLabels(c)

	if c.StoryPackage != "" {
		addStoryPackageRelationQuery := addStoryPackageRelationQuery(c.UUID, c.StoryPackage)
		queries = append(queries, addStoryPackageRelationQuery)
	}

	if c.ContentPackage != "" {
		addContentPackageRelationQuery := addContentPackageRelationQuery(c.UUID, c.ContentPackage)
		queries = append(queries, addContentPackageRelationQuery)
	}

	query := fmt.Sprintf(`MERGE (n:Thing {uuid: $uuid})
		      set n=$allprops
		      set n %s`, labels)

	writeContentQuery := &cmneo4j.Query{
		Cypher: query,
		Params: map[string]interface{}{
			"uuid":     c.UUID,
			"allprops": params,
		},
	}

	queries = append(queries, writeContentQuery)
	err = cd.driver.Write(context.Background(), queries...)
	if err != nil {
		return err
	}
	return nil
}

func addStoryPackageRelationQuery(articleUUID, packageUUID string) *cmneo4j.Query {
	query := `MERGE(sp:Thing{uuid:$packageUuid})
			MERGE(c:Thing{uuid:$contentUuid})
			MERGE(c)<-[rel:IS_CURATED_FOR]-(sp)`

	return &cmneo4j.Query{
		Cypher: query,
		Params: map[string]interface{}{
			"packageUuid": packageUUID,
			"contentUuid": articleUUID,
		},
	}
}

func addContentPackageRelationQuery(articleUUID, packageUUID string) *cmneo4j.Query {
	query := `MERGE(cp:Thing{uuid:$packageUuid})
			MERGE(c:Thing{uuid:$contentUuid})
			MERGE(c)-[rel:CONTAINS]->(cp)`

	return &cmneo4j.Query{
		Cypher: query,
		Params: map[string]interface{}{
			"packageUuid": packageUUID,
			"contentUuid": articleUUID,
		},
	}
}

// Delete - Deletes a content item
func (cd Service) Delete(uuid string, transID string) (bool, error) {
	// "clearCollectionNode" query handles a specific case when
	// a Content Collection was deleted, which means its contents are removed
	// and the "ContentCollection" label was removed, but the node remains in Neo4j
	// with the label "Thing" only and still has a relation to a Content Package.
	// When a delete request occurs for the very same Content Package,
	// the related hanging node gets deleted by this query.

	// Check "content-collection-rw-neo4j" service for the the Content Collection deletion query.
	clearCollectionNode := &cmneo4j.Query{
		Cypher: `
			MATCH (p:ContentPackage {uuid: $uuid})-[rel:CONTAINS]->(cc:Thing)
			OPTIONAL MATCH (cc)-[rel]-()
			WITH cc, count(rel) AS relCount
			WHERE relCount = 1 AND NOT cc:ContentCollection
			DETACH DELETE cc
		`,
		Params: map[string]interface{}{
			"uuid": uuid,
		},
	}

	removeNode := &cmneo4j.Query{
		Cypher: `
			MATCH (p:Thing {uuid: $uuid})
			DETACH DELETE p
		`,
		Params: map[string]interface{}{
			"uuid": uuid,
		},
		IncludeSummary: true,
	}

	err := cd.driver.Write(context.Background(), clearCollectionNode)
	if err != nil {
		return false, err
	}
	// The queries should be executed in the specified order but `CypherBatch` does not guarantee order,
	// so we execute them in separate batches
	// dependency: if a CP is deleted before the first query is executed, there is no way to find the related node
	// left after a ContentCollections is deleted
	err = cd.driver.Write(context.Background(), removeNode)
	if err != nil {
		return false, err
	}

	s1, err := removeNode.Summary()
	if err != nil {
		return false, err
	}
	return s1.Counters().NodesDeleted() > 0, nil
}

// DecodeJSON - Decodes JSON into content
func (cd Service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	c := content{}
	err := dec.Decode(&c)
	return c, c.UUID, err
}

// Count - Returns a count of the number of content items in this Neo instance
func (cd Service) Count() (int, error) {
	var results []struct {
		Count int `json:"c"`
	}

	query := &cmneo4j.Query{
		Cypher: `MATCH (n:Content) return count(n) as c`,
		Result: &results,
	}

	err := cd.driver.Read(context.Background(), query)

	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}

func getContentLabels(c content) string {
	specialTypes := map[string]bool{
		"Content":        true,
		"ContentPackage": true,
		LiveBlogPackage:  true,
	}

	labels := []string{
		"Content",
	}

	if c.Type != "" && !specialTypes[c.Type] {
		labels = append(labels, c.Type)
	}

	if c.ContentPackage != "" {
		labels = append(labels, "ContentPackage")
		if c.Type == LiveBlogPackage {
			labels = append(labels, LiveBlogPackage)
		}
	}
	return ":" + strings.Join(labels, ":")
}
