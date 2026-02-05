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
	"github.com/apache/incubator-devlake/plugins/basecamp/models/migrationscripts/archived"
)

type addAccountScope struct{}

func (*addAccountScope) Up(basicRes context.BasicRes) errors.Error {
	// Create the account scope table and todolist table
	if err := migrationhelper.AutoMigrateTables(
		basicRes,
		&archived.BasecampAccount{},
		&archived.BasecampTodolist{},
	); err != nil {
		return err
	}

	// Add todoset_id column to projects table for collecting all todos
	db := basicRes.GetDal()
	if err := db.AutoMigrate(&projectWithTodoset{}); err != nil {
		return errors.Default.Wrap(err, "error adding todoset_id to projects")
	}

	return nil
}

// projectWithTodoset adds todoset_id to existing project table
type projectWithTodoset struct {
	TodosetId string `json:"todosetId" gorm:"type:varchar(255)"`
}

func (projectWithTodoset) TableName() string {
	return "_tool_basecamp_projects"
}

func (*addAccountScope) Version() uint64 {
	return 20260130000001
}

func (*addAccountScope) Name() string {
	return "basecamp add account scope"
}
