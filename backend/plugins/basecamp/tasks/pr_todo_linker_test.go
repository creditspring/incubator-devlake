/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
)

// nopLogger implements log.Logger with no-op methods for testing.
type nopLogger struct{}

func (n *nopLogger) IsLevelEnabled(log.LogLevel) bool                { return false }
func (n *nopLogger) Printf(string, ...interface{})                   {}
func (n *nopLogger) Log(log.LogLevel, string, ...interface{})        {}
func (n *nopLogger) Debug(string, ...interface{})                    {}
func (n *nopLogger) Info(string, ...interface{})                     {}
func (n *nopLogger) Warn(error, string, ...interface{})              {}
func (n *nopLogger) Error(error, string, ...interface{})             {}
func (n *nopLogger) Nested(string) log.Logger                        { return n }
func (n *nopLogger) GetConfig() *log.LoggerConfig                    { return &log.LoggerConfig{} }
func (n *nopLogger) SetStream(*log.LoggerStreamConfig)               {}

// newTestApiClient creates an api.ApiClient pointing at the given test server.
func newTestApiClient(server *httptest.Server) *api.ApiClient {
	client := &api.ApiClient{}
	client.Setup(server.URL, nil, 0)
	return client
}

func TestResolveTodoId_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/12345/buckets/100/todos/999.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": 42, "title": "resolved todo"}`)
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	cache := make(map[string]string)

	result := resolveTodoId(client, logger, cache, "12345", "100", "999")

	assert.Equal(t, "42", result)
	assert.Equal(t, "42", cache["999"])
}

func TestResolveTodoId_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	cache := make(map[string]string)

	result := resolveTodoId(client, logger, cache, "12345", "100", "999")

	assert.Equal(t, "", result)
	// Failure is cached so subsequent calls don't hit the API
	cached, ok := cache["999"]
	assert.True(t, ok)
	assert.Equal(t, "", cached)
}

func TestResolveTodoId_CacheHit(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": 42}`)
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	cache := map[string]string{"999": "42"}

	result := resolveTodoId(client, logger, cache, "12345", "100", "999")

	assert.Equal(t, "42", result)
	assert.False(t, serverCalled, "API should not be called when cache already has the result")
}

func TestResolveTodoId_CacheHitEmpty(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	// Empty string means a previous lookup failed — should not retry
	cache := map[string]string{"999": ""}

	result := resolveTodoId(client, logger, cache, "12345", "100", "999")

	assert.Equal(t, "", result)
	assert.False(t, serverCalled, "API should not be called when a failed lookup is cached")
}

func TestResolveTodoId_InvalidJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	cache := make(map[string]string)

	result := resolveTodoId(client, logger, cache, "12345", "100", "999")

	assert.Equal(t, "", result)
	assert.Equal(t, "", cache["999"])
}

func TestResolveTodoId_Redirect(t *testing.T) {
	// Simulates the moved-todo scenario: the server redirects to a new location
	// and returns the new todo's JSON. Go's http.Client follows redirects by default.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/12345/buckets/100/todos/OLD.json" {
			http.Redirect(w, r, "/12345/buckets/200/todos/NEW.json", http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": 9536886492, "title": "moved todo"}`)
	}))
	defer server.Close()

	client := newTestApiClient(server)
	logger := &nopLogger{}
	cache := make(map[string]string)

	result := resolveTodoId(client, logger, cache, "12345", "100", "OLD")

	assert.Equal(t, "9536886492", result)
	assert.Equal(t, "9536886492", cache["OLD"])
}

func TestBasecampTodoRegex_CaptureGroups(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantMatch bool
		accountId string
		bucketId  string
		todoId    string
	}{
		{
			name:      "standard URL",
			url:       "https://3.basecamp.com/12345/buckets/26549168/todos/9517412113",
			wantMatch: true,
			accountId: "12345",
			bucketId:  "26549168",
			todoId:    "9517412113",
		},
		{
			name:      "http URL",
			url:       "http://3.basecamp.com/99999/buckets/111/todos/222",
			wantMatch: true,
			accountId: "99999",
			bucketId:  "111",
			todoId:    "222",
		},
		{
			name:      "no match - wrong domain",
			url:       "https://basecamp.com/12345/buckets/100/todos/200",
			wantMatch: false,
		},
		{
			name:      "no match - missing todos path",
			url:       "https://3.basecamp.com/12345/buckets/100/todolists/200",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := basecampTodoRegex.FindStringSubmatch(tt.url)
			if !tt.wantMatch {
				assert.Nil(t, matches)
				return
			}
			assert.Len(t, matches, 4, "expected 4 groups: full match + 3 captures")
			assert.Equal(t, tt.accountId, matches[1])
			assert.Equal(t, tt.bucketId, matches[2])
			assert.Equal(t, tt.todoId, matches[3])
		})
	}
}

func TestBasecampTodoRegex_MultipleMatchesInText(t *testing.T) {
	text := `This PR fixes:
- https://3.basecamp.com/111/buckets/222/todos/333
- https://3.basecamp.com/111/buckets/444/todos/555`

	matches := basecampTodoRegex.FindAllStringSubmatch(text, -1)
	assert.Len(t, matches, 2)
	assert.Equal(t, "333", matches[0][3])
	assert.Equal(t, "555", matches[1][3])
}

func TestGeneratePrReferenceId_Deterministic(t *testing.T) {
	id1 := generatePrReferenceId(1, "pr-123", "https://example.com")
	id2 := generatePrReferenceId(1, "pr-123", "https://example.com")
	assert.Equal(t, id1, id2, "same inputs should produce the same ID")
}

func TestGeneratePrReferenceId_DifferentInputs(t *testing.T) {
	id1 := generatePrReferenceId(1, "pr-123", "https://example.com/a")
	id2 := generatePrReferenceId(1, "pr-123", "https://example.com/b")
	assert.NotEqual(t, id1, id2, "different URLs should produce different IDs")

	id3 := generatePrReferenceId(1, "pr-123", "https://example.com")
	id4 := generatePrReferenceId(2, "pr-123", "https://example.com")
	assert.NotEqual(t, id3, id4, "different connection IDs should produce different IDs")
}
