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

type BasecampScopeConfig struct {
	common.ScopeConfig `mapstructure:",squash" json:",inline" gorm:"embedded"`
	// TodoUrls contains newline-separated Basecamp todo URLs to sync
	// Format: https://3.basecamp.com/{account}/buckets/{project_id}/todos/{todo_id}
	TodoUrls string `mapstructure:"todoUrls" json:"todoUrls" gorm:"type:text"`
	// PermanentProjectIds contains comma-separated project IDs that should always sync
	// regardless of the project age cutoff
	PermanentProjectIds string `mapstructure:"permanentProjectIds" json:"permanentProjectIds" gorm:"type:text"`
	// ProjectAgeLimitMonths is the number of months to look back for projects (0 = no limit)
	// Projects older than this won't have their todos synced (unless in PermanentProjectIds)
	ProjectAgeLimitMonths int `mapstructure:"projectAgeLimitMonths" json:"projectAgeLimitMonths"`
	// TodoAgeLimitMonths is the number of months to look back for todos (0 = no limit)
	// Todos older than this (based on updated_at) won't be synced
	TodoAgeLimitMonths int `mapstructure:"todoAgeLimitMonths" json:"todoAgeLimitMonths"`
}

func (BasecampScopeConfig) TableName() string {
	return "_tool_basecamp_scope_configs"
}

func (t *BasecampScopeConfig) SetConnectionId(c *BasecampScopeConfig, connectionId uint64) {
	c.ConnectionId = connectionId
	c.ScopeConfig.ConnectionId = connectionId
}
