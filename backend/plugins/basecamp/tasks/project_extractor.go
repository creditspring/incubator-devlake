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
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ExtractProjects

var ExtractProjectsMeta = plugin.SubTaskMeta{
	Name:             "ExtractProjects",
	EntryPoint:       ExtractProjects,
	EnabledByDefault: true,
	Description:      "Extract raw projects into tool layer table _tool_basecamp_projects",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiProject represents the API response for a project
type BasecampApiProject struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Dock      []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	} `json:"dock"`
}

func ExtractProjects(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_PROJECT_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiProject := &BasecampApiProject{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiProject))
			if err != nil {
				return nil, err
			}

			project := &models.BasecampProject{
				ConnectionId: taskData.Options.ConnectionId,
				ProjectId:    strconv.FormatInt(apiProject.ID, 10),
				Name:         apiProject.Name,
			}

			// Parse created_at timestamp
			if apiProject.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiProject.CreatedAt); err == nil {
					project.CreatedAt = &t
				}
			}

			// Find all todoset IDs from the dock (projects can have multiple todosets)
			var todosetIds []string
			for _, dock := range apiProject.Dock {
				if dock.Name == "todoset" && dock.Enabled {
					todosetIds = append(todosetIds, strconv.FormatInt(dock.ID, 10))
				}
			}
			if len(todosetIds) > 0 {
				project.TodosetIds = strings.Join(todosetIds, ",")
			}

			return []interface{}{project}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
