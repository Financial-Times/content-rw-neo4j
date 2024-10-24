//go:build integration
// +build integration

package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Financial-Times/opa-client-go"

	"github.com/Financial-Times/content-rw-neo4j/v3/policy"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"

	cmneo4j "github.com/Financial-Times/cm-neo4j-driver/v2"
	"github.com/Financial-Times/go-logger/v2"
)

const (
	contentUUID                  = "ce3f2f5e-33d1-4c36-89e3-51aa00fd5660"
	conceptUUID                  = "412e4ca3-f8d5-4456-8606-064c1dba3c45"
	liveBlogUUID                 = "1520b6b9-d466-49a0-b3ec-894b72338e7d"
	noBodyContentUUID            = "6440aa4a-1298-4a49-9346-78d546bc0229"
	noBodyInvalidTypeContentUUID = "1674d8b6-f3b2-4f18-9f3b-e28bcf5553a0"
	contentPlaceholderUUID       = "ed2d9fc2-b515-4f7d-8b4e-3b0c1fa90986"
	videoContentUUID             = "41bb9444-e3cf-46d4-8182-6702844dc5c1"
	storyPackageUUID             = "3b08c76c-7479-461d-9f0e-a4e92dca56f7"
	contentPackageUUID           = "45163790-eec9-11e6-abbc-ee7d9c5b3b90"
	contentCollectionUUID        = "cc65c43a-fe4e-4315-854b-9b82435be036"
	thingUUID                    = "ebcfe37d-9a70-4c8b-bf01-1feee4dff4b7"
	genericContentPackageUUID    = "27cfe7eb-549d-4d51-9cfd-98ea887a571c"
	graphicUUID                  = "087b42c2-ac7f-40b9-b112-98b3a7f9cd72"
	audioContentUUID             = "128cfcf4-c394-4e71-8c65-198a675acf53"
	liveEventUUID                = "23531906-9f98-45c7-a9db-d05bdb72eeaf"
	unExistentContent            = "ce3f2f5e-33d1-4c36-89e3-555555555555"
	defaultPolicy                = `
	package content_rw_neo4j.special_content
	
	default is_special_content := false
	`
	specialContentPolicy = `
	package content_rw_neo4j.special_content

	import future.keywords.if

	default is_special_content := false

	is_special_content := true if {
		input.editorialDesk == "/FT/Professional/Central Banking"
	}
	`
)

var contentWithoutABody = content{
	UUID:  noBodyContentUUID,
	Title: "Missing Body",
}

var contentPlaceholder = content{
	UUID:  contentPlaceholderUUID,
	Title: "Missing Body",
	Type:  "Content",
}

var liveBlog = content{
	UUID:  liveBlogUUID,
	Title: "Live blog",
	Type:  "Article",
}

var contentWithoutABodyWithType = content{
	UUID:  noBodyInvalidTypeContentUUID,
	Title: "Missing Body",
	Type:  "Image",
}

var videoContent = content{
	UUID:  videoContentUUID,
	Title: "Missing Body",
	Type:  "Video",
}

var graphicContent = content{
	UUID:  graphicUUID,
	Title: "Missing Body",
	Type:  "Graphic",
}

var audioContent = content{
	UUID:  audioContentUUID,
	Title: "Missing Body",
	Type:  "Audio",
}

var standardContent = content{
	UUID:          contentUUID,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
	StoryPackage:  storyPackageUUID,
	EditorialDesk: "/FT/Standard Content",
	Publication:   []string{"8e6c705e-1132-42a2-8db0-c295e29e8658"},
}

var standardContentPackage = content{
	UUID:           contentUUID,
	Title:          "Content Title",
	PublishedDate:  "1970-01-01T01:00:00.000Z",
	Body:           "Some body",
	StoryPackage:   storyPackageUUID,
	ContentPackage: contentPackageUUID,
}

var genericContentPackage = content{
	UUID:           contentUUID,
	Title:          "Content Title",
	PublishedDate:  "1970-01-01T01:00:00.000Z",
	ContentPackage: genericContentPackageUUID,
	Type:           "ContentPackage",
}

