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

// BasecampProject represents a Basecamp project (bucket)
// Projects are collected automatically when syncing an account scope
type BasecampProject struct {
	common.NoPKModel `json:"-" mapstructure:"-"`
	ConnectionId     uint64     `json:"connectionId" mapstructure:"connectionId" gorm:"primaryKey"`
	ProjectId        string     `json:"projectId" mapstructure:"projectId" gorm:"primaryKey;type:varchar(255)"`
	Name string `json:"name" mapstructure:"name" gorm:"type:varchar(255)"`
	// TodosetIds stores all todoset IDs as comma-separated values (projects can have multiple todosets)
	TodosetIds string `json:"todosetIds" mapstructure:"todosetIds" gorm:"type:text"`
	// VaultIds stores all vault IDs as comma-separated values (the "Docs & Files" dock item per project)
	VaultIds  string     `json:"vaultIds" mapstructure:"vaultIds" gorm:"type:text"`
	CreatedAt *time.Time `json:"createdAt" mapstructure:"createdAt"`
}

func (BasecampProject) TableName() string {
	return "_tool_basecamp_projects"
}

// BasecampApiParams holds parameters for API calls and raw data storage
type BasecampApiParams struct {
	ConnectionId uint64
	AccountId    string
}
