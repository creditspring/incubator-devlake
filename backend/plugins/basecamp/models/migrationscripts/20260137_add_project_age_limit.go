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
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type basecampScopeConfig20260137 struct {
	ProjectAgeLimitMonths int `gorm:"default:18"`
}

func (basecampScopeConfig20260137) TableName() string {
	return "_tool_basecamp_scope_configs"
}

type addProjectAgeLimit struct{}

func (*addProjectAgeLimit) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampScopeConfig20260137{},
	)
}

func (*addProjectAgeLimit) Version() uint64 {
	return 20260137000001
}

func (*addProjectAgeLimit) Name() string {
	return "add project_age_limit_months to basecamp_scope_configs"
}