var liveBlogPackage = content{
	UUID:           contentUUID,
	Title:          "Content Title",
	PublishedDate:  "1970-01-01T01:00:00.000Z",
	ContentPackage: genericContentPackageUUID,
	Type:           "LiveBlogPackage",
}

var liveEventContent = content{
	UUID: liveEventUUID,
	Type: "LiveEvent",
}

var shorterContent = content{
	UUID: contentUUID,
	Body: "With No Publish Date and No Title",
}

var updatedContent = content{
	UUID:          contentUUID,
	Title:         "New Title",
	PublishedDate: "1999-12-12T01:00:00.000Z",
	Body:          "Doesn't matter",
}

var specialContent = content{
	UUID:          contentUUID,
	Title:         "Content Title",
	PublishedDate: "1970-01-01T01:00:00.000Z",
	Body:          "Some body",
	StoryPackage:  storyPackageUUID,
	EditorialDesk: "/FT/Professional/Central Banking",
}

var sortStringSlicesDesc = cmpopts.SortSlices(func(a, b string) bool { return a < b })

func TestGetContentLabels(t *testing.T) {
	tests := map[string]struct {
		Content  content
		Expected string
	}{
		"No body content": {
			Content:  contentWithoutABody,
			Expected: ":Content",
		},
		"Placeholder": {
			Content:  contentPlaceholder,
			Expected: ":Content",
		},
		"Live blog": {
			Content:  liveBlog,
			Expected: ":Content:Article",
		},
		"Content Package": {
			Content:  standardContentPackage,
			Expected: ":Content:ContentPackage",
		},
		"generic Content Package": {
			Content:  genericContentPackage,
			Expected: ":Content:ContentPackage",
		},
		"live blog package": {
			Content:  liveBlogPackage,
			Expected: ":Content:ContentPackage:LiveBlogPackage",
		},
		"live event": {
			Content:  liveEventContent,
			Expected: ":Content:LiveEvent",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := getContentLabels(test.Content)
			if actual != test.Expected {
				t.Errorf("expected: '%s', got '%s'", test.Expected, actual)
			}
		})
	}
}

func TestGetContentNotFound(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	content, found, err := s.Read(unExistentContent, "TEST_TRANS_ID")
	asst.Empty(content, "Content should be empty")
	asst.False(found, "Content should not be found")
	asst.NoError(err, "Error should be nil")
}

func TestDeleteWithNoRelsIsDeleted(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write content")

	deleted, err := s.Delete(shorterContent.UUID, "TEST_TRANS_ID")
	asst.True(deleted, "Didn't manage to delete content for uuid %s", shorterContent.UUID)
	asst.NoError(err, "Error deleting content for uuid %s", shorterContent.UUID)

	c, deleted, err := s.Read(shorterContent.UUID, "TEST_TRANS_ID")

	asst.Equal(content{}, c, "Found content %s who should have been deleted", c)
	asst.False(
		deleted,
		"Found content for uuid %s who should have been deleted",
		standardContent.UUID,
	)
	asst.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)
}

func TestDeleteWithRelsIsDeleted(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")
	writeRelationship(d, standardContent.UUID, conceptUUID, asst)

	deleted, err := s.Delete(standardContent.UUID, "TEST_TRANS_ID")
	asst.NoError(err, "Error deleting content for uuid %s", standardContent.UUID)
	asst.True(deleted, "Didn't manage to delete content for uuid %s", standardContent.UUID)

	c, found, err := s.Read(standardContent.UUID, "TEST_TRANS_ID")

	asst.Equal(content{}, c, "Found content %s who should have been deleted", c)
	asst.False(
		found,
		"Found content for uuid %s who should have been deleted",
		standardContent.UUID,
	)
	asst.NoError(err, "Error trying to find content for uuid %s", standardContent.UUID)

	exists, err := doesThingExist(standardContent.UUID, d)
	asst.NoError(err)
	asst.False(exists, "Thing should not exist for deleted content with relations")
}

