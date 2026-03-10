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

var _ plugin.SubTaskEntryPoint = CleanupData

var CleanupDataMeta = plugin.SubTaskMeta{
	Name:             "CleanupData",
	EntryPoint:       CleanupData,
	EnabledByDefault: true,
	Description:      "Delete existing Basecamp data before sync to ensure a clean slate",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CleanupData(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*BasecampTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	connectionId := data.Options.ConnectionId
	logger.Info("Cleaning up Basecamp data for connection %d", connectionId)

	// Delete in reverse dependency order to avoid foreign key issues:
	// 1. Comments (depends on todos)
	// 2. Todos
	// 3. Todolist Groups
	// 4. Todolists
	// 5. Projects
	// 6. PR References
	// Note: Keep Accounts (scope), Connections, and ScopeConfigs

	tablesToClean := []dal.Tabler{
		&models.BasecampTodoComment{},
		&models.BasecampTodo{},
		&models.BasecampTodolistGroup{},
		&models.BasecampTodolist{},
		&models.BasecampProject{},
		// Note: PrReference is NOT cleaned here. It is managed by the
		// post-source LinkPrToTodo task which runs after all data sources
		// have synced. PrReferences use deterministic IDs (FNV-64a hash)
		// so BatchSave handles upserts correctly.
	}

	for _, table := range tablesToClean {
		err := db.Delete(table, dal.Where("connection_id = ?", connectionId))
		if err != nil {
			logger.Warn(err, "Failed to delete from %s", table.TableName())
			// Continue with other tables even if one fails
		} else {
			logger.Info("Deleted data from %s for connection %d", table.TableName(), connectionId)
		}
	}

	logger.Info("Cleanup completed for connection %d", connectionId)
	return nil
}
