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
	"strconv"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = ExtractDocuments

var ExtractDocumentsMeta = plugin.SubTaskMeta{
	Name:             "ExtractDocuments",
	EntryPoint:       ExtractDocuments,
	EnabledByDefault: false,
	Description:      "Extract raw data into tool layer table _tool_basecamp_documents",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// BasecampApiDocument represents the Basecamp API response for a document
type BasecampApiDocument struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Creator   struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		EmailAddress string `json:"email_address"`
	} `json:"creator"`
	Bucket struct {
		ID int64 `json:"id"`
	} `json:"bucket"`
	Parent struct {
		ID int64 `json:"id"`
	} `json:"parent"`
}

func ExtractDocuments(taskCtx plugin.SubTaskContext) errors.Error {
	taskData := taskCtx.GetData().(*BasecampTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: BasecampApiParams{
				ConnectionId: taskData.Options.ConnectionId,
				AccountId:    taskData.AccountId,
			},
			Table: RAW_DOCUMENT_TABLE,
		},
		Extract: func(resData *api.RawData) ([]interface{}, errors.Error) {
			apiDoc := &BasecampApiDocument{}
			err := errors.Convert(json.Unmarshal(resData.Data, apiDoc))
			if err != nil {
				return nil, err
			}

			doc := &models.BasecampDocument{
				ConnectionId: taskData.Options.ConnectionId,
				DocumentId:   strconv.FormatInt(apiDoc.ID, 10),
				ProjectId:    strconv.FormatInt(apiDoc.Bucket.ID, 10),
				VaultId:      strconv.FormatInt(apiDoc.Parent.ID, 10),
				Title:        apiDoc.Title,
				CreatorId:    strconv.FormatInt(apiDoc.Creator.ID, 10),
				CreatorName:  apiDoc.Creator.Name,
				CreatorEmail: apiDoc.Creator.EmailAddress,
			}

			if apiDoc.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiDoc.CreatedAt); err == nil {
					doc.CreatedAt = &t
				}
			}
			if apiDoc.UpdatedAt != "" {
				if t, err := time.Parse(time.RFC3339Nano, apiDoc.UpdatedAt); err == nil {
					doc.UpdatedAt = &t
				}
			}

			return []interface{}{doc}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
