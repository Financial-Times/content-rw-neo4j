package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Financial-Times/go-logger/v2"
)

const SpecialContentKey string = "SPECIAL_CONTENT"
const pathPrefix string = "v1/data"

type Agent struct {
	url        string
	paths      map[string]string
	httpClient *http.Client
	logger     *logger.UPPLogger
}

func NewAgent(u string, p map[string]string, c *http.Client, l *logger.UPPLogger) *Agent {
	return &Agent{
		url:        u,
		paths:      p,
		httpClient: c,
		logger:     l,
	}
}

func (a *Agent) CheckSpecialContentPolicy(q SpecialContentQuery) (*Decision, error) {
	specialContentPath, ok := a.paths[SpecialContentKey]
	if !ok {
		return nil, &QueryMissingPathConfigError{Key: SpecialContentKey}
	}

	res, err := a.queryPolicyAgent(q, specialContentPath)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(res, &m)
	if err != nil {
		return nil, &DecisionUnmarshallError{Err: err}
	}

	d := Decision{}

	decisionID, ok := m["decision_id"].(string)
	if !ok {
		return nil, &DecisionPayloadError{Msg: "could not cast decision_id to string"}
	}
	d.DecisionID = decisionID

	result, ok := m["result"].(map[string]interface{})
	if !ok {
		return nil, &DecisionPayloadError{Msg: "result is either nil or there was a problem casting it"}
	}

	isSpecialContent, ok := result["is_special_content"].(bool)
	if !ok {
		return nil, &DecisionPayloadError{Msg: "could not cast is_special_content to bool"}
	}

	sd := SpecialContentDecision{}
	sd.IsSpecialContent = isSpecialContent

	d.Result = sd

	return &d, nil
}

func (a *Agent) queryPolicyAgent(query interface{}, path string) ([]byte, error) {
	q := Query{
		Input: query,
	}

	m, err := json.Marshal(q)
	if err != nil {
		return nil, &QueryMarshallError{Err: err}
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s/%s", a.url, pathPrefix, path),
		bytes.NewReader(m),
	)
	if err != nil {
		return nil, &QueryRequestCreationError{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.httpClient.Do(req)
	if err != nil {
		return nil, &QueryRequestError{Err: err}
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &QueryResponseReadingError{Err: err}
	}

	return b, nil
}
