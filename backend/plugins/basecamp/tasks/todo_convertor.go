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

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ConvertTodos

var ConvertTodosMeta = plugin.SubTaskMeta{
	Name:             "ConvertTodos",
	EntryPoint:       ConvertTodos,
	EnabledByDefault: true,
	Description:      "Convert tool layer todos to domain layer issues",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertTodos(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	taskData := taskCtx.GetData().(*BasecampTaskData)

	// Create ID generators
	todoIdGen := didgen.NewDomainIdGenerator(&models.BasecampTodo{})
	projectIdGen := didgen.NewDomainIdGenerator(&models.BasecampProject{})

	// Query todos for this connection
	cursor, err := db.Cursor(
		dal.From(&models.BasecampTodo{}),
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
			Table: RAW_TODO_TABLE,
		},
		InputRowType: reflect.TypeOf(models.BasecampTodo{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			todo := inputRow.(*models.BasecampTodo)

			// Map status: Basecamp uses "active" for open, completed=true for done
			status := ticket.TODO
			if todo.Completed {
				status = ticket.DONE
			}

			issue := &ticket.Issue{
				DomainEntity: domainlayer.DomainEntity{
					Id: todoIdGen.Generate(taskData.Options.ConnectionId, todo.TodoId),
				},
				Url:             todo.Url,
				IssueKey:        todo.TodoId,
				Title:           todo.Title,
				Status:          status,
				OriginalStatus:  todo.Status,
				Type:            ticket.TASK,
				OriginalType:    "todo",
				CreatorId:       todo.CreatorId,
				CreatorName:     todo.CreatorName,
				AssigneeId:      todo.AssigneeId,
				AssigneeName:    todo.AssigneeName,
				OriginalProject: todo.ProjectName,
				CreatedDate:     todo.CreatedAt,
				UpdatedDate:     todo.UpdatedAt,
				ResolutionDate:  todo.CompletedAt,
			}

			// Create BoardIssue to link issue to project
			boardIssue := &ticket.BoardIssue{
				BoardId: projectIdGen.Generate(taskData.Options.ConnectionId, todo.ProjectId),
				IssueId: issue.Id,
			}

			return []interface{}{issue, boardIssue}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}
