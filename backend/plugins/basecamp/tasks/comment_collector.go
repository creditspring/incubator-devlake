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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_COMMENT_TABLE = "basecamp_comments"

var _ plugin.SubTaskEntryPoint = CollectComments

var CollectCommentsMeta = plugin.SubTaskMeta{
	Name:             "CollectComments",
	EntryPoint:       CollectComments,
	EnabledByDefault: true,
	Description:      "Collect comments from todos in Basecamp account",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// TodoInput represents a todo to iterate over for collecting comments
type TodoInput struct {
	TodoId    string
	ProjectId string
}

func CollectComments(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()

	// Query todos, applying age filter if configured
	clauses := []dal.Clause{
		dal.From(&models.BasecampTodo{}),
		dal.Where("connection_id = ?", data.Options.ConnectionId),
	}
	if data.ScopeConfig != nil && data.ScopeConfig.TodoAgeLimitMonths > 0 {
		cutoff := time.Now().AddDate(0, -data.ScopeConfig.TodoAgeLimitMonths, 0)
		clauses = append(clauses, dal.Where("updated_at >= ?", cutoff))
		taskCtx.GetLogger().Info("Comment collection filtered by todo age limit: %d months (cutoff: %s)",
			data.ScopeConfig.TodoAgeLimitMonths, cutoff.Format("2006-01-02"))
	}
	var todos []models.BasecampTodo
	err := db.All(&todos, clauses...)
	if err != nil {
		return err
	}

	if len(todos) == 0 {
		taskCtx.GetLogger().Info("No todos found, skipping comment collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting comments from %d todos", len(todos))

	// Create iterator from todos
	iterator := api.NewQueueIterator()
	for _, todo := range todos {
		iterator.Push(&TodoInput{
			TodoId:    todo.TodoId,
			ProjectId: todo.ProjectId,
		})
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_COMMENT_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15, // Basecamp returns 15 items per page
		UrlTemplate: fmt.Sprintf("%s/buckets/{{ .Input.ProjectId }}/recordings/{{ .Input.TodoId }}/comments.json",
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
			var comments []json.RawMessage
			err := api.UnmarshalResponse(res, &comments)
			if err != nil {
				return nil, err
			}
			return comments, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
