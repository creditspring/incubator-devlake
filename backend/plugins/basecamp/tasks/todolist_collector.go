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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

const RAW_TODOLIST_TABLE = "basecamp_todolists"

var _ plugin.SubTaskEntryPoint = CollectTodolists

var CollectTodolistsMeta = plugin.SubTaskMeta{
	Name:             "CollectTodolists",
	EntryPoint:       CollectTodolists,
	EnabledByDefault: true,
	Description:      "Collect todolists from all projects in Basecamp account",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// ProjectInput represents a project to iterate over for collecting todolists
type ProjectInput struct {
	ProjectId string
	TodosetId string
}

func CollectTodolists(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()

	// Get project age limit from scope config (0 = no limit)
	ageLimitMonths := 0
	if data.ScopeConfig != nil && data.ScopeConfig.ProjectAgeLimitMonths > 0 {
		ageLimitMonths = data.ScopeConfig.ProjectAgeLimitMonths
	}

	// Parse permanent project IDs from scope config
	var permanentIds []string
	if data.ScopeConfig != nil && data.ScopeConfig.PermanentProjectIds != "" {
		for _, id := range strings.Split(data.ScopeConfig.PermanentProjectIds, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				permanentIds = append(permanentIds, trimmed)
			}
		}
	}

	// Query projects with todoset IDs
	var projects []models.BasecampProject
	var err errors.Error

	// If age limit is 0, fetch all projects (no date filter)
	if ageLimitMonths == 0 {
		taskCtx.GetLogger().Info("No project age limit configured, collecting todolists from all projects")
		err = db.All(&projects,
			dal.From(&models.BasecampProject{}),
			dal.Where("connection_id = ? AND todoset_ids != '' AND project_id = '45992106'", data.Options.ConnectionId),
		)
	} else {
		cutoffDate := time.Now().AddDate(0, -ageLimitMonths, 0)
		taskCtx.GetLogger().Info("Project age limit: %d months (cutoff: %s)", ageLimitMonths, cutoffDate.Format("2006-01-02"))

		if len(permanentIds) > 0 {
			taskCtx.GetLogger().Info("Including %d permanent project IDs in todolist collection", len(permanentIds))
			err = db.All(&projects,
				dal.From(&models.BasecampProject{}),
				dal.Where("connection_id = ? AND todoset_ids != '' AND project_id = '45992106' AND (created_at > ? OR project_id IN ?)",
					data.Options.ConnectionId, cutoffDate, permanentIds),
			)
		} else {
			err = db.All(&projects,
				dal.From(&models.BasecampProject{}),
				dal.Where("connection_id = ? AND todoset_ids != '' AND project_id = '45992106' AND created_at > ?",
					data.Options.ConnectionId, cutoffDate),
			)
		}
	}
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		taskCtx.GetLogger().Info("No projects with todosets found matching criteria, skipping todolist collection")
		return nil
	}

	taskCtx.GetLogger().Info("Collecting todolists from %d projects", len(projects))

	// Create iterator from projects - iterate over ALL todosets per project
	iterator := api.NewQueueIterator()
	for _, p := range projects {
		for _, id := range strings.Split(p.TodosetIds, ",") {
			if todosetId := strings.TrimSpace(id); todosetId != "" {
				iterator.Push(&ProjectInput{
					ProjectId: p.ProjectId,
					TodosetId: todosetId,
				})
			}
		}
	}

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: data.Options.ConnectionId,
				AccountId:    data.AccountId,
			},
			Table: RAW_TODOLIST_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		PageSize:  15, // Basecamp returns 15 items per page
		UrlTemplate: fmt.Sprintf("%s/buckets/{{ .Input.ProjectId }}/todosets/{{ .Input.TodosetId }}/todolists.json",
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
			var todolists []json.RawMessage
			err := api.UnmarshalResponse(res, &todolists)
			if err != nil {
				return nil, err
			}
			return todolists, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
