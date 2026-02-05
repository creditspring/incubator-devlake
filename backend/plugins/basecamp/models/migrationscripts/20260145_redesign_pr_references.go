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

type prReference20260145 struct {
	common.NoPKModel
	Id            uint64 `gorm:"primaryKey;autoIncrement"`
	ConnectionId  uint64 `gorm:"index"`
	PullRequestId string `gorm:"type:varchar(255);index"`
	Url           string `gorm:"type:text"`
}

func (prReference20260145) TableName() string {
	return "_tool_pr_references"
}

type redesignPrReferences struct{}

func (*redesignPrReferences) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Drop the existing table with composite primary key
	err := db.DropTables(&prReference20260145{})
	if err != nil {
		return err
	}

	// Recreate with new schema (auto-increment ID, TEXT url)
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&prReference20260145{},
	)
}

func (*redesignPrReferences) Version() uint64 {
	return 20260145000001
}

func (*redesignPrReferences) Name() string {
	return "redesign pr_references table with auto-increment primary key"
}
