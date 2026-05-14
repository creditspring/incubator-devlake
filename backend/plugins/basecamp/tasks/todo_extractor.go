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
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ExtractTodos

var ExtractTodosMeta = plugin.SubTaskMeta{
	Name:             "ExtractTodos",
	EntryPoint:       ExtractTodos,
	EnabledByDefault: true,
	Description:      "Extract raw data into tool layer table _tool_basecamp_todos",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiTodo represents the API response for a todo
type BasecampApiTodo struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Completed   bool   `json:"completed"`
	CompletedAt string `json:"completed_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	AppUrl      string `json:"app_url"`
	Bucket      struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"bucket"`
	Parent struct {
		ID int64 `json:"id"`
	} `json:"parent"`
	Creator struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"creator"`
	Assignees []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"assignees"`
}

func ExtractTodos(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	// Get todo age limit from scope config (default: 0 = no limit)
	todoAgeLimitMonths := 0
	if taskData.ScopeConfig != nil && taskData.ScopeConfig.TodoAgeLimitMonths > 0 {
		todoAgeLimitMonths = taskData.ScopeConfig.TodoAgeLimitMonths
	}

	var cutoffDate time.Time
	if todoAgeLimitMonths > 0 {
		cutoffDate = time.Now().AddDate(0, -todoAgeLimitMonths, 0)
		taskCtx.GetLogger().Info("Todo age limit: %d months (cutoff: %s)", todoAgeLimitMonths, cutoffDate.Format("2006-01-02"))
	}

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_TODO_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiTodo := &BasecampApiTodo{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiTodo))
			if err != nil {
				return nil, err
			}

			// Filter by todo age limit (based on updated_at)
			if todoAgeLimitMonths > 0 && apiTodo.UpdatedAt != "" {
				if updatedAt, parseErr := time.Parse(time.RFC3339Nano, apiTodo.UpdatedAt); parseErr == nil {
					if updatedAt.Before(cutoffDate) {
						// Skip this todo - it's older than the cutoff
						return nil, nil
					}
				}
			}

			todo := &models.BasecampTodo{
				ConnectionId: taskData.Options.ConnectionId,
				TodoId:       strconv.FormatInt(apiTodo.ID, 10),
				ProjectId:    strconv.FormatInt(apiTodo.Bucket.ID, 10),
				ProjectName:  apiTodo.Bucket.Name,
				TodoListId:   strconv.FormatInt(apiTodo.Parent.ID, 10),
				Title:        apiTodo.Title,
				Status:       apiTodo.Status,
				Completed:    apiTodo.Completed,
				Url:          apiTodo.AppUrl,
				CreatorId:    strconv.FormatInt(apiTodo.Creator.ID, 10),
				CreatorName:  apiTodo.Creator.Name,
			}

			// Set first assignee if available
			if len(apiTodo.Assignees) > 0 {
				todo.AssigneeId = strconv.FormatInt(apiTodo.Assignees[0].ID, 10)
				todo.AssigneeName = apiTodo.Assignees[0].Name
			}

			// Parse timestamps
			if apiTodo.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiTodo.CreatedAt); err == nil {
					todo.CreatedAt = &t
				}
			}
			if apiTodo.UpdatedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiTodo.UpdatedAt); err == nil {
					todo.UpdatedAt = &t
				}
			}
			if apiTodo.CompletedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiTodo.CompletedAt); err == nil {
					todo.CompletedAt = &t
				}
			}

			// Note: Projects are created by project_extractor with todoset_id
			// Don't create/update projects here as it would overwrite the todoset_id

			return []interface{}{todo}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
