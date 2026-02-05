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

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type BasecampTodo struct {
	common.NoPKModel `json:"-" mapstructure:"-"`
	ConnectionId     uint64     `json:"connectionId" gorm:"primaryKey"`
	TodoId           string     `json:"todoId" gorm:"primaryKey;type:varchar(255)"`
	ProjectId        string     `json:"projectId" gorm:"type:varchar(255);index"`
	ProjectName      string     `json:"projectName" gorm:"type:varchar(255)"`
	TodoListId       string     `json:"todoListId" gorm:"type:varchar(255)"`
	Title            string     `json:"title" gorm:"type:text"`
	Status           string     `json:"status" gorm:"type:varchar(100)"`
	Completed        bool       `json:"completed"`
	Url              string     `json:"url" gorm:"type:varchar(500)"`
	CreatorId        string     `json:"creatorId" gorm:"type:varchar(255)"`
	CreatorName      string     `json:"creatorName" gorm:"type:varchar(255)"`
	AssigneeId       string     `json:"assigneeId" gorm:"type:varchar(255)"`
	AssigneeName     string     `json:"assigneeName" gorm:"type:varchar(255)"`
	CreatedAt        *time.Time `json:"createdAt"`
	UpdatedAt        *time.Time `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt"`
}

func (BasecampTodo) TableName() string {
	return "_tool_basecamp_todos"
}
