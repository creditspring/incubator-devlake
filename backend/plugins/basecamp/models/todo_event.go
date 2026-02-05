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

type BasecampTodoEvent struct {
	common.NoPKModel `json:"-" mapstructure:"-"`
	ConnectionId     uint64     `json:"connectionId" gorm:"primaryKey"`
	EventId          string     `json:"eventId" gorm:"primaryKey;type:varchar(255)"`
	TodoId           string     `json:"todoId" gorm:"type:varchar(255);index"`
	ProjectId        string     `json:"projectId" gorm:"type:varchar(255);index"`
	Action           string     `json:"action" gorm:"type:varchar(100)"` // "completed", "uncompleted", "created", etc.
	CreatorId        string     `json:"creatorId" gorm:"type:varchar(255)"`
	CreatorName      string     `json:"creatorName" gorm:"type:varchar(255)"`
	CreatedAt        *time.Time `json:"createdAt"`
	Details          string     `json:"details" gorm:"type:text"` // JSON blob for action-specific data
}

func (BasecampTodoEvent) TableName() string {
	return "_tool_basecamp_todo_events"
}
