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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_VAULT_TABLE = "basecamp_vaults"

var _ plugin.SubTaskEntryPoint = CollectVaults

var CollectVaultsMeta = plugin.SubTaskMeta{
	Name:             "CollectVaults",
	EntryPoint:       CollectVaults,
	EnabledByDefault: true,
	Description:      "Recursively collect sub-vaults (folders) from all root vaults in synced Basecamp projects",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectVaults(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()

	ageLimitMonths := 0
	if data.ScopeConfig != nil && data.ScopeConfig.ProjectAgeLimitMonths > 0 {
		ageLimitMonths = data.ScopeConfig.ProjectAgeLimitMonths
	}

	var permanentIds []string
	if data.ScopeConfig != nil && data.ScopeConfig.PermanentProjectIds != "" {
		for _, id := range strings.Split(data.ScopeConfig.PermanentProjectIds, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				permanentIds = append(permanentIds, trimmed)
			}
		}
	}

	var projects []models.BasecampProject
	var err errors.Error

	if ageLimitMonths == 0 {
		err = db.All(&projects,
			dal.From(&models.BasecampProject{}),
			dal.Where("connection_id = ? AND vault_ids != ''", data.Options.ConnectionId),
		)
	} else {
		cutoffDate := data.now().AddDate(0, -ageLimitMonths, 0)
		if len(permanentIds) > 0 {
			err = db.All(&projects,
				dal.From(&models.BasecampProject{}),
				dal.Where("connection_id = ? AND vault_ids != '' AND (created_at > ? OR project_id IN ?)",
					data.Options.ConnectionId, cutoffDate, permanentIds),
			)
		} else {
			err = db.All(&projects,
				dal.From(&models.BasecampProject{}),
				dal.Where("connection_id = ? AND vault_ids != '' AND created_at > ?",
					data.Options.ConnectionId, cutoffDate),
			)
		}
	}
	if err != nil {
		return err
	}

	rootVaults := buildVaultInputs(projects)
	if len(rootVaults) == 0 {
		taskCtx.GetLogger().Info("No root vaults found, skipping sub-vault collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting sub-vaults from %d root vaults across %d projects", len(rootVaults), len(projects))

	// iterator is captured by the ResponseParser closure to enable recursive discovery:
	// when sub-vaults are found, their IDs are pushed back so they are also queried.
	iterator := api.NewQueueIterator()
	for _, v := range rootVaults {
		iterator.Push(v)
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_VAULT_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15,
		UrlTemplate: fmt.Sprintf("%s/vaults/{{ .Input.VaultId }}/vaults.json",
			data.AccountId),
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			if reqData.CustomData != nil {
				if page, ok := reqData.CustomData.(int); ok && page > 1 {
					query.Set("page", fmt.Sprintf("%d", page))
				}
			}
			return query, nil
		},
		GetNextPageCustomData: func(prevReqData *api.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			nextURL := GetNextPageURL(prevPageResponse)
			if nextURL == "" {
				return nil, api.ErrFinishCollect
			}
			return prevReqData.Pager.Page + 1, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var rawVaults []json.RawMessage
			if err := api.UnmarshalResponse(res, &rawVaults); err != nil {
				return nil, err
			}
			// Push each discovered sub-vault ID back into the iterator so its
			// own sub-vaults are also collected (BFS recursive discovery).
			for _, raw := range rawVaults {
				var v struct {
					ID int64 `json:"id"`
				}
				if jsonErr := json.Unmarshal(raw, &v); jsonErr == nil && v.ID > 0 {
					iterator.Push(&VaultInput{VaultId: strconv.FormatInt(v.ID, 10)})
				}
			}
			return rawVaults, nil
		},
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
