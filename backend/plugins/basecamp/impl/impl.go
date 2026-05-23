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

package impl

import (
	"fmt"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/apache/incubator-devlake/plugins/basecamp/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/basecamp/tasks"
)

var _ interface {
	plugin.PluginTask
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginApi
	plugin.PluginModel
	plugin.PluginMigration
	plugin.CloseablePluginTask
	plugin.DataSourcePluginBlueprintV200
	plugin.PostSourcePlanProvider
} = (*Basecamp)(nil)

type Basecamp struct{}

func (p Basecamp) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p Basecamp) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.BasecampConnection{},
		&models.BasecampAccount{},
		&models.BasecampProject{},
		&models.BasecampTodolist{},
		&models.BasecampTodolistGroup{},
		&models.BasecampTodo{},
		&models.BasecampTodoComment{},
		&models.BasecampTodoEvent{},
		&models.BasecampDocument{},
		&models.BasecampVault{},
		&models.PrReference{},
		&models.BasecampScopeConfig{},
	}
}

func (p Basecamp) Description() string {
	return "To collect and enrich data from Basecamp"
}

func (p Basecamp) Name() string {
	return "basecamp"
}

func (p Basecamp) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CleanupDataMeta,          // Delete existing data before sync
		tasks.CollectProjectsMeta,      // GET /projects.json
		tasks.ExtractProjectsMeta,      // Parse projects, extract todoset IDs
		tasks.CollectTodolistsMeta,     // GET todolists for each project
		tasks.ExtractTodolistsMeta,     // Parse todolists
		tasks.CollectGroupsMeta,        // GET groups for each todolist
		tasks.ExtractGroupsMeta,        // Parse groups
		tasks.CollectTodosMeta,         // GET todos for each todolist AND group
		tasks.ExtractTodosMeta,         // Parse todos
		tasks.CollectCommentsMeta,      // GET comments for each todo
		tasks.ExtractCommentsMeta,      // Parse comments
		tasks.CollectEventsMeta,        // GET events for each todo
		tasks.ExtractEventsMeta,        // Parse events
		tasks.UpdateTodoCompletionMeta,  // Update CompletedAt from events
		tasks.CollectVaultsMeta,        // Recursively collect sub-vaults (folders)
		tasks.ExtractVaultsMeta,        // Parse sub-vaults
		tasks.CollectDocumentsMeta,     // GET documents from all vaults (root + sub)
		tasks.ExtractDocumentsMeta,     // Parse documents
		tasks.ConvertProjectsMeta,      // projects → domain boards
		tasks.ConvertTodosMeta,         // todos → domain issues
		tasks.ConvertCommentsMeta,      // comments → domain issue comments
		tasks.LinkPrToTodoMeta,         // Link PRs ↔ todos
	}
}

func (p Basecamp) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.BasecampOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "Basecamp plugin could not decode options")
	}
	if op.ConnectionId == 0 {
		return nil, errors.BadInput.New("basecamp connectionId is invalid")
	}

	connection := &models.BasecampConnection{}
	connectionHelper := helper.NewConnectionHelper(
		taskCtx,
		nil,
		p.Name(),
	)
	err = connectionHelper.FirstById(connection, op.ConnectionId)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error getting connection for Basecamp plugin")
	}

	apiClient, err := tasks.CreateApiClient(taskCtx, connection)
	if err != nil {
		return nil, err
	}

	// Load scope config if specified
	var scopeConfig *models.BasecampScopeConfig
	if op.ScopeConfigId != 0 {
		scopeConfig = &models.BasecampScopeConfig{}
		db := taskCtx.GetDal()
		err = db.First(scopeConfig, dal.Where("id = ?", op.ScopeConfigId))
		if err != nil {
			return nil, errors.Default.Wrap(err, "error loading scope config")
		}
	}

	// Use AccountId from options, fallback to connection's AccountId
	accountId := op.AccountId
	if accountId == "" {
		accountId = connection.AccountId
	}

	return &tasks.BasecampTaskData{
		Options:     &op,
		ApiClient:   apiClient,
		AccountId:   accountId,
		ScopeConfig: scopeConfig,
	}, nil
}

func (p Basecamp) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/basecamp"
}

func (p Basecamp) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p Basecamp) Connection() dal.Tabler {
	return &models.BasecampConnection{}
}

func (p Basecamp) Scope() plugin.ToolLayerScope {
	return &models.BasecampAccount{}
}

func (p Basecamp) ScopeConfig() dal.Tabler {
	return &models.BasecampScopeConfig{}
}

func (p Basecamp) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"test": {
			"POST": api.TestConnection,
		},
		"connections": {
			"POST": api.PostConnections,
			"GET":  api.ListConnections,
		},
		"connections/:connectionId": {
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
			"GET":    api.GetConnection,
		},
		"connections/:connectionId/test": {
			"POST": api.TestExistingConnection,
		},
		"connections/:connectionId/scope-configs": {
			"POST": api.PostScopeConfig,
			"GET":  api.GetScopeConfigList,
		},
		"connections/:connectionId/scope-configs/:scopeConfigId": {
			"PATCH":  api.PatchScopeConfig,
			"GET":    api.GetScopeConfig,
			"DELETE": api.DeleteScopeConfig,
		},
		"connections/:connectionId/scopes/:scopeId": {
			"GET":    api.GetScope,
			"PATCH":  api.PatchScope,
			"DELETE": api.DeleteScope,
		},
		"connections/:connectionId/scopes": {
			"GET": api.GetScopeList,
			"PUT": api.PutScopes,
		},
		"connections/:connectionId/remote-scopes": {
			"GET": api.RemoteScopes,
		},
		"scope-config/:scopeConfigId/projects": {
			"GET": api.GetProjectsByScopeConfig,
		},
	}
}

func (p Basecamp) MakeDataSourcePipelinePlanV200(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	return api.MakePipelinePlanV200(p.SubTaskMetas(), connectionId, scopes)
}

func (p Basecamp) MakePostSourcePipelinePlan(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, errors.Error) {
	return api.MakePostSourcePipelinePlan(connectionId, scopes)
}

func (p Basecamp) Close(taskCtx plugin.TaskContext) errors.Error {
	data, ok := taskCtx.GetData().(*tasks.BasecampTaskData)
	if !ok {
		return errors.Default.New(fmt.Sprintf("GetData failed when try to close %+v", taskCtx))
	}
	data.ApiClient.Release()
	return nil
}
