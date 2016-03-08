package content

type content struct {
	UUID          string  `json:"uuid,omitempty"`
	Title         string  `json:"title,omitempty"`
	PublishedDate string  `json:"publishedDate,omitempty"`
	Body          string  `json:"body,omitempty"`
	Brands        []brand `json:"brands,omitempty"`
}

type brand struct {
	Id string `json:"id"`
}
