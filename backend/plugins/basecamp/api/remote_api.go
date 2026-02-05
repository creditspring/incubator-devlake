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

package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

type BasecampRemotePagination struct{}

func listBasecampRemoteScopes(
	connection *models.BasecampConnection,
	apiClient plugin.ApiClient,
	groupId string,
	page BasecampRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.BasecampAccount],
	nextPage *BasecampRemotePagination,
	err errors.Error,
) {
	// Return the configured account as the only available scope
	children = []dsmodels.DsRemoteApiScopeListEntry[models.BasecampAccount]{
		{
			Type:     api.RAS_ENTRY_TYPE_SCOPE,
			Id:       connection.AccountId,
			Name:     "Basecamp Account " + connection.AccountId,
			FullName: "Basecamp Account " + connection.AccountId,
			Data: &models.BasecampAccount{
				AccountId: connection.AccountId,
				Name:      "Basecamp Account " + connection.AccountId,
			},
		},
	}
	return children, nil, nil
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}
