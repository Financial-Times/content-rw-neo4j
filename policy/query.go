package policy

type Query struct {
	Input interface{} `json:"input"`
}

type SpecialContentQuery struct {
	EditorialDesk string `json:"editorialDesk"`
}
