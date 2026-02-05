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
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type basecampConnection20260129 struct {
	ClientId       string     `gorm:"type:varchar(255)"`
	ClientSecret   string     `gorm:"type:text;serializer:encdec"`
	RefreshToken   string     `gorm:"type:text;serializer:encdec"`
	TokenExpiresAt *time.Time `gorm:"type:datetime"`
}

func (basecampConnection20260129) TableName() string {
	return "_tool_basecamp_connections"
}

type addOAuth2Fields struct{}

func (*addOAuth2Fields) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&basecampConnection20260129{},
	)
}

func (*addOAuth2Fields) Version() uint64 {
	return 20260129000001
}

func (*addOAuth2Fields) Name() string {
	return "add OAuth2 fields to basecamp_connections"
}
