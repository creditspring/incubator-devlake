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

package tasks

import (
	"testing"

	"github.com/apache/incubator-devlake/plugins/basecamp/models"
	"github.com/stretchr/testify/assert"
)

func TestBuildVaultInputs(t *testing.T) {
	tests := []struct {
		name     string
		projects []models.BasecampProject
		expected []*VaultInput
	}{
		{
			name: "single project single vault",
			projects: []models.BasecampProject{
				{VaultIds: "2663047427"},
			},
			expected: []*VaultInput{{VaultId: "2663047427"}},
		},
		{
			name: "single project multiple vaults",
			projects: []models.BasecampProject{
				{VaultIds: "111,222,333"},
			},
			expected: []*VaultInput{
				{VaultId: "111"},
				{VaultId: "222"},
				{VaultId: "333"},
			},
		},
		{
			name: "multiple projects",
			projects: []models.BasecampProject{
				{VaultIds: "111"},
				{VaultIds: "222"},
			},
			expected: []*VaultInput{
				{VaultId: "111"},
				{VaultId: "222"},
			},
		},
		{
			name: "project with empty VaultIds produces no inputs",
			projects: []models.BasecampProject{
				{VaultIds: ""},
			},
			expected: nil,
		},
		{
			name:     "no projects",
			projects: []models.BasecampProject{},
			expected: nil,
		},
		{
			name: "vault IDs with surrounding whitespace are trimmed",
			projects: []models.BasecampProject{
				{VaultIds: " 111 , 222 "},
			},
			expected: []*VaultInput{
				{VaultId: "111"},
				{VaultId: "222"},
			},
		},
		{
			name: "mixed: some projects have vaults, some do not",
			projects: []models.BasecampProject{
				{VaultIds: ""},
				{VaultIds: "999"},
				{VaultIds: ""},
			},
			expected: []*VaultInput{{VaultId: "999"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildVaultInputs(tt.projects)
			assert.Equal(t, tt.expected, result)
		})
	}
}
