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

type removeTodosetId struct{}

func (*removeTodosetId) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// First, migrate any existing data from todoset_id to todoset_ids where todoset_ids is empty
	err := db.Exec(`
		UPDATE _tool_basecamp_projects
		SET todoset_ids = todoset_id
		WHERE (todoset_ids IS NULL OR todoset_ids = '')
		  AND todoset_id IS NOT NULL
		  AND todoset_id != ''
	`)
	if err != nil {
		return err
	}

	// Then drop the todoset_id column
	return db.DropColumns("_tool_basecamp_projects", "todoset_id")
}

func (*removeTodosetId) Version() uint64 {
	return 20260139000001
}

func (*removeTodosetId) Name() string {
	return "remove todoset_id column from basecamp_projects, use todoset_ids only"
}
