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
	"regexp"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/basecamp/models"
)

var _ plugin.SubTaskEntryPoint = LinkPrToTodo

var LinkPrToTodoMeta = plugin.SubTaskMeta{
	Name:             "LinkPrToTodo",
	EntryPoint:       LinkPrToTodo,
	EnabledByDefault: true,
	Description:      "Link PRs to Basecamp todos via URL in PR description",
	DependencyTables: []string{code.PullRequest{}.TableName(), ticket.Issue{}.TableName()},
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE, plugin.DOMAIN_TYPE_TICKET, plugin.DOMAIN_TYPE_CROSS},
	ProductTables:    []string{crossdomain.PullRequestIssue{}.TableName(), models.PrReference{}.TableName()},
}

// basecampTodoRegex matches Basecamp todo URLs
// Format: https://3.basecamp.com/{account}/buckets/{bucket_id}/todos/{todo_id}
var basecampTodoRegex = regexp.MustCompile(`https?://3\.basecamp\.com/\d+/buckets/(\d+)/todos/(\d+)`)

// urlRegex matches any http/https URL
var urlRegex = regexp.MustCompile(`https?://[^\s<>\[\]()'"]+`)

func LinkPrToTodo(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BasecampTaskData)
	logger := taskCtx.GetLogger()

	// Create ID generator for todos
	todoIdGen := didgen.NewDomainIdGenerator(&models.BasecampTodo{})

	// Query all PRs - we scan all PRs since Basecamp URLs could be in any PR
	cursor, err := db.Cursor(
		dal.From(&code.PullRequest{}),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	// Build a map of todo IDs for quick lookup
	var todos []models.BasecampTodo
	if err := db.All(&todos,
		dal.From(&models.BasecampTodo{}),
		dal.Where("connection_id = ?", data.Options.ConnectionId),
	); err != nil {
		return err
	}

	// Map todoId -> domainId for quick lookup
	todoIdMap := make(map[string]string)
	for _, todo := range todos {
		domainId := todoIdGen.Generate(data.Options.ConnectionId, todo.TodoId)
		todoIdMap[todo.TodoId] = domainId
	}

	logger.Info("Processing PRs: extracting URLs and linking to %d todos", len(todoIdMap))

	enricher, err := api.NewDataEnricher(api.DataEnricherArgs[code.PullRequest]{
		Ctx:   taskCtx,
		Name:  "link_pr_to_basecamp_todo",
		Input: cursor,
		Enrich: func(pr *code.PullRequest) ([]interface{}, errors.Error) {
			var results []interface{}

			// Track URLs we've already added to avoid duplicates within this PR
			seenUrls := make(map[string]bool)

			// Search for ALL URLs in title and description
			for _, text := range []string{pr.Title, pr.Description} {
				// Extract all URLs and store as PrReference
				urls := urlRegex.FindAllString(text, -1)
				for _, url := range urls {
					if !seenUrls[url] {
						seenUrls[url] = true
						results = append(results, &models.PrReference{
							ConnectionId:  data.Options.ConnectionId,
							PullRequestId: pr.Id,
							Url:           url,
						})
					}
				}

				// Create pull_request_issues links for Basecamp todos (existing logic)
				basecampMatches := basecampTodoRegex.FindAllStringSubmatch(text, -1)
				for _, match := range basecampMatches {
					if len(match) == 3 {
						// match[1] is bucket_id, match[2] is todo_id
						todoId := match[2]

						// Check if this todo exists in our database
						if domainId, exists := todoIdMap[todoId]; exists {
							// Check if link already exists to avoid duplicates
							var existing crossdomain.PullRequestIssue
							err := db.First(&existing,
								dal.Where("pull_request_id = ? AND issue_id = ?", pr.Id, domainId),
							)
							if err != nil || existing.PullRequestId == "" {
								// Create the link
								results = append(results, &crossdomain.PullRequestIssue{
									PullRequestId:  pr.Id,
									IssueId:        domainId,
									PullRequestKey: pr.PullRequestKey,
									IssueKey:       todoId,
								})
							}
						}
					}
				}
			}

			return results, nil
		},
	})
	if err != nil {
		return err
	}

	return enricher.Execute()
}
