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
	"encoding/json"
	"strconv"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ExtractVaults

var ExtractVaultsMeta = plugin.SubTaskMeta{
	Name:             "ExtractVaults",
	EntryPoint:       ExtractVaults,
	EnabledByDefault: true,
	Description:      "Extract raw vault data into tool layer table _tool_basecamp_vaults",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type BasecampApiVault struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Parent struct {
		ID int64 `json:"id"`
	} `json:"parent"`
}

func ExtractVaults(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_VAULT_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiVault := &BasecampApiVault{}
			if err := errors.Convert(json.Unmarshal(resData.Data, apiVault)); err != nil {
				return nil, err
			}

			vault := &models.BasecampVault{
				ConnectionId:  taskData.Options.ConnectionId,
				VaultId:       strconv.FormatInt(apiVault.ID, 10),
				ParentVaultId: strconv.FormatInt(apiVault.Parent.ID, 10),
				Title:         apiVault.Title,
			}

			return []interface{}{vault}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
