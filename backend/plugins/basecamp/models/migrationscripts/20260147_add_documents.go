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

type basecampDocument20260147 struct {
	common.NoPKModel
	ConnectionId uint64     `gorm:"primaryKey"`
	DocumentId   string     `gorm:"primaryKey;type:varchar(255)"`
	ProjectId    string     `gorm:"type:varchar(255);index"`
	VaultId      string     `gorm:"type:varchar(255);index"`
	Title        string     `gorm:"type:varchar(1000)"`
	CreatorId    string     `gorm:"type:varchar(255)"`
	CreatorName  string     `gorm:"type:varchar(255)"`
	CreatorEmail string     `gorm:"type:varchar(255)"`
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

func (basecampDocument20260147) TableName() string {
	return "_tool_basecamp_documents"
}

type addDocuments struct{}

func (*addDocuments) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampDocument20260147{},
	)
}

func (*addDocuments) Version() uint64 {
	return 20260147000001
}

func (*addDocuments) Name() string {
	return "add basecamp_documents table"
}
