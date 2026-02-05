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

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_TODO_TABLE = "basecamp_todos"

// TodoUrlPattern matches Basecamp todo URLs and extracts bucket_id and todo_id
// Format: https://3.basecamp.com/{account}/buckets/{bucket_id}/todos/{todo_id}
var TodoUrlPattern = regexp.MustCompile(`https?://3\.basecamp\.com/\d+/buckets/(\d+)/todos/(\d+)`)

var _ plugin.SubTaskEntryPoint = CollectTodos

var CollectTodosMeta = plugin.SubTaskMeta{
	Name:             "CollectTodos",
	EntryPoint:       CollectTodos,
	EnabledByDefault: true,
	Description:      "Collect todos from all todolists and todolist groups in Basecamp account",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// TodolistInput represents a todolist to iterate over for collecting todos
type TodolistInput struct {
	ProjectId     string
	TodolistId    string
	FetchComplete bool // If true, fetch completed todos; if false, fetch active todos
}

func CollectTodos(taskCtx plugin.SubTaskContext) errors.Error {
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

	// Query all todolist groups (groups are nested containers within todolists)
	var groups []models.BasecampTodolistGroup
	err = db.All(&groups,
		dal.From(&models.BasecampTodolistGroup{}),
		dal.Where("connection_id = ?", data.Options.ConnectionId),
	)
	if err != nil {
		return err
	}

	if len(todolists) == 0 && len(groups) == 0 {
		taskCtx.GetLogger().Info("No todolists or groups found, skipping todo collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting todos from %d todolists and %d groups (active + completed)", len(todolists), len(groups))

	// Create iterator from todolists and groups - push each twice: once for active, once for completed
	iterator := api.NewQueueIterator()

	// Add todolists to iterator
	for _, tl := range todolists {
		// First fetch active todos
		iterator.Push(&TodolistInput{
			ProjectId:     tl.ProjectId,
			TodolistId:    tl.TodolistId,
			FetchComplete: false,
		})
		// Then fetch completed todos
		iterator.Push(&TodolistInput{
			ProjectId:     tl.ProjectId,
			TodolistId:    tl.TodolistId,
			FetchComplete: true,
		})
	}

	// Add groups to iterator (groups use the same API pattern as todolists for fetching todos)
	for _, g := range groups {
		// First fetch active todos from group
		iterator.Push(&TodolistInput{
			ProjectId:     g.ProjectId,
			TodolistId:    g.GroupId, // Group ID is used like a todolist ID
			FetchComplete: false,
		})
		// Then fetch completed todos from group
		iterator.Push(&TodolistInput{
			ProjectId:     g.ProjectId,
			TodolistId:    g.GroupId,
			FetchComplete: true,
		})
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_TODO_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15, // Basecamp returns 15 items per page
		UrlTemplate: fmt.Sprintf("%s/buckets/{{ .Input.ProjectId }}/todolists/{{ .Input.TodolistId }}/todos.json",
			data.AccountId),
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			input := reqData.Input.(*TodolistInput)
			// Fetch completed todos if FetchComplete is true
			if input.FetchComplete {
				query.Set("completed", "true")
			}
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
			var todos []json.RawMessage
			err := api.UnmarshalResponse(res, &todos)
			if err != nil {
				return nil, err
			}
			return todos, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
