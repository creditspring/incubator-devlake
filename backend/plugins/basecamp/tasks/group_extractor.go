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

var _ plugin.SubTaskEntryPoint = ExtractGroups

var ExtractGroupsMeta = plugin.SubTaskMeta{
	Name:             "ExtractGroups",
	EntryPoint:       ExtractGroups,
	EnabledByDefault: true,
	Description:      "Extract raw todolist groups into tool layer table _tool_basecamp_todolist_groups",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiGroup represents the API response for a todolist group
type BasecampApiGroup struct {
	ID     int64 `json:"id"`
	Name   string `json:"name"`
	Parent struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"parent"`
	Bucket struct {
		ID int64 `json:"id"`
	} `json:"bucket"`
	TodosUrl string `json:"todos_url"`
}

func ExtractGroups(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_GROUP_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiGroup := &BasecampApiGroup{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiGroup))
			if err != nil {
				return nil, err
			}

			group := &models.BasecampTodolistGroup{
				ConnectionId: taskData.Options.ConnectionId,
				GroupId:      strconv.FormatInt(apiGroup.ID, 10),
				TodolistId:   strconv.FormatInt(apiGroup.Parent.ID, 10),
				ProjectId:    strconv.FormatInt(apiGroup.Bucket.ID, 10),
				Name:         apiGroup.Name,
				TodosUrl:     apiGroup.TodosUrl,
			}

			return []interface{}{group}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