func TestDeleteContentPackageIsDeletedAttachedContentCollectionRemains(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(genericContentPackage, "TEST_TRANS_ID"), "Failed to write content package")
	writeNodeWithLabels(d, contentCollectionUUID, "Thing:Content:ContentCollection", asst)
	writeContentPackageContainsRelation(d, genericContentPackage.UUID, contentCollectionUUID, asst)

	deleted, err := s.Delete(genericContentPackage.UUID, "TEST_TRANS_ID")
	asst.NoError(
		err,
		"Error deleting Content Package for uuid %contentService",
		genericContentPackage.UUID,
	)
	asst.True(
		deleted,
		"Didn't manage to delete Content Package for uuid %contentService",
		genericContentPackage.UUID,
	)

	c, found, err := s.Read(genericContentPackage.UUID, "TEST_TRANS_ID")

	asst.Equal(
		content{},
		c,
		"Found Content Package %contentService who should have been deleted",
		c,
	)
	asst.False(
		found,
		"Found Content Package for uuid %contentService who should have been deleted",
		genericContentPackage.UUID,
	)
	asst.NoError(
		err,
		"Error trying to find Content Package for uuid %s",
		genericContentPackage.UUID,
	)

	exists, err := doesThingExist(genericContentPackage.UUID, d)
	asst.NoError(err)
	asst.False(exists, "Thing should not exist for deleted Content Package")

	existsCC, err := doesThingExist(contentCollectionUUID, d)
	asst.NoError(err)
	asst.True(existsCC, "Content Collection should exist")
}

func TestDeleteContentPackageIsDeletedAttachedNodeIsAlsoDeleted(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(genericContentPackage, "TEST_TRANS_ID"), "Failed to write content package")
	writeNodeWithLabels(d, thingUUID, "Thing", asst)
	writeContentPackageContainsRelation(d, genericContentPackage.UUID, thingUUID, asst)

	deleted, err := s.Delete(genericContentPackage.UUID, "TEST_TRANS_ID")
	asst.NoError(
		err,
		"Error deleting Content Package for uuid %contentService",
		genericContentPackage.UUID,
	)
	asst.True(
		deleted,
		"Didn't manage to delete Content Package for uuid %contentService",
		genericContentPackage.UUID,
	)

	c, found, err := s.Read(genericContentPackage.UUID, "TEST_TRANS_ID")

	asst.Equal(
		content{},
		c,
		"Found Content Package %contentService who should have been deleted",
		c,
	)
	asst.False(
		found,
		"Found Content Package for uuid %contentService who should have been deleted",
		genericContentPackage.UUID,
	)
	asst.NoError(
		err,
		"Error trying to find Content Package for uuid %contentService",
		genericContentPackage.UUID,
	)

	exists, err := doesThingExist(genericContentPackage.UUID, d)
	asst.NoError(err)
	asst.False(exists, "Thing should not exist for deleted Content Package")

	existsThing, err := doesThingExist(thingUUID, d)
	asst.NoError(err)
	asst.False(existsThing, "Thing related to deleted Content Package should not exist")
}

func TestCreateAllValuesPresent(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := s.Read(standardContent.UUID, "TEST_TRANS_ID")
	asst.NoError(err)
	asst.NotEmpty(storedContent, "Failed to retrieve stored content")
	actualContent := storedContent.(content)

	asst.Equal(standardContent.UUID, actualContent.UUID, "Failed to match UUID")
	asst.Equal(standardContent.Title, actualContent.Title, "Failed to match Title")
	asst.Equal(
		standardContent.PublishedDate,
		actualContent.PublishedDate,
		"Failed to match PublishedDate",
	)
	asst.Empty(actualContent.Body, "Body should not have been stored")
	asst.Equal(standardContent.StoryPackage, actualContent.StoryPackage)
	asst.Equal(standardContent.ContentPackage, actualContent.ContentPackage)
	asst.Equal(standardContent.Publication, actualContent.Publication)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := s.Read(shorterContent.UUID, "TEST_TRANS_ID")

	asst.NoError(err)
	asst.Empty(storedContent.(content).PublishedDate)
	asst.Empty(storedContent.(content).Title)
}

