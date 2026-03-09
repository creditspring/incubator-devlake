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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const (
	// RefreshURL is the 37signals OAuth2 token endpoint
	RefreshURL = "https://launchpad.37signals.com/authorization/token"
	// DefaultRefreshBuffer is the time before expiry to trigger a refresh
	DefaultRefreshBuffer = 5 * time.Minute
)

// TokenProvider manages OAuth2 token refresh for Basecamp connections
type TokenProvider struct {
	conn       *models.BasecampConnection
	dal        dal.Dal
	httpClient *http.Client
	logger     log.Logger
	mu         sync.Mutex
}

// NewTokenProvider creates a TokenProvider for the given Basecamp connection
func NewTokenProvider(conn *models.BasecampConnection, d dal.Dal, client *http.Client, logger log.Logger) *TokenProvider {
	return &TokenProvider{
		conn:       conn,
		dal:        d,
		httpClient: client,
		logger:     logger,
	}
}

// GetToken returns the current access token, refreshing if necessary
func (tp *TokenProvider) GetToken() (string, errors.Error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.needsRefresh() {
		if err := tp.refreshToken(); err != nil {
			return "", err
		}
	}
	return tp.conn.Token, nil
}

// needsRefresh checks if the token needs to be refreshed
func (tp *TokenProvider) needsRefresh() bool {
	// No refresh token means we can't refresh
	if tp.conn.RefreshToken == "" {
		return false
	}

	// No expiry time means we don't know when to refresh - let API call fail first
	if tp.conn.TokenExpiresAt == nil {
		return false
	}

	buffer := DefaultRefreshBuffer
	if envBuffer := os.Getenv("BASECAMP_TOKEN_REFRESH_BUFFER_MINUTES"); envBuffer != "" {
		if val, err := strconv.Atoi(envBuffer); err == nil {
			buffer = time.Duration(val) * time.Minute
		}
	}

	return time.Now().Add(buffer).After(*tp.conn.TokenExpiresAt)
}

// refreshToken performs the OAuth2 token refresh with 37signals
func (tp *TokenProvider) refreshToken() errors.Error {
	tp.logger.Info("Refreshing Basecamp token for connection %d", tp.conn.ID)

	// Basecamp uses form-encoded POST with type=refresh
	data := url.Values{
		"type":          {"refresh"},
		"client_id":     {tp.conn.ClientId},
		"client_secret": {tp.conn.ClientSecret},
		"refresh_token": {tp.conn.RefreshToken},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", RefreshURL, strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Convert(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tp.httpClient.Do(req)
	if err != nil {
		return errors.Convert(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Convert(err)
	}

	if resp.StatusCode != http.StatusOK {
		tp.logger.Error(nil, "failed to refresh token from Basecamp, status=%d, body=%s", resp.StatusCode, string(body))
		return errors.Default.New(fmt.Sprintf("failed to refresh token: %d, body: %s", resp.StatusCode, string(body)))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.Convert(err)
	}

	if result.AccessToken == "" {
		bodyStr := string(body)
		const maxBodySnippet = 512
		if len(bodyStr) > maxBodySnippet {
			bodyStr = bodyStr[:maxBodySnippet] + "..."
		}
		return errors.Default.New(fmt.Sprintf("empty access token returned; response body: %s", bodyStr))
	}

	// Update connection with new token
	tp.conn.UpdateToken(result.AccessToken, result.ExpiresIn)

	// Persist to database using Update (Save) so GORM's encdec serializer
	// encrypts the token field correctly, unlike UpdateColumns which bypasses it.
	if tp.dal != nil {
		err := tp.dal.Update(tp.conn)
		if err != nil {
			tp.logger.Warn(err, "failed to persist refreshed token")
		}
	}

	tp.logger.Info("Successfully refreshed Basecamp token for connection %d", tp.conn.ID)
	return nil
}

// ForceRefresh forces a token refresh if the current token matches oldToken
func (tp *TokenProvider) ForceRefresh(oldToken string) errors.Error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// If the token has changed, another goroutine already refreshed it
	if tp.conn.Token != oldToken {
		return nil
	}

	return tp.refreshToken()
}
