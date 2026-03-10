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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreLog "github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
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
	EnabledByDefault: false,
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

// parseTime parses an ISO 8601 / RFC 3339 timestamp string into *time.Time.
// Returns nil for empty strings or parse errors.
func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// resolveTodo resolves a possibly-moved todo by calling the Basecamp API.
// If the todo was moved, the API redirects and returns JSON with the current todo.
// Results are cached in resolvedTodos to avoid redundant API calls.
// A nil value in the cache means a previous lookup failed.
func resolveTodo(
	apiClient *api.ApiClient,
	logger coreLog.Logger,
	resolvedTodos map[string]*BasecampApiTodo,
	accountId, bucketId, todoId string,
) *BasecampApiTodo {
	if apiTodo, ok := resolvedTodos[todoId]; ok {
		return apiTodo
	}
	path := fmt.Sprintf("%s/buckets/%s/todos/%s.json", accountId, bucketId, todoId)
	resp, err := apiClient.Get(path, nil, nil)
	if err != nil {
		logger.Warn(err, "failed to resolve todo %s via API", todoId)
		resolvedTodos[todoId] = nil
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		logger.Warn(nil, "API returned %d resolving todo %s", resp.StatusCode, todoId)
		resp.Body.Close()
		resolvedTodos[todoId] = nil
		return nil
	}
	var apiTodo BasecampApiTodo
	if unmarshalErr := api.UnmarshalResponse(resp, &apiTodo); unmarshalErr != nil {
		logger.Warn(unmarshalErr, "failed to unmarshal API response for todo %s", todoId)
		resolvedTodos[todoId] = nil
		return nil
	}
	resolvedTodos[todoId] = &apiTodo
	return &apiTodo
}

func LinkPrToTodo(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BasecampTaskData)
	logger := taskCtx.GetLogger()

	// Create ID generators
	todoIdGen := didgen.NewDomainIdGenerator(&models.BasecampTodo{})
	projectIdGen := didgen.NewDomainIdGenerator(&models.BasecampProject{})

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

	// Cache for API-resolved todos (for moved or uncollected todos)
	resolvedTodos := make(map[string]*BasecampApiTodo)

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
						var apiTodo *BasecampApiTodo
						if _, exists := todoIdMap[todoId]; !exists {
							// Todo not found — it may have been moved or not collected. Resolve via API.
							apiTodo = resolveTodo(data.ApiClient.ApiClient, logger, resolvedTodos, accountId, bucketId, todoId)
							if apiTodo == nil {
								continue
							}
							effectiveTodoId = strconv.FormatInt(apiTodo.ID, 10)
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
						} else if apiTodo != nil {
							// Todo not in local DB — create domain records from API response
							domainId := todoIdGen.Generate(data.Options.ConnectionId, effectiveTodoId)
							projectId := strconv.FormatInt(apiTodo.Bucket.ID, 10)
							boardId := projectIdGen.Generate(data.Options.ConnectionId, projectId)

							// Map status
							status := ticket.TODO
							if apiTodo.Completed {
								status = ticket.DONE
							}

							// Get assignee info (first assignee if any)
							var assigneeId, assigneeName string
							if len(apiTodo.Assignees) > 0 {
								assigneeId = strconv.FormatInt(apiTodo.Assignees[0].ID, 10)
								assigneeName = apiTodo.Assignees[0].Name
							}

							board := ticket.NewBoard(boardId, apiTodo.Bucket.Name)

							issue := &ticket.Issue{
								DomainEntity: domainlayer.DomainEntity{
									Id: domainId,
								},
								Url:             apiTodo.AppUrl,
								IssueKey:        effectiveTodoId,
								Title:           apiTodo.Title,
								Status:          status,
								OriginalStatus:  apiTodo.Status,
								Type:            ticket.TASK,
								OriginalType:    "todo",
								CreatorId:       strconv.FormatInt(apiTodo.Creator.ID, 10),
								CreatorName:     apiTodo.Creator.Name,
								AssigneeId:      assigneeId,
								AssigneeName:    assigneeName,
								OriginalProject: apiTodo.Bucket.Name,
								CreatedDate:     parseTime(apiTodo.CreatedAt),
								UpdatedDate:     parseTime(apiTodo.UpdatedAt),
								ResolutionDate:  parseTime(apiTodo.CompletedAt),
							}

							boardIssue := &ticket.BoardIssue{
								BoardId: boardId,
								IssueId: domainId,
							}

							prIssue := &crossdomain.PullRequestIssue{
								PullRequestId:  pr.Id,
								IssueId:        domainId,
								PullRequestKey: pr.PullRequestKey,
								IssueKey:       effectiveTodoId,
							}

							results = append(results, board, issue, boardIssue, prIssue)

							// Cache for subsequent PRs referencing the same todo
							todoIdMap[effectiveTodoId] = domainId

							logger.Info("Created domain records for uncollected todo %s (project %s %q)", effectiveTodoId, projectId, apiTodo.Bucket.Name)
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
