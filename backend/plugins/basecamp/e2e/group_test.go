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

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/basecamp/impl"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/apache/incubator-devlake/plugins/basecamp/tasks"
)

func TestGroupDataFlow(t *testing.T) {
	var plugin impl.Basecamp
	dataflowTester := e2ehelper.NewDataFlowTester(t, "basecamp", plugin)

	taskData := &tasks.BasecampTaskData{
		Options: &tasks.BasecampOptions{
			ConnectionId: 1,
			AccountId:    "4450519",
		},
		AccountId: "4450519",
	}

	// Import raw API response data
	dataflowTester.ImportCsvIntoRawTable(
		"./raw_tables/_raw_basecamp_todolist_groups.csv",
		"_raw_basecamp_todolist_groups",
	)

	// Test extraction
	dataflowTester.FlushTabler(&models.BasecampTodolistGroup{})
	dataflowTester.Subtask(tasks.ExtractGroupsMeta, taskData)
	dataflowTester.VerifyTableWithOptions(
		models.BasecampTodolistGroup{},
		e2ehelper.TableOptions{
			CSVRelPath:  "./snapshot_tables/_tool_basecamp_todolist_groups.csv",
			IgnoreTypes: []interface{}{common.NoPKModel{}},
		},
	)
}
