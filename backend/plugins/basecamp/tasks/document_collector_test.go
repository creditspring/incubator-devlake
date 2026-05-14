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

	"github.com/stretchr/testify/assert"
)

func TestParseVaultURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []*VaultInput
	}{
		{
			name:     "single URL",
			input:    "https://3.basecamp.com/4450519/buckets/15909529/vaults/2663047427",
			expected: []*VaultInput{{VaultId: "2663047427"}},
		},
		{
			name: "multiple URLs newline separated",
			input: "https://3.basecamp.com/4450519/buckets/15909529/vaults/2663047427\nhttps://3.basecamp.com/4450519/buckets/99999999/vaults/11111111",
			expected: []*VaultInput{
				{VaultId: "2663047427"},
				{VaultId: "11111111"},
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "blank lines ignored",
			input:    "\n\nhttps://3.basecamp.com/4450519/buckets/15909529/vaults/2663047427\n\n",
			expected: []*VaultInput{{VaultId: "2663047427"}},
		},
		{
			name:     "invalid URL skipped",
			input:    "not-a-url\nhttps://3.basecamp.com/4450519/buckets/15909529/vaults/2663047427",
			expected: []*VaultInput{{VaultId: "2663047427"}},
		},
		{
			name:     "non-vault Basecamp URL skipped",
			input:    "https://3.basecamp.com/4450519/buckets/15909529/todos/12345",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVaultURLs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