func TestWillUpdateProperties(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := s.Read(contentUUID, "TEST_TRANS_ID")

	asst.NoError(err)
	asst.Equal(storedContent.(content).Title, standardContent.Title)
	asst.Equal(storedContent.(content).PublishedDate, standardContent.PublishedDate)

	asst.NoError(s.Write(updatedContent, "TEST_TRANS_ID"), "Failed to write updated content")
	storedContent, _, err = s.Read(contentUUID, "TEST_TRANS_ID")

	asst.NoError(err)
	asst.Equal(
		storedContent.(content).Title,
		updatedContent.Title,
		"Should have updated Title but it is still present",
	)
	asst.Equal(
		storedContent.(content).PublishedDate,
		updatedContent.PublishedDate,
		"Should have updated PublishedDate but it is still present",
	)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(standardContentPackage, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := s.Read(contentUUID, "TEST_TRANS_ID")

	asst.NoError(err)
	asst.NotEmpty(storedContent.(content).Title)
	asst.NotEmpty(storedContent.(content).PublishedDate)
	asst.Equal(
		1,
		checkIsCuratedForRelationship(d, storyPackageUUID, asst),
		"incorrect number of isCuratedFor relationships",
	)
	asst.Equal(
		1,
		checkContainsRelationship(d, contentPackageUUID, asst),
		"incorrect number of contains relationships",
	)

	asst.NoError(s.Write(shorterContent, "TEST_TRANS_ID"), "Failed to write updated content")
	storedContent, _, err = s.Read(contentUUID, "TEST_TRANS_ID")

	asst.NoError(err)
	asst.NotEmpty(storedContent, "Failed to rеtriеve updated content")
	asst.Empty(
		storedContent.(content).Title,
		"Update should have removed Title but it is still present",
	)
	asst.Empty(
		storedContent.(content).PublishedDate,
		"Update should have removed PublishedDate but it is still present",
	)
	asst.Equal(
		0,
		checkIsCuratedForRelationship(d, storyPackageUUID, asst),
		"incorrect number of isCuratedFor relationships",
	)
	asst.Equal(
		0,
		checkContainsRelationship(d, contentPackageUUID, asst),
		"incorrect number of contains relationships",
	)
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	uuid := standardContent.UUID
	contentReceived := content{
		UUID:          uuid,
		Title:         "TestContent",
		PublishedDate: "1970-01-01T01:00:00.000Z",
		Body:          "Some Test text",
	}
	err := s.Write(contentReceived, "TEST_TRANS_ID")
	asst.NoError(err)

	var result []struct {
		PublishedDateEpoc int `json:"t.publishedDateEpoch"`
	}

	getEpochQuery := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Content {uuid: $uuid}) RETURN t.publishedDateEpoch
			`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(context.Background(), getEpochQuery)
	asst.NoError(err)
	asst.Equal(3600, result[0].PublishedDateEpoc, "Epoc of 1970-01-01T01:00:00.000Z should be 3600")
}

func TestWritePrefLabelIsAlsoWrittenAndIsEqualToTitle(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	err := s.Write(standardContent, "TEST_TRANS_ID")
	asst.NoError(err)

	var result []struct {
		PrefLabel string `json:"t.prefLabel"`
	}

	getPrefLabelQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid: $uuid}) RETURN t.prefLabel
				`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(context.Background(), getPrefLabelQuery)
	asst.NoError(err)
	asst.Equal(standardContent.Title, result[0].PrefLabel, "PrefLabel should be 'Content Title'")
}

func TestWriteNodeLabelsAreWrittenForContent(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	err := s.Write(standardContent, "TEST_TRANS_ID")
	asst.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}

	getNodeLabelsQuery := []*cmneo4j.Query{
		{
			Cypher: `
				MATCH (t:Content {uuid: $uuid}) RETURN labels(t)
				`,
			Params: map[string]interface{}{
				"uuid": standardContent.UUID,
			},
			Result: &result,
		},
	}

	err = d.Write(context.Background(), getNodeLabelsQuery...)
	asst.NoError(err)

	want := []string{"Thing", "Content"}
	eq := cmp.Equal(result[0].NodeLabels, want, sortStringSlicesDesc)
	diff := cmp.Diff(result[0].NodeLabels, want, sortStringSlicesDesc)
	asst.True(eq, fmt.Sprintf("- got, + want: %s", diff))
}

