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

package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/utils"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// BasecampConn holds the essential information to connect to the Basecamp API
type BasecampConn struct {
	helper.RestConnection `mapstructure:",squash"`
	helper.AccessToken    `mapstructure:",squash"`
	AccountId             string `mapstructure:"accountId" validate:"required" json:"accountId" gorm:"type:varchar(255)"`

	// OAuth2 fields for automatic token refresh
	ClientId       string    `mapstructure:"clientId" json:"clientId" gorm:"type:varchar(255)"`
	ClientSecret   string    `mapstructure:"clientSecret" json:"clientSecret" gorm:"type:text;serializer:encdec"`
	RefreshToken   string    `mapstructure:"refreshToken" json:"refreshToken" gorm:"type:text;serializer:encdec"`
	TokenExpiresAt time.Time `mapstructure:"tokenExpiresAt" json:"tokenExpiresAt"`
}

// UpdateToken updates the access token and expiry time
func (bc *BasecampConn) UpdateToken(newToken string, expiresIn int) {
	bc.Token = newToken
	bc.TokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func (bc *BasecampConn) Sanitize() BasecampConn {
	bc.Token = utils.SanitizeString(bc.Token)
	bc.ClientSecret = utils.SanitizeString(bc.ClientSecret)
	bc.RefreshToken = utils.SanitizeString(bc.RefreshToken)
	return *bc
}

// SetupAuthentication sets up the HTTP Request Authentication
func (bc *BasecampConn) SetupAuthentication(req *http.Request) errors.Error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bc.Token))
	req.Header.Set("User-Agent", "DevLake (https://devlake.apache.org)")
	return nil
}

// BasecampConnection holds BasecampConn plus ID/Name for database storage
type BasecampConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	BasecampConn          `mapstructure:",squash"`
}

func (connection *BasecampConnection) MergeFromRequest(target *BasecampConnection, body map[string]interface{}) error {
	// Preserve existing sensitive fields
	token := target.Token
	clientSecret := target.ClientSecret
	refreshToken := target.RefreshToken

	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}

	// Restore token if unchanged or sanitized
	modifiedToken := target.Token
	if modifiedToken == "" || modifiedToken == utils.SanitizeString(token) {
		target.Token = token
	}

	// Restore clientSecret if unchanged or sanitized
	modifiedClientSecret := target.ClientSecret
	if modifiedClientSecret == "" || modifiedClientSecret == utils.SanitizeString(clientSecret) {
		target.ClientSecret = clientSecret
	}

	// Restore refreshToken if unchanged or sanitized
	modifiedRefreshToken := target.RefreshToken
	if modifiedRefreshToken == "" || modifiedRefreshToken == utils.SanitizeString(refreshToken) {
		target.RefreshToken = refreshToken
	}

	return nil
}

func (connection BasecampConnection) Sanitize() BasecampConnection {
	connection.BasecampConn = connection.BasecampConn.Sanitize()
	return connection
}

func (BasecampConnection) TableName() string {
	return "_tool_basecamp_connections"
}
