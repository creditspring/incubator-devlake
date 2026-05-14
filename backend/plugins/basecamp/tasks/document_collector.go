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
	"regexp"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const RAW_DOCUMENT_TABLE = "basecamp_documents"

var _ plugin.SubTaskEntryPoint = CollectDocuments

var CollectDocumentsMeta = plugin.SubTaskMeta{
	Name:             "CollectDocuments",
	EntryPoint:       CollectDocuments,
	EnabledByDefault: false,
	Description:      "Collect documents from configured Basecamp vaults",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// vaultURLRegex matches Basecamp vault URLs:
// https://3.basecamp.com/{account}/buckets/{project_id}/vaults/{vault_id}
var vaultURLRegex = regexp.MustCompile(`https?://3\.basecamp\.com/(\d+)/buckets/(\d+)/vaults/(\d+)`)

// VaultInput holds the vault ID needed to collect documents from one vault.
// project_id is not needed in the URL — it comes back in the response via bucket.id.
type VaultInput struct {
	VaultId string
}

// parseVaultURLs parses newline-separated Basecamp vault URLs and returns VaultInputs.
func parseVaultURLs(raw string) []*VaultInput {
	var inputs []*VaultInput
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := vaultURLRegex.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}
		inputs = append(inputs, &VaultInput{VaultId: m[3]})
	}
	return inputs
}

func CollectDocuments(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)

	if data.ScopeConfig == nil || data.ScopeConfig.DocumentVaultUrls == "" {
		taskCtx.GetLogger().Info("No document vault URLs configured, skipping document collection")
		return nil
	}

	vaults := parseVaultURLs(data.ScopeConfig.DocumentVaultUrls)
	if len(vaults) == 0 {
		taskCtx.GetLogger().Info("No valid vault URLs found in DocumentVaultUrls, skipping document collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting documents from %d vaults", len(vaults))

	iterator := api.NewQueueIterator()
	for _, v := range vaults {
		iterator.Push(v)
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
