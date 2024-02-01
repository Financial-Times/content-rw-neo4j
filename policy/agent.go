package policy

import (
	"errors"
	"fmt"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/opa-client-go"
)

var ErrEvaluatePolicy = errors.New("error evaluating policy")

const (
	SpecialContentKey = "special_content"
)

type SpecialContentPolicyResult struct {
	IsSpecialContent bool `json:"is_special_content"`
}

type Agent interface {
	EvaluateSpecialContentPolicy(q map[string]interface{}) (*SpecialContentPolicyResult, error)
}

type OpenPolicyAgent struct {
	client *opa.OpenPolicyAgentClient
	log    *logger.UPPLogger
}

func NewOpenPolicyAgent(c *opa.OpenPolicyAgentClient, l *logger.UPPLogger) *OpenPolicyAgent {
	return &OpenPolicyAgent{
		client: c,
		log:    l,
	}
}

func (o *OpenPolicyAgent) EvaluateSpecialContentPolicy(
	q map[string]interface{},
) (*SpecialContentPolicyResult, error) {
	r := &SpecialContentPolicyResult{}

	decisionID, err := o.client.DoQuery(q, SpecialContentKey, r)
	if err != nil {
		return nil, fmt.Errorf("%w: Special Content Policy: %w", ErrEvaluatePolicy, err)
	}

	o.log.Infof("Evaluated Special Content Policy: decisionID: %q, result: %v", decisionID, r)

	return r, nil
}