func TestWriteNodeLabelsAreWrittenForContentPackage(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	err := s.Write(standardContentPackage, "TEST_TRANS_ID")
	asst.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}

	getNodeLabelsQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid: $uuid}) RETURN labels(t)
				`,
		Params: map[string]interface{}{
			"uuid": standardContent.UUID,
		},
		Result: &result,
	}

	err = d.Write(context.Background(), getNodeLabelsQuery)
	asst.NoError(err)

	want := []string{"Thing", "Content", "ContentPackage"}
	eq := cmp.Equal(result[0].NodeLabels, want, sortStringSlicesDesc)
	diff := cmp.Diff(result[0].NodeLabels, want, sortStringSlicesDesc)

	asst.True(eq, fmt.Sprintf("- got, + want: %s", diff))
}

func TestWriteNodeLabelsAreWrittenForGenericContentPackage(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	err := s.Write(genericContentPackage, "TEST_TRANS_ID")
	asst.NoError(err)

	var result []struct {
		NodeLabels []string `json:"labels(t)"`
	}
	getNodeLabelsQuery := &cmneo4j.Query{
		Cypher: `
				MATCH (t:Content {uuid: $uuid}) RETURN labels(t)
				`,
		Params: map[string]interface{}{
			"uuid": genericContentPackage.UUID,
		},
		Result: &result,
	}

	err = d.Write(context.Background(), getNodeLabelsQuery)
	asst.NoError(err)

	want := []string{"Thing", "Content", "ContentPackage"}
	eq := cmp.Equal(result[0].NodeLabels, want, sortStringSlicesDesc)
	diff := cmp.Diff(result[0].NodeLabels, want, sortStringSlicesDesc)

	asst.True(eq, fmt.Sprintf("- got, + want: %s", diff))
}

func TestContentWontBeWrittenIfNoBody(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	err := s.Write(contentWithoutABody, "TEST_TRANS_ID")
	asst.NoError(err, "Failed to write content")

	storedContent, _, err := s.Read(contentWithoutABody.UUID, "TEST_TRANS_ID")
	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		asst.Empty(storedContent)
	} else {
		asst.NoError(err)
	}
	asst.Equal(
		content{},
		storedContent,
		"No content should be written when the content has no body",
	)
}

func TestContentWontBeWrittenIfNoBodyWithInvalidType(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")
	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(contentWithoutABodyWithType, "TEST_TRANS_ID"), "Failed to write content")
	storedContent, _, err := s.Read(contentWithoutABodyWithType.UUID, "TEST_TRANS_ID")

	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		asst.Empty(storedContent)
	} else {
		asst.NoError(err)
	}
	asst.Equal(
		content{},
		storedContent,
		"No content should be written when the content has no body",
	)
}

func TestLiveEventWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, liveEventContent)
}

func TestLiveBlogsWillBeWrittenDespiteNoBody(t *testing.T) {
	testContentWillBeWritten(t, liveBlog)
}

func TestContentPlaceholderWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, contentPlaceholder)
}

func TestVideoContentWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, videoContent)
}

func TestGraphicWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, graphicContent)
}

func TestAudioWillBeWritten(t *testing.T) {
	testContentWillBeWritten(t, audioContent)
}

func TestSpecialContentWillNotBeWritten(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")

	a := getAgent(specialContentPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(specialContent, "TEST_TRANS_ID"), "Failed to write content")

	c, _, err := s.Read(specialContent.UUID, "TEST_TRANS_ID")
	asst.NoError(err)
	asst.Empty(
		c,
		"There should not have been any content retrieved. Policy filtering is not working.",
	)
}

func TestContentWillBeWrittenSpecialContentCheck(t *testing.T) {
	asst := assert.New(t)
	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")

	a := getAgent(specialContentPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(standardContent, "TEST_TRANS_ID"), "Failed to write content")

	c, _, err := s.Read(standardContent.UUID, "TEST_TRANS_ID")
	asst.NoError(err)
	asst.NotEmpty(
		c,
		"Failed to retrieve stored content. There is something wrong with the special content policy.",
	)

	r := c.(content)
	asst.Equal(standardContent.UUID, r.UUID, "Failed to match UUID")
	asst.Equal(standardContent.Title, r.Title, "Failed to match Title")
}

func testContentWillBeWritten(t *testing.T, c content) {
	asst := assert.New(t)

	l := logger.NewUPPLogger("content-rw-neo4j-test", "PANIC")

	a := getAgent(defaultPolicy, l, t)
	d := getDriverAndCheckClean(t, asst, l)
	s := getContentService(d, a, l)
	defer cleanDB(d, asst)

	asst.NoError(s.Write(c, "TEST_TRANS_ID"), "Failed to write content")

	storedContent, _, err := s.Read(c.UUID, "TEST_TRANS_ID")
	asst.NoError(err)
	asst.NotEmpty(storedContent, "Failed to retrieve stored content")
	actualContent := storedContent.(content)

	asst.Equal(c.UUID, actualContent.UUID, "Failed to match UUID")
	asst.Equal(c.Title, actualContent.Title, "Failed to match Title")
}

func getDriverAndCheckClean(
	t *testing.T,
	a *assert.Assertions,
	l *logger.UPPLogger,
) *cmneo4j.Driver {
	d := getNeoDriver(a, l)
	cleanDB(d, a)
	checkDBClean(d, t)
	return d
}

func getNeoDriver(a *assert.Assertions, l *logger.UPPLogger) *cmneo4j.Driver {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "bolt://localhost:7687"
	}
	d, err := cmneo4j.NewDefaultDriver(context.Background(), url, l)
	a.NoError(err, "Failed to connect to Neo4j")
	return d
}

func cleanDB(d *cmneo4j.Driver, a *assert.Assertions) {
	uuids := []string{
		contentUUID,
		conceptUUID,
		liveBlogUUID,
		noBodyContentUUID,
		noBodyInvalidTypeContentUUID,
		contentPlaceholderUUID,
		videoContentUUID,
		storyPackageUUID,
		contentPackageUUID,
		contentCollectionUUID,
		thingUUID,
		genericContentPackageUUID,
		graphicUUID,
		audioContentUUID,
		liveEventUUID,
	}

	var qs []*cmneo4j.Query
	for _, uuid := range uuids {
		qs = append(qs, &cmneo4j.Query{
			Cypher: `MATCH (t:Thing {uuid: $uuid}) DETACH DELETE t`,
			Params: map[string]interface{}{
				"uuid": uuid,
			},
		})
	}

	err := d.Write(context.Background(), qs...)
	a.NoError(err)
}

func writeContentPackageContainsRelation(
	d *cmneo4j.Driver,
	cpUUID string,
	UUID string,
	a *assert.Assertions,
) {
	writeRelation := `
	MATCH (cp:ContentPackage {uuid: $cpUUID}), (t {uuid:$UUID})
	CREATE (cp)-[pred:CONTAINS]->(t)
	`

	qs := []*cmneo4j.Query{
		{
			Cypher: writeRelation,
			Params: map[string]interface{}{
				"cpUUID": cpUUID,
				"UUID":   UUID,
			},
		},
	}

	err := d.Write(context.Background(), qs...)
	a.NoError(err)
}

func writeNodeWithLabels(d *cmneo4j.Driver, UUID string, labels string, a *assert.Assertions) {
	writeThingWithLabelsQuery := `CREATE (n:` + labels + `{uuid: $uuid})`

	qs := []*cmneo4j.Query{
		{
			Cypher: writeThingWithLabelsQuery,
			Params: map[string]interface{}{
				"uuid": UUID,
			},
		},
	}

	err := d.Write(context.Background(), qs...)
	a.NoError(err)
}

func writeRelationship(
	d *cmneo4j.Driver,
	contentID string,
	conceptID string,
	a *assert.Assertions,
) {
	annotateQuery := `
		MERGE (content:Thing{uuid:$contentId})
		MERGE (concept:Thing) ON CREATE SET concept.uuid = $conceptId
		MERGE (content)-[pred:SOME_PREDICATE]->(concept)
		`

	qs := []*cmneo4j.Query{
		{
			Cypher: annotateQuery,
			Params: map[string]interface{}{
				"contentId": contentID,
				"conceptId": conceptID,
			},
		},
	}

	err := d.Write(context.Background(), qs...)
	a.NoError(err)
}

func doesThingExist(uuid string, d *cmneo4j.Driver) (bool, error) {
	var result []struct {
		UUID string `json:"uuid,omitempty"`
	}
	query := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Thing {uuid: $uuid})
			RETURN t.uuid as uuid`,
		Params: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &result,
	}
	err := d.Write(context.Background(), query)
	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		return false, nil
	}
	return len(result) > 0, err
}

