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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/slack/apimodels"
)

const RAW_CHANNEL_MESSAGE_TABLE = "slack_channel_message"

var _ plugin.SubTaskEntryPoint = CollectChannelMessage

type ChannelInput struct {
	ChannelId string `json:"channel_id"`
}

func CollectChannelMessage(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*SlackTaskData)

	// Create stateful collector to get timeAfter from SyncPolicy
	apiCollector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx:     taskCtx,
		Options: data.Options,
		Table:   RAW_CHANNEL_MESSAGE_TABLE,
	})
	if err != nil {
		return err
	}

	// Build a single-item iterator for the specific channel passed in options
	iterator := helper.NewQueueIterator()
	iterator.Push(&ChannelInput{ChannelId: data.Options.ChannelId})

	pageSize := 100
	err = apiCollector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   data.ApiClient,
		Input:       iterator,
		UrlTemplate: "conversations.history",
		PageSize:    pageSize,
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			res := apimodels.SlackChannelMessageApiResult{}
			err := helper.UnmarshalResponse(prevPageResponse, &res)
			if err != nil {
				return nil, err
			}
			if res.ResponseMetadata.NextCursor == "" {
				return nil, helper.ErrFinishCollect
			}
			return res.ResponseMetadata.NextCursor, nil
		},
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			input := reqData.Input.(*ChannelInput)
			query := url.Values{}
			query.Set("channel", input.ChannelId)
			query.Set("limit", strconv.Itoa(pageSize))
			if pageToken, ok := reqData.CustomData.(string); ok && pageToken != "" {
				query.Set("cursor", reqData.CustomData.(string))
			}
			// Add oldest parameter if timeAfter is set in SyncPolicy
			if apiCollector.GetSince() != nil {
				query.Set("oldest", fmt.Sprintf("%d", apiCollector.GetSince().Unix()))
			}
			return query, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body := &apimodels.SlackChannelMessageApiResult{}
			err := helper.UnmarshalResponse(res, body)
			if err != nil {
				return nil, err
			}
			return body.Messages, nil
		},
	})
	if err != nil {
		return err
	}

	return apiCollector.Execute()
}

var CollectChannelMessageMeta = plugin.SubTaskMeta{
	Name:             "collectChannelMessage",
	EntryPoint:       CollectChannelMessage,
	EnabledByDefault: true,
	Description:      "Collect channel message from Slack api",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}
