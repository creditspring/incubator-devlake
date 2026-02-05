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

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_GROUP_TABLE = "basecamp_todolist_groups"

var _ plugin.SubTaskEntryPoint = CollectGroups

var CollectGroupsMeta = plugin.SubTaskMeta{
	Name:             "CollectGroups",
	EntryPoint:       CollectGroups,
	EnabledByDefault: true,
	Description:      "Collect todolist groups from all todolists in Basecamp account",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// TodolistForGroupInput represents a todolist to iterate over for collecting groups
type TodolistForGroupInput struct {
	ProjectId  string
	TodolistId string
}

func CollectGroups(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()

	// Query all todolists
	var todolists []models.BasecampTodolist
	err := db.All(&todolists,
		dal.From(&models.BasecampTodolist{}),
		dal.Where("connection_id = ?", data.Options.ConnectionId),
	)
	if err != nil {
		return err
	}

	if len(todolists) == 0 {
		taskCtx.GetLogger().Info("No todolists found, skipping group collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting groups from %d todolists", len(todolists))

	// Create iterator from todolists
	iterator := api.NewQueueIterator()
	for _, tl := range todolists {
		iterator.Push(&TodolistForGroupInput{
			ProjectId:  tl.ProjectId,
			TodolistId: tl.TodolistId,
		})
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_GROUP_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15, // Basecamp returns 15 items per page
		UrlTemplate: fmt.Sprintf("%s/buckets/{{ .Input.ProjectId }}/todolists/{{ .Input.TodolistId }}/groups.json",
			data.AccountId),
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			// Use page number from CustomData if available (for pagination)
			if reqData.CustomData != nil {
				if page, ok := reqData.CustomData.(int); ok && page > 1 {
					query.Set("page", fmt.Sprintf("%d", page))
				}
			}
			return query, nil
		},
		GetNextPageCustomData: func(prevReqData *api.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			// Check Link header for next page URL
			nextURL := GetNextPageURL(prevPageResponse)
			if nextURL == "" {
				return nil, api.ErrFinishCollect
			}
			// Return the next page number
			nextPage := prevReqData.Pager.Page + 1
			return nextPage, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var groups []json.RawMessage
			err := api.UnmarshalResponse(res, &groups)
			if err != nil {
				return nil, err
			}
			return groups, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
