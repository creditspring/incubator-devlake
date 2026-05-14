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
	"net/http"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/apache/incubator-devlake/plugins/basecamp/token"
)

// basecampResponseHook is registered on every Basecamp async client.
// 404 — resource deleted since last sync; skip silently.
// 401 — retrying won't help (bad token); skip and log so the rest of the task continues.
func basecampResponseHook(logger log.Logger) plugin.ApiClientAfterResponse {
	return func(res *http.Response) errors.Error {
		switch res.StatusCode {
		case http.StatusNotFound:
			logger.Warn(nil, "Basecamp resource not found (404), skipping: %s", res.Request.URL)
			return api.ErrIgnoreAndContinue
		case http.StatusUnauthorized:
			logger.Warn(nil, "Basecamp unauthorized (401), skipping: %s", res.Request.URL)
			return api.ErrIgnoreAndContinue
		}
		return nil
	}
}

// CreateApiClient creates a new API Client for Basecamp
func CreateApiClient(taskCtx plugin.TaskContext, connection *models.BasecampConnection) (*api.ApiAsyncClient, errors.Error) {
	apiClient, err := api.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, connection)
	if err != nil {
		return nil, err
	}

	// Add token refresh middleware if OAuth2 credentials are present
	if connection.RefreshToken != "" {
		tp := token.NewTokenProvider(
			connection,
			taskCtx.GetDal(),
			&http.Client{},
			taskCtx.GetLogger(),
		)
		rt := token.NewRefreshRoundTripper(
			apiClient.GetClient().Transport,
			tp,
		)
		apiClient.GetClient().Transport = rt
	}

	// create async api client
	asyncApiClient, err := api.CreateAsyncApiClient(taskCtx, apiClient, nil)
	if err != nil {
		return nil, err
	}

	asyncApiClient.SetAfterFunction(basecampResponseHook(taskCtx.GetLogger()))

	return asyncApiClient, nil
}
