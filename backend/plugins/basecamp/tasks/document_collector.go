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
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_DOCUMENT_TABLE = "basecamp_documents"

var _ plugin.SubTaskEntryPoint = CollectDocuments

var CollectDocumentsMeta = plugin.SubTaskMeta{
	Name:             "CollectDocuments",
	EntryPoint:       CollectDocuments,
	EnabledByDefault: true,
	Description:      "Collect documents from all vaults in synced Basecamp projects",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// VaultInput holds the vault ID needed to collect documents from one vault.
type VaultInput struct {
	VaultId string
}

// buildVaultInputs converts a slice of projects into VaultInputs by splitting each project's VaultIds.
func buildVaultInputs(projects []models.BasecampProject) []*VaultInput {
	var inputs []*VaultInput
	for _, p := range projects {
		for _, id := range strings.Split(p.VaultIds, ",") {
			if vaultId := strings.TrimSpace(id); vaultId != "" {
				inputs = append(inputs, &VaultInput{VaultId: vaultId})
			}
		}
	}
	return inputs
}

func CollectDocuments(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()

	// Apply the same project age + permanent-ID filter as CollectTodolists
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
		taskCtx.GetLogger().Info("No project age limit configured, collecting documents from all projects")
		err = db.All(&projects,
			dal.From(&models.BasecampProject{}),
			dal.Where("connection_id = ? AND vault_ids != ''", data.Options.ConnectionId),
		)
	} else {
		cutoffDate := data.now().AddDate(0, -ageLimitMonths, 0)
		taskCtx.GetLogger().Info("Project age limit: %d months (cutoff: %s)", ageLimitMonths, cutoffDate.Format("2006-01-02"))

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

	// Also include sub-vaults discovered by CollectVaults
	var subVaults []models.BasecampVault
	if err := db.All(&subVaults,
		dal.From(&models.BasecampVault{}),
		dal.Where("connection_id = ?", data.Options.ConnectionId),
	); err != nil {
		return err
	}

	totalVaults := len(rootVaults) + len(subVaults)
	if totalVaults == 0 {
		taskCtx.GetLogger().Info("No vaults found in synced projects, skipping document collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting documents from %d root vaults and %d sub-vaults", len(rootVaults), len(subVaults))

	iterator := api.NewQueueIterator()
	for _, v := range rootVaults {
		iterator.Push(v)
	}
	for _, v := range subVaults {
		iterator.Push(&VaultInput{VaultId: v.VaultId})
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_DOCUMENT_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15,
		UrlTemplate: fmt.Sprintf("%s/vaults/{{ .Input.VaultId }}/documents.json",
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
			var documents []json.RawMessage
			err := api.UnmarshalResponse(res, &documents)
			if err != nil {
				return nil, err
			}
			return documents, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
