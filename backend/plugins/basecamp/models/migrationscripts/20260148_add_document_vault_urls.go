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

type basecampScopeConfig20260148 struct {
	DocumentVaultUrls string `gorm:"type:text"`
}

func (basecampScopeConfig20260148) TableName() string {
	return "_tool_basecamp_scope_configs"
}

type addDocumentVaultUrls struct{}

func (*addDocumentVaultUrls) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampScopeConfig20260148{},
	)
}

func (*addDocumentVaultUrls) Version() uint64 {
	return 20260148000001
}

func (*addDocumentVaultUrls) Name() string {
	return "add document_vault_urls to basecamp_scope_configs"
}
