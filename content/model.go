package content

type content struct {
	UUID           string `json:"uuid,omitempty"`
	Title          string `json:"title,omitempty"`
	PublishedDate  string `json:"publishedDate,omitempty"`
	Body           string `json:"body,omitempty"`
	Type           string `json:"type,omitempty"`
	StoryPackage   string `json:"storyPackage,omitempty"`
	ContentPackage string `json:"contentPackage,omitempty"`
}
