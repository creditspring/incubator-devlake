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

var _ plugin.SubTaskEntryPoint = ExtractTodolists

var ExtractTodolistsMeta = plugin.SubTaskMeta{
	Name:             "ExtractTodolists",
	EntryPoint:       ExtractTodolists,
	EnabledByDefault: true,
	Description:      "Extract raw todolists into tool layer table _tool_basecamp_todolists",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiTodolist represents the API response for a todolist
type BasecampApiTodolist struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	TodosUrl string `json:"todos_url"`
	Bucket   struct {
		ID int64 `json:"id"`
	} `json:"bucket"`
}

func ExtractTodolists(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_TODOLIST_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiTodolist := &BasecampApiTodolist{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiTodolist))
			if err != nil {
				return nil, err
			}

			todolist := &models.BasecampTodolist{
				ConnectionId: taskData.Options.ConnectionId,
				TodolistId:   strconv.FormatInt(apiTodolist.ID, 10),
				ProjectId:    strconv.FormatInt(apiTodolist.Bucket.ID, 10),
				Name:         apiTodolist.Name,
				TodosUrl:     apiTodolist.TodosUrl,
			}

			return []interface{}{todolist}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
