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

type prReference20260132 struct {
	common.NoPKModel
	ConnectionId  uint64 `gorm:"primaryKey"`
	PullRequestId string `gorm:"primaryKey;type:varchar(255)"`
	Url           string `gorm:"primaryKey;type:varchar(500)"`
}

func (prReference20260132) TableName() string {
	return "_tool_pr_references"
}

type addPrReferences struct{}

func (*addPrReferences) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&prReference20260132{},
	)
}

func (*addPrReferences) Version() uint64 {
	return 20260132000001
}

func (*addPrReferences) Name() string {
	return "add pr_references table to store all URLs found in PR descriptions"
}