func checkIsCuratedForRelationship(d *cmneo4j.Driver, spID string, a *assert.Assertions) int {
	countQuery := `
		MATCH (t:Thing{uuid:$storyPackageId})-[r:IS_CURATED_FOR]->(x)
		RETURN count(r) as c`

	var results []struct {
		Count int `json:"c"`
	}

	qs := &cmneo4j.Query{
		Cypher: countQuery,
		Params: map[string]interface{}{
			"storyPackageId": spID,
		},
		Result: &results,
	}

	err := d.Write(context.Background(), qs)
	a.NoError(err)

	return results[0].Count
}

func checkContainsRelationship(d *cmneo4j.Driver, cpID string, a *assert.Assertions) int {
	countQuery := `
		MATCH (t:Thing{uuid:$contentPackageId})<-[r:CONTAINS]-(x)
		RETURN count(r) as c`

	var results []struct {
		Count int `json:"c"`
	}

	qs := &cmneo4j.Query{
		Cypher: countQuery,
		Params: map[string]interface{}{"contentPackageId": cpID},
		Result: &results,
	}

	err := d.Write(context.Background(), qs)
	a.NoError(err)

	return results[0].Count
}

func checkDBClean(d *cmneo4j.Driver, t *testing.T) {
	a := assert.New(t)

	var result []struct {
		UUID string `json:"t.uuid"`
	}

	checkGraph := &cmneo4j.Query{
		Cypher: `
			MATCH (t:Thing) WHERE t.uuid in $uuids
			RETURN t.uuid
		`,
		Params: map[string]interface{}{
			"uuids": []string{standardContent.UUID, conceptUUID},
		},
		Result: &result,
	}
	err := d.Write(context.Background(), checkGraph)
	if errors.Is(err, cmneo4j.ErrNoResultsFound) {
		a.Empty(result)
	} else {
		a.NoError(err)
	}
}

