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

var _ plugin.SubTaskEntryPoint = ExtractComments

var ExtractCommentsMeta = plugin.SubTaskMeta{
	Name:             "ExtractComments",
	EntryPoint:       ExtractComments,
	EnabledByDefault: true,
	Description:      "Extract raw data into tool layer table _tool_basecamp_todo_comments",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiComment represents the API response for a comment
type BasecampApiComment struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Creator   struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		EmailAddress string `json:"email_address"`
	} `json:"creator"`
	Parent struct {
		ID int64 `json:"id"`
	} `json:"parent"`
	Bucket struct {
		ID int64 `json:"id"`
	} `json:"bucket"`
}

func ExtractComments(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_COMMENT_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiComment := &BasecampApiComment{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiComment))
			if err != nil {
				return nil, err
			}

			comment := &models.BasecampTodoComment{
				ConnectionId: taskData.Options.ConnectionId,
				CommentId:    strconv.FormatInt(apiComment.ID, 10),
				TodoId:       strconv.FormatInt(apiComment.Parent.ID, 10),
				ProjectId:    strconv.FormatInt(apiComment.Bucket.ID, 10),
				Content:      apiComment.Content,
				CreatorId:    strconv.FormatInt(apiComment.Creator.ID, 10),
				CreatorName:  apiComment.Creator.Name,
				CreatorEmail: apiComment.Creator.EmailAddress,
			}

			// Parse timestamps
			if apiComment.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, apiComment.CreatedAt); err == nil {
					comment.CreatedAt = &t
				}
			}
			if apiComment.UpdatedAt != "" {
				if t, err := time.Parse(time.RFC3339, apiComment.UpdatedAt); err == nil {
					comment.UpdatedAt = &t
				}
			}

			return []interface{}{comment}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
