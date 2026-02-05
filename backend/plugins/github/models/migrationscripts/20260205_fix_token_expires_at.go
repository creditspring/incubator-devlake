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
)

type fixGithubTokenExpiresAt struct{}

func (*fixGithubTokenExpiresAt) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	err := db.Exec("ALTER TABLE _tool_github_connections MODIFY COLUMN token_expires_at DATETIME NULL")
	if err != nil {
		return err
	}
	return db.Exec("ALTER TABLE _tool_github_connections MODIFY COLUMN refresh_token_expires_at DATETIME NULL")
}

func (*fixGithubTokenExpiresAt) Version() uint64 {
	return 20260205000001
}

func (*fixGithubTokenExpiresAt) Name() string {
	return "fix token_expires_at and refresh_token_expires_at columns to allow NULL values"
}