func getAgent(p string, l *logger.UPPLogger, t *testing.T) policy.Agent {
	url := os.Getenv("OPA_URL")
	if url == "" {
		url = "http://localhost:8181"
	}
	paths := map[string]string{
		policy.SpecialContentKey: "content_rw_neo4j/special_content",
	}

	c := http.DefaultClient

	opaClient := opa.NewOpenPolicyAgentClient(url, paths, opa.WithLogger(l), opa.WithHttpClient(c))
	a := policy.NewOpenPolicyAgent(opaClient, l)

	// TODO: This is a nice reminder that our opa-client-go library could be expanded to do more. Such logic could be refactored there.
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/v1/policies/c1d20fbc-e9bc-44fb-8b88-0e01d6b13225", url),
		bytes.NewReader([]byte(p)),
	)
	if err != nil {
		t.Logf(
			"could not create a request for creating a test policy in the testing policy agent: %s",
			err,
		)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "text/plain")

	res, err := c.Do(req)
	if err != nil {
		t.Logf("could not create a test policy in the testing policy agent: %s", err)
		t.FailNow()
	}
	defer res.Body.Close()

	return a
}

func getContentService(d *cmneo4j.Driver, a policy.Agent, l *logger.UPPLogger) Service {
	cs := NewContentService(d, a, l)
	_ = cs.Initialise()
	return cs
}
