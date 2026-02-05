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
	"reflect"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ConvertComments

var ConvertCommentsMeta = plugin.SubTaskMeta{
	Name:             "ConvertComments",
	EntryPoint:       ConvertComments,
	EnabledByDefault: true,
	Description:      "Convert tool layer todo comments to domain layer issue comments",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertComments(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	taskData := taskCtx.GetData().(*BasecampTaskData)

	// Create ID generators
	commentIdGen := didgen.NewDomainIdGenerator(&models.BasecampTodoComment{})
	todoIdGen := didgen.NewDomainIdGenerator(&models.BasecampTodo{})

	// Query comments for this connection
	cursor, err := db.Cursor(
		dal.From(&models.BasecampTodoComment{}),
		dal.Where("connection_id = ?", taskData.Options.ConnectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := api.NewDataConverter(api.DataConverterArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_COMMENT_TABLE,
		},
		InputRowType: reflect.TypeOf(models.BasecampTodoComment{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			comment := inputRow.(*models.BasecampTodoComment)

			issueComment := &ticket.IssueComment{
				DomainEntity: domainlayer.DomainEntity{
					Id: commentIdGen.Generate(taskData.Options.ConnectionId, comment.CommentId),
				},
				IssueId:   todoIdGen.Generate(taskData.Options.ConnectionId, comment.TodoId),
				Body:      comment.Content,
				AccountId: comment.CreatorId,
			}

			// Set CreatedDate (required field)
			if comment.CreatedAt != nil {
				issueComment.CreatedDate = *comment.CreatedAt
			} else {
				issueComment.CreatedDate = time.Now()
			}

			// Set UpdatedDate (optional field)
			if comment.UpdatedAt != nil {
				issueComment.UpdatedDate = comment.UpdatedAt
			}

			return []interface{}{issueComment}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}
