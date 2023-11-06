package policy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/go-logger/v2"
)

const specialContentEditorialDesk string = "/FT/Professional/Central Banking"

func TestAgent_CheckSpecialContentPolicyNoPathsConfig(t *testing.T) {
	q := SpecialContentQuery{
		EditorialDesk: specialContentEditorialDesk,
	}

	a := NewAgent("", make(map[string]string), nil, nil)

	_, err := a.CheckSpecialContentPolicy(q)
	assert.NotNil(t, err)
	assert.IsType(t, &QueryMissingPathConfigError{}, err)
}

func TestAgent_CheckSpecialContentPolicyEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(
			[]byte(`{"decision_id": "TEST_UUID"}`),
		)
		if err != nil {
			t.Logf(
				"could not setup test server, failed to send mock response: %s",
				err,
			)
			t.FailNow()
		}
	}))
	defer srv.Close()

	q := SpecialContentQuery{
		EditorialDesk: specialContentEditorialDesk,
	}
	p := map[string]string{
		SpecialContentKey: "content_rw_neo4j/special_content",
	}
	c := http.Client{}
	l := logger.UPPLogger{}

	a := NewAgent(srv.URL, p, &c, &l)

	_, err := a.CheckSpecialContentPolicy(q)

	assert.NotNil(t, err)
	assert.IsType(t, &DecisionPayloadError{}, err)
}

func TestAgent_CheckSpecialContentPolicy(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Logf(
				"could not setup test server, failed to read mock request: %s",
				err,
			)
			t.FailNow()
		}

		m := make(map[string]interface{})
		err = json.Unmarshal(b, &m)
		if err != nil {
			t.Logf(
				"could not setup test server, failed to unmarshal mock request: %s",
				err,
			)
			t.FailNow()
		}

		if m["input"].(map[string]interface{})["editorialDesk"] == specialContentEditorialDesk {
			_, err := w.Write(
				[]byte(`{"decision_id": "TEST_UUID", "result": {"is_special_content": true}}`),
			)
			if err != nil {
				t.Logf(
					"could not setup test server, failed to send mock response: %s",
					err,
				)
				t.FailNow()
			}
		} else {
			_, err := w.Write([]byte(`{"decision_id": "TEST_UUID", "result": {"is_special_content": false}}`))
			if err != nil {
				t.Logf(
					"could not setup test server, failed to send mock response: %s",
					err,
				)
				t.FailNow()
			}
		}
	}))
	defer srv.Close()

	tests := []struct {
		name     string
		query    SpecialContentQuery
		expected bool
	}{
		{
			name: "Special content editorial desk.",
			query: SpecialContentQuery{
				EditorialDesk: specialContentEditorialDesk,
			},
			expected: true,
		},
		{
			name: "Standard content editorial desk.",
			query: SpecialContentQuery{
				EditorialDesk: "/FT/Professional/Standard Content",
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := map[string]string{
				SpecialContentKey: "content_rw_neo4j/special_content",
			}
			c := http.Client{}
			l := logger.UPPLogger{}

			a := NewAgent(srv.URL, p, &c, &l)

			d, err := a.CheckSpecialContentPolicy(test.query)
			if err != nil {
				t.Logf(
					"an error occurred while testing CheckSpecialContentPolicy: %s",
					err,
				)
				t.FailNow()
			}

			assert.Equal(t, test.expected, d.Result.(SpecialContentDecision).IsSpecialContent)
		})
	}
}
