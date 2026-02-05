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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type todolistGroup20260133 struct {
	common.RawDataOrigin

	ConnectionId uint64 `gorm:"primaryKey"`
	GroupId      string `gorm:"primaryKey;type:varchar(255)"`
	TodolistId   string `gorm:"index;type:varchar(255)"`
	ProjectId    string `gorm:"index;type:varchar(255)"`
	Name         string `gorm:"type:varchar(255)"`
	TodosUrl     string `gorm:"type:varchar(500)"`
}

func (todolistGroup20260133) TableName() string {
	return "_tool_basecamp_todolist_groups"
}

type addTodolistGroups struct{}

func (*addTodolistGroups) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&todolistGroup20260133{},
	)
}

func (*addTodolistGroups) Version() uint64 {
	return 20260133000001
}

func (*addTodolistGroups) Name() string {
	return "add todolist_groups table to store nested groups within todolists"
}
