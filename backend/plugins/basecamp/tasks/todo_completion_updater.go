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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = UpdateTodoCompletion

var UpdateTodoCompletionMeta = plugin.SubTaskMeta{
	Name:             "UpdateTodoCompletion",
	EntryPoint:       UpdateTodoCompletion,
	EnabledByDefault: true,
	Description:      "Update todo CompletedAt timestamps from event data",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func UpdateTodoCompletion(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Get all completed todos for this connection
	var todos []models.BasecampTodo
	err := db.All(&todos,
		dal.From(&models.BasecampTodo{}),
		dal.Where("connection_id = ? AND completed = ?", data.Options.ConnectionId, true),
	)
	if err != nil {
		return err
	}

	if len(todos) == 0 {
		logger.Info("No completed todos found, skipping completion update")
		return nil
	}

	logger.Info("Updating completion timestamps for %d completed todos", len(todos))

	updatedCount := 0
	for _, todo := range todos {
		// Find the most recent "completed" event for this todo
		var events []models.BasecampTodoEvent
		err := db.All(&events,
			dal.From(&models.BasecampTodoEvent{}),
			dal.Where("connection_id = ? AND todo_id = ? AND action = ?",
				data.Options.ConnectionId, todo.TodoId, "completed"),
			dal.Orderby("created_at DESC"),
			dal.Limit(1),
		)
		if err != nil {
			logger.Warn(err, "Failed to query events for todo %s", todo.TodoId)
			continue
		}

		if len(events) == 0 {
			// No completed event found, skip
			continue
		}

		completedEvent := events[0]
		if completedEvent.CreatedAt == nil {
			continue
		}

		// Update the todo's CompletedAt if different
		if todo.CompletedAt == nil || !todo.CompletedAt.Equal(*completedEvent.CreatedAt) {
			err = db.UpdateColumn(
				&models.BasecampTodo{},
				"completed_at",
				completedEvent.CreatedAt,
				dal.Where("connection_id = ? AND todo_id = ?",
					data.Options.ConnectionId, todo.TodoId),
			)
			if err != nil {
				logger.Warn(err, "Failed to update CompletedAt for todo %s", todo.TodoId)
				continue
			}
			updatedCount++
		}
	}

	logger.Info("Updated CompletedAt for %d todos from event timestamps", updatedCount)
	return nil
}
