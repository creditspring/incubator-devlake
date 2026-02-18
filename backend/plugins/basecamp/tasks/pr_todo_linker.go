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
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"strconv"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreLog "github.com/apache/incubator-devlake/core/log"
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
// Captures: [1]=account_id, [2]=bucket_id, [3]=todo_id
var basecampTodoRegex = regexp.MustCompile(`https?://3\.basecamp\.com/(\d+)/buckets/(\d+)/todos/(\d+)`)

// urlRegex matches any http/https URL
var urlRegex = regexp.MustCompile(`https?://[^\s<>\[\]()'"]+`)

// generatePrReferenceId creates a deterministic ID from the composite key fields
// This ensures BatchSave properly deduplicates records
func generatePrReferenceId(connectionId uint64, pullRequestId string, url string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%d:%s:%s", connectionId, pullRequestId, url)))
	return h.Sum64()
}

// resolveTodoId resolves a possibly-moved todo by calling the Basecamp API.
// If the todo was moved, the API redirects and returns JSON with the current ID.
// Results are cached in resolvedIds to avoid redundant API calls.
func resolveTodoId(
	apiClient *api.ApiClient,
	logger coreLog.Logger,
	resolvedIds map[string]string,
	accountId, bucketId, todoId string,
) string {
	if resolved, ok := resolvedIds[todoId]; ok {
		return resolved
	}
	path := fmt.Sprintf("%s/buckets/%s/todos/%s.json", accountId, bucketId, todoId)
	resp, err := apiClient.Get(path, nil, nil)
	if err != nil {
		logger.Warn(err, "failed to resolve todo %s via API", todoId)
		resolvedIds[todoId] = ""
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		logger.Warn(nil, "API returned %d resolving todo %s", resp.StatusCode, todoId)
		resp.Body.Close()
		resolvedIds[todoId] = ""
		return ""
	}
	var apiTodo BasecampApiTodo
	if unmarshalErr := api.UnmarshalResponse(resp, &apiTodo); unmarshalErr != nil {
		logger.Warn(unmarshalErr, "failed to unmarshal API response for todo %s", todoId)
		resolvedIds[todoId] = ""
		return ""
	}
	resolved := strconv.FormatInt(apiTodo.ID, 10)
	resolvedIds[todoId] = resolved
	return resolved
}

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

	// Cache for API-resolved todo IDs (for moved todos)
	resolvedIds := make(map[string]string)

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
							Id:            generatePrReferenceId(data.Options.ConnectionId, pr.Id, url),
							ConnectionId:  data.Options.ConnectionId,
							PullRequestId: pr.Id,
							Url:           url,
						})
					}
				}

				// Create pull_request_issues links for Basecamp todos
				basecampMatches := basecampTodoRegex.FindAllStringSubmatch(text, -1)
				for _, match := range basecampMatches {
					if len(match) == 4 {
						// match[1] is account_id, match[2] is bucket_id, match[3] is todo_id
						accountId := match[1]
						bucketId := match[2]
						todoId := match[3]

						effectiveTodoId := todoId
						if _, exists := todoIdMap[todoId]; !exists {
							// Todo not found — it may have been moved. Resolve via API.
							effectiveTodoId = resolveTodoId(data.ApiClient.ApiClient, logger, resolvedIds, accountId, bucketId, todoId)
							if effectiveTodoId == "" {
								continue
							}
						}

						// Check if this todo exists in our database
						if domainId, exists := todoIdMap[effectiveTodoId]; exists {
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
									IssueKey:       effectiveTodoId,
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
