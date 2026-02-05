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

var _ plugin.SubTaskEntryPoint = ExtractEvents

var ExtractEventsMeta = plugin.SubTaskMeta{
	Name:             "ExtractEvents",
	EntryPoint:       ExtractEvents,
	EnabledByDefault: true,
	Description:      "Extract raw data into tool layer table _tool_basecamp_todo_events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiEvent represents the API response for an event
type BasecampApiEvent struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	CreatedAt string `json:"created_at"`
	Creator   struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"creator"`
	RecordingId int64           `json:"recording_id"`
	Details     json.RawMessage `json:"details"`
}

// EventInput represents the input passed from collector
type EventInput struct {
	TodoId    string `json:"TodoId"`
	ProjectId string `json:"ProjectId"`
}

func ExtractEvents(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_EVENT_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiEvent := &BasecampApiEvent{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiEvent))
			if err != nil {
				return nil, err
			}

			// Parse input to get ProjectId and TodoId
			var input EventInput
			if err := errors.Convert(json.Unmarshal(resData.Input, &input)); err != nil {
				return nil, err
			}

			// Convert details to string
			detailsStr := ""
			if apiEvent.Details != nil {
				detailsStr = string(apiEvent.Details)
			}

			event := &models.BasecampTodoEvent{
				ConnectionId: taskData.Options.ConnectionId,
				EventId:      strconv.FormatInt(apiEvent.ID, 10),
				TodoId:       input.TodoId,
				ProjectId:    input.ProjectId,
				Action:       apiEvent.Action,
				CreatorId:    strconv.FormatInt(apiEvent.Creator.ID, 10),
				CreatorName:  apiEvent.Creator.Name,
				Details:      detailsStr,
			}

			// Parse timestamp
			if apiEvent.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, apiEvent.CreatedAt); err == nil {
					event.CreatedAt = &t
				}
			}

			return []interface{}{event}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
