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

package api

import (
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"

	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/apache/incubator-devlake/plugins/basecamp/tasks"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/utils"

	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/helpers/srvhelper"
)

func MakePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	connectionId uint64,
	bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	connection, err := dsHelper.ConnSrv.FindByPk(connectionId)
	if err != nil {
		return nil, nil, err
	}
	scopeDetails, err := dsHelper.ScopeSrv.MapScopeDetails(connectionId, bpScopes)
	if err != nil {
		return nil, nil, err
	}
	plan, err := makePipelinePlanV200(subtaskMetas, scopeDetails, connection)
	if err != nil {
		return nil, nil, err
	}
	scopes, err := makeScopesV200(scopeDetails, connection)
	return plan, scopes, err
}

func makePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	scopeDetails []*srvhelper.ScopeDetail[models.BasecampAccount, models.BasecampScopeConfig],
	connection *models.BasecampConnection,
) (coreModels.PipelinePlan, errors.Error) {
	plan := make(coreModels.PipelinePlan, len(scopeDetails))
	for i, scopeDetail := range scopeDetails {
		stage := plan[i]
		if stage == nil {
			stage = coreModels.PipelineStage{}
		}

		scope, scopeConfig := scopeDetail.Scope, scopeDetail.ScopeConfig
		// construct task options for basecamp
		task, err := helper.MakePipelinePlanTask(
			"basecamp",
			subtaskMetas,
			scopeConfig.Entities,
			tasks.BasecampOptions{
				ConnectionId:  connection.ID,
				AccountId:     scope.AccountId,
				ScopeConfigId: scopeConfig.ID,
			},
		)
		if err != nil {
			return nil, err
		}
		stage = append(stage, task)
		plan[i] = stage
	}

	return plan, nil
}

// MakePostSourcePipelinePlan creates a plan for tasks that must run AFTER all
// data source plugins have completed. This ensures LinkPrToTodo can see PRs
// from all GitHub repos, not just those synced in the same pipeline stage.
func MakePostSourcePipelinePlan(
	connectionId uint64,
	bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, errors.Error) {
	connection, err := dsHelper.ConnSrv.FindByPk(connectionId)
	if err != nil {
		return nil, err
	}
	scopeDetails, err := dsHelper.ScopeSrv.MapScopeDetails(connectionId, bpScopes)
	if err != nil {
		return nil, err
	}

	// Create one task per scope with only LinkPrToTodo enabled
	plan := make(coreModels.PipelinePlan, len(scopeDetails))
	for i, scopeDetail := range scopeDetails {
		scope, scopeConfig := scopeDetail.Scope, scopeDetail.ScopeConfig

		optionsMap := map[string]interface{}{
			"connectionId":  connection.ID,
			"accountId":     scope.AccountId,
			"scopeConfigId": scopeConfig.ID,
		}
		plan[i] = coreModels.PipelineStage{
			&coreModels.PipelineTask{
				Plugin:   "basecamp",
				Subtasks: []string{"LinkPrToTodo"},
				Options:  optionsMap,
			},
		}
	}

	return plan, nil
}

func makeScopesV200(
	scopeDetails []*srvhelper.ScopeDetail[models.BasecampAccount, models.BasecampScopeConfig],
	connection *models.BasecampConnection,
) ([]plugin.Scope, errors.Error) {
	scopes := make([]plugin.Scope, 0, len(scopeDetails))

	idgen := didgen.NewDomainIdGenerator(&models.BasecampAccount{})
	for _, scopeDetail := range scopeDetails {
		scope, scopeConfig := scopeDetail.Scope, scopeDetail.ScopeConfig
		id := idgen.Generate(connection.ID, scope.AccountId)

		// add boards to scopes
		if utils.StringsContains(scopeConfig.Entities, plugin.DOMAIN_TYPE_TICKET) {
			scopes = append(scopes, ticket.NewBoard(id, scope.Name))
		}
	}

	return scopes, nil
}
