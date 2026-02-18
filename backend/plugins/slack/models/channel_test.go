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

package models

import (
	"encoding/json"
	"testing"

	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlackChannelDecodeMapStruct verifies that DecodeMapStruct correctly
// maps snake_case JSON keys to SlackChannel struct fields via mapstructure tags.
// This is the code path used by PUT /plugins/slack/connections/:id/scopes.
func TestSlackChannelDecodeMapStruct(t *testing.T) {
	// Simulate the JSON payload that arrives via PUT /scopes
	rawJSON := `{
		"data": [{
			"connectionId": 1,
			"id": "C07NDG25B0E",
			"name": "incident-triage",
			"is_channel": true,
			"is_group": false,
			"is_im": false,
			"is_mpim": false,
			"is_private": true,
			"created": 1727093173,
			"is_archived": false,
			"is_general": false,
			"unlinked": 0,
			"name_normalized": "incident-triage",
			"is_shared": false,
			"is_org_shared": false,
			"is_pending_ext_shared": false,
			"context_team_id": "T2BVAS22E",
			"updated": 1749539377862,
			"creator": "U027L8B3TQA",
			"is_ext_shared": false,
			"is_member": true,
			"num_members": 25
		}]
	}`

	// Parse JSON into a map (same as the framework does with request body)
	var body map[string]interface{}
	err := json.Unmarshal([]byte(rawJSON), &body)
	require.NoError(t, err)

	// Decode using DecodeMapStruct (the same function used by PutMultipleCb)
	var req struct {
		Data []*SlackChannel `json:"data"`
	}
	decodeErr := helper.DecodeMapStruct(body, &req, false)
	require.Nil(t, decodeErr)
	require.Len(t, req.Data, 1)

	ch := req.Data[0]

	// Verify all fields were correctly mapped
	assert.Equal(t, uint64(1), ch.ConnectionId)
	assert.Equal(t, "C07NDG25B0E", ch.Id)
	assert.Equal(t, "incident-triage", ch.Name)
	assert.True(t, ch.IsChannel, "is_channel should be true")
	assert.False(t, ch.IsGroup, "is_group should be false")
	assert.False(t, ch.IsIm, "is_im should be false")
	assert.False(t, ch.IsMpim, "is_mpim should be false")
	assert.True(t, ch.IsPrivate, "is_private should be true")
	assert.Equal(t, 1727093173, ch.Created)
	assert.False(t, ch.IsArchived, "is_archived should be false")
	assert.False(t, ch.IsGeneral, "is_general should be false")
	assert.Equal(t, 0, ch.Unlinked)
	assert.Equal(t, "incident-triage", ch.NameNormalized)
	assert.False(t, ch.IsShared, "is_shared should be false")
	assert.False(t, ch.IsOrgShared, "is_org_shared should be false")
	assert.False(t, ch.IsPendingExtShared, "is_pending_ext_shared should be false")
	assert.Equal(t, "T2BVAS22E", ch.ContextTeamId)
	assert.Equal(t, int64(1749539377862), ch.Updated)
	assert.Equal(t, "U027L8B3TQA", ch.Creator)
	assert.False(t, ch.IsExtShared, "is_ext_shared should be false")
	assert.True(t, ch.IsMember, "is_member should be true")
	assert.Equal(t, 25, ch.NumMembers)
}

// TestSlackChannelDecodeMapStructPublicChannel verifies decoding of a public channel
// to ensure non-private channels also work correctly.
func TestSlackChannelDecodeMapStructPublicChannel(t *testing.T) {
	rawJSON := `{
		"data": [{
			"connectionId": 1,
			"id": "CPBVAGSBT",
			"name": "engineering_humans",
			"is_channel": true,
			"is_group": false,
			"is_im": false,
			"is_mpim": false,
			"is_private": false,
			"created": 1571397872,
			"is_archived": false,
			"is_general": false,
			"unlinked": 0,
			"name_normalized": "engineering_humans",
			"is_shared": false,
			"is_org_shared": false,
			"is_pending_ext_shared": false,
			"context_team_id": "T2BVAS22E",
			"updated": 1749734552339,
			"creator": "U5KNGR604",
			"is_ext_shared": false,
			"is_member": false,
			"num_members": 102
		}]
	}`

	var body map[string]interface{}
	err := json.Unmarshal([]byte(rawJSON), &body)
	require.NoError(t, err)

	var req struct {
		Data []*SlackChannel `json:"data"`
	}
	decodeErr := helper.DecodeMapStruct(body, &req, false)
	require.Nil(t, decodeErr)
	require.Len(t, req.Data, 1)

	ch := req.Data[0]

	assert.Equal(t, "CPBVAGSBT", ch.Id)
	assert.Equal(t, "engineering_humans", ch.Name)
	assert.True(t, ch.IsChannel, "is_channel should be true")
	assert.False(t, ch.IsPrivate, "is_private should be false")
	assert.Equal(t, "engineering_humans", ch.NameNormalized)
	assert.Equal(t, 102, ch.NumMembers)
	assert.False(t, ch.IsMember, "is_member should be false")
}
