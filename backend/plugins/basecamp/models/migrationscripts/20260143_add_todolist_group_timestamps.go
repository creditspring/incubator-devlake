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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type basecampTodolistGroup20260143 struct {
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func (basecampTodolistGroup20260143) TableName() string {
	return "_tool_basecamp_todolist_groups"
}

type addTodolistGroupTimestamps struct{}

func (*addTodolistGroupTimestamps) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampTodolistGroup20260143{},
	)
}

func (*addTodolistGroupTimestamps) Version() uint64 {
	return 20260143000001
}

func (*addTodolistGroupTimestamps) Name() string {
	return "add created_at and updated_at to basecamp_todolist_groups for consistency"
}
