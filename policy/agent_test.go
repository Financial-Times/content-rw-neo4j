package policy

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/opa-client-go"
)

const (
	testDecisionID = "1e58b3bf-995c-473e-90e9-ab1f10af74ab"
)

func TestAgent_EvaluateSpecialContentPolicy(t *testing.T) {
	tests := []struct {
		name           string
		server         *httptest.Server
		paths          map[string]string
		query          map[string]interface{}
		expectedResult *SpecialContentPolicyResult
		expectedError  error
	}{
		{
			name: "Evaluate a valid decision for special content",
			server: createHTTPTestServer(
				t,
				fmt.Sprintf(`{"decision_id": %q, "result": {"is_special_content": true}}`, testDecisionID),
			),
			paths: map[string]string{
				SpecialContentKey: "special/content",
			},
			query: map[string]interface{}{
				"editorialDesk": "/FT/Professional/Central Banking",
			},
			expectedResult: &SpecialContentPolicyResult{
				IsSpecialContent: true,
			},
			expectedError: nil,
		},
		{
			name: "Evaluate a valid decision for non-special content",
			server: createHTTPTestServer(
				t,
				fmt.Sprintf(`{"decision_id": %q, "result": {"is_special_content": false}}`, testDecisionID),
			),
			paths: map[string]string{
				SpecialContentKey: "special/content",
			},
			query: map[string]interface{}{
				"editorialDesk": "/FT/Professional/Not Central Banking",
			},
			expectedResult: &SpecialContentPolicyResult{
				IsSpecialContent: false,
			},
			expectedError: nil,
		},
		{
			name: "Evaluate and receive an error.",
			server: createHTTPTestServer(
				t,
				``,
			),
			paths:          make(map[string]string),
			query:          make(map[string]interface{}),
			expectedResult: nil,
			expectedError:  ErrEvaluatePolicy,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(T *testing.T) {
			defer test.server.Close()

			l := logger.NewUPPLogger("content-rw-neo4j", "INFO")
			c := opa.NewOpenPolicyAgentClient(test.server.URL, test.paths, opa.WithLogger(l))

			o := NewOpenPolicyAgent(c, l)

			result, err := o.EvaluateSpecialContentPolicy(test.query)

			if err != nil {
				if !errors.Is(err, test.expectedError) {
					t.Errorf(
						"Unexpected error received from call to EvaluateSpecialContentPolicy: %v",
						err,
					)
				}
			} else {
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func createHTTPTestServer(t *testing.T, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		if err != nil {
			t.Fatalf("could not write response from test http server: %v", err)
		}
	}))
}
