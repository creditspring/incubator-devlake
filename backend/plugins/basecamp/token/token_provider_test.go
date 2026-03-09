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

package token

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/impls/dalgorm"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testEncryptionSecret = "test-secret-key!"

// noopLogger satisfies log.Logger for tests without producing output.
type noopLogger struct{}

func (noopLogger) IsLevelEnabled(log.LogLevel) bool                    { return false }
func (noopLogger) Printf(string, ...interface{})                       {}
func (noopLogger) Log(log.LogLevel, string, ...interface{})            {}
func (noopLogger) Debug(string, ...interface{})                        {}
func (noopLogger) Info(string, ...interface{})                         {}
func (noopLogger) Warn(error, string, ...interface{})                  {}
func (noopLogger) Error(error, string, ...interface{})                 {}
func (noopLogger) Nested(string) log.Logger                            { return noopLogger{} }
func (noopLogger) GetConfig() *log.LoggerConfig                       { return &log.LoggerConfig{} }
func (noopLogger) SetStream(*log.LoggerStreamConfig)                   {}

// TestForceRefresh_EncDecRoundTrip verifies that token refresh persists the
// token through GORM's encdec serializer and reads it back correctly.
// This is an integration test using a real SQLite database — no mocks.
func TestForceRefresh_EncDecRoundTrip(t *testing.T) {
	// 1. Register the encdec serializer with a known secret
	dalgorm.Init(testEncryptionSecret)

	// 2. Open an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 3. Create the table
	require.NoError(t, db.AutoMigrate(&models.BasecampConnection{}))

	// 4. Wrap as dal.Dal
	dalDB := dalgorm.NewDalgorm(db)

	// 5. Insert a connection with a known token
	oldToken := "old-access-token"
	conn := &models.BasecampConnection{}
	conn.ID = 1
	conn.Name = "test-connection"
	conn.Endpoint = "https://3.basecampapi.com/"
	conn.Token = oldToken
	conn.AccountId = "12345"
	conn.ClientId = "test-client-id"
	conn.ClientSecret = "test-client-secret"
	conn.RefreshToken = "test-refresh-token"

	require.NoError(t, dalDB.Create(conn))

	// 6. Set up a fake OAuth2 server returning a new token
	newToken := "refreshed-access-token-xyz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"access_token": newToken,
			"expires_in":   1209600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Patch the refresh URL by creating a custom HTTP client that redirects to our test server
	transport := &rewriteTransport{server.URL}
	httpClient := &http.Client{Transport: transport}

	// 7. Create the TokenProvider and call ForceRefresh
	tp := NewTokenProvider(conn, dalDB, httpClient, noopLogger{})
	refreshErr := tp.ForceRefresh(oldToken)
	require.NoError(t, refreshErr)

	// 8. Verify the in-memory conn was updated
	assert.Equal(t, newToken, conn.Token, "in-memory token should be updated")

	// 9. Read the connection back from DB through dal (which decrypts via encdec)
	readConn := &models.BasecampConnection{}
	require.NoError(t, dalDB.First(readConn))

	assert.Equal(t, newToken, readConn.Token,
		"token read back from DB should match the new plaintext token (encrypt→decrypt round-trip)")
	assert.Equal(t, "test-client-secret", readConn.ClientSecret,
		"client secret should survive the round-trip")
	assert.Equal(t, "test-refresh-token", readConn.RefreshToken,
		"refresh token should survive the round-trip")

	// 10. Verify the raw DB value is NOT plaintext (i.e. it was encrypted)
	var rawToken string
	row := db.Raw("SELECT token FROM _tool_basecamp_connections WHERE id = 1").Row()
	require.NoError(t, row.Scan(&rawToken))

	assert.NotEqual(t, newToken, rawToken,
		"raw DB value should be encrypted, not plaintext")
	assert.NotEmpty(t, rawToken,
		"raw DB value should not be empty")
}

// rewriteTransport redirects all requests to the test server URL,
// preserving the request body and headers.
type rewriteTransport struct {
	targetURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.targetURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}
