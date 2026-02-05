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
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type basecampTodoEvent20260144 struct {
	common.NoPKModel
	ConnectionId uint64     `gorm:"primaryKey"`
	EventId      string     `gorm:"primaryKey;type:varchar(255)"`
	TodoId       string     `gorm:"type:varchar(255);index"`
	ProjectId    string     `gorm:"type:varchar(255);index"`
	Action       string     `gorm:"type:varchar(100)"`
	CreatorId    string     `gorm:"type:varchar(255)"`
	CreatorName  string     `gorm:"type:varchar(255)"`
	CreatedAt    *time.Time
	Details      string `gorm:"type:text"`
}

func (basecampTodoEvent20260144) TableName() string {
	return "_tool_basecamp_todo_events"
}

type addTodoEvents struct{}

func (*addTodoEvents) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampTodoEvent20260144{},
	)
}

func (*addTodoEvents) Version() uint64 {
	return 20260144000001
}

func (*addTodoEvents) Name() string {
	return "add basecamp_todo_events table"
}
