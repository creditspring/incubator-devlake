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
	"github.com/apache/incubator-devlake/core/models/common"
)

// BasecampTodolistGroup represents a group within a todolist
// Groups are containers for todos within a todolist, providing an additional level of organization
type BasecampTodolistGroup struct {
	common.NoPKModel `json:"-" mapstructure:"-"`

	ConnectionId uint64 `gorm:"primaryKey"`
	GroupId      string `gorm:"primaryKey;type:varchar(255)"`
	TodolistId   string `gorm:"index;type:varchar(255)"` // Parent todolist
	ProjectId    string `gorm:"index;type:varchar(255)"` // Parent project/bucket
	Name         string `gorm:"type:varchar(255)"`
	TodosUrl     string `gorm:"type:varchar(500)"` // URL to fetch todos in this group
}

func (BasecampTodolistGroup) TableName() string {
	return "_tool_basecamp_todolist_groups"
}
