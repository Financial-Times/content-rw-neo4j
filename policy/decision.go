package policy

type Decision struct {
	DecisionID string      `json:"decision_id,omitempty"`
	Result     interface{} `json:"result"`
}

type SpecialContentDecision struct {
	IsSpecialContent bool `json:"is_special_content"`
}
