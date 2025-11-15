package githubv4mock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryMatcher(t *testing.T) {
	t.Run("creates matcher with string query", func(t *testing.T) {
		// Given a string query
		query := "query { viewer { login } }"
		variables := map[string]any{"var1": "value1"}
		response := DataResponse(map[string]any{"viewer": map[string]any{"login": "test"}})

		// When we create a matcher
		matcher := NewQueryMatcher(query, variables, response)

		// Then it should be properly initialized
		assert.Equal(t, query, matcher.Request)
		assert.Equal(t, variables, matcher.Variables)
		assert.Equal(t, response, matcher.Response)
	})

	t.Run("creates matcher with struct query", func(t *testing.T) {
		// Given a struct query
		type Query struct {
			Viewer struct {
				Login string
			}
		}
		query := Query{}
		variables := map[string]any{}
		response := DataResponse(map[string]any{"viewer": map[string]any{"login": "test"}})

		// When we create a matcher
		matcher := NewQueryMatcher(query, variables, response)

		// Then it should construct the query string
		assert.NotEmpty(t, matcher.Request)
		assert.Contains(t, matcher.Request, "viewer")
		assert.Equal(t, variables, matcher.Variables)
	})

	t.Run("handles nil variables", func(t *testing.T) {
		// Given a query with nil variables
		query := "query { test }"
		response := DataResponse(map[string]any{})

		// When we create a matcher
		matcher := NewQueryMatcher(query, nil, response)

		// Then it should handle gracefully
		assert.NotNil(t, matcher)
		assert.Nil(t, matcher.Variables)
	})

	t.Run("handles empty variables", func(t *testing.T) {
		// Given a query with empty variables
		query := "query { test }"
		variables := map[string]any{}
		response := DataResponse(map[string]any{})

		// When we create a matcher
		matcher := NewQueryMatcher(query, variables, response)

		// Then it should preserve empty map
		assert.NotNil(t, matcher.Variables)
		assert.Empty(t, matcher.Variables)
	})
}

func TestNewMutationMatcher(t *testing.T) {
	t.Run("creates matcher with string mutation", func(t *testing.T) {
		// Given a string mutation
		mutation := "mutation { createIssue(input: $input) { issue { id } } }"
		input := map[string]any{"title": "Test"}
		variables := map[string]any{"var1": "value1"}
		response := DataResponse(map[string]any{"createIssue": map[string]any{"issue": map[string]any{"id": "123"}}})

		// When we create a matcher
		matcher := NewMutationMatcher(mutation, input, variables, response)

		// Then it should be properly initialized
		assert.Equal(t, mutation, matcher.Request)
		assert.NotNil(t, matcher.Variables)
	})

	t.Run("creates matcher with struct mutation", func(t *testing.T) {
		// Given a struct mutation
		type Mutation struct {
			CreateIssue struct {
				Issue struct {
					ID string
				}
			} `graphql:"createIssue(input: $input)"`
		}
		mutation := Mutation{}
		input := map[string]any{"title": "Test"}
		response := DataResponse(map[string]any{})

		// When we create a matcher
		matcher := NewMutationMatcher(mutation, input, nil, response)

		// Then it should construct the mutation string
		assert.NotEmpty(t, matcher.Request)
		assert.Contains(t, matcher.Request, "mutation")
		assert.Contains(t, matcher.Variables, "input")
	})

	t.Run("adds input to variables when not present", func(t *testing.T) {
		// Given a mutation without input in variables
		mutation := "mutation { test }"
		input := map[string]any{"field": "value"}
		response := DataResponse(map[string]any{})

		// When we create a matcher with nil variables
		matcher := NewMutationMatcher(mutation, input, nil, response)

		// Then input should be added to variables
		assert.NotNil(t, matcher.Variables)
		// Note: for string mutations, variables are used as-is
	})

	t.Run("handles struct input conversion", func(t *testing.T) {
		// Given a struct input
		type Input struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		input := Input{Title: "Test", Body: "Body"}
		
		type Mutation struct {
			CreateIssue struct {
				Issue struct {
					ID string
				}
			} `graphql:"createIssue(input: $input)"`
		}
		mutation := Mutation{}
		response := DataResponse(map[string]any{})

		// When we create a matcher
		matcher := NewMutationMatcher(mutation, input, nil, response)

		// Then input should be converted to map
		assert.NotNil(t, matcher.Variables)
		assert.Contains(t, matcher.Variables, "input")
		inputMap, ok := matcher.Variables["input"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "Test", inputMap["title"])
		assert.Equal(t, "Body", inputMap["body"])
	})
}

func TestDataResponse(t *testing.T) {
	t.Run("creates response with data", func(t *testing.T) {
		// Given data map
		data := map[string]any{
			"viewer": map[string]any{
				"login": "testuser",
			},
		}

		// When we create a data response
		response := DataResponse(data)

		// Then it should have data and no errors
		assert.Equal(t, data, response.Data)
		assert.Nil(t, response.Errors)
	})

	t.Run("handles empty data", func(t *testing.T) {
		// Given empty data
		data := map[string]any{}

		// When we create a data response
		response := DataResponse(data)

		// Then it should create valid response
		assert.NotNil(t, response.Data)
		assert.Empty(t, response.Data)
	})

	t.Run("handles nil data", func(t *testing.T) {
		// When we create a response with nil data
		response := DataResponse(nil)

		// Then it should handle gracefully
		assert.Nil(t, response.Data)
	})
}

func TestErrorResponse(t *testing.T) {
	t.Run("creates response with error message", func(t *testing.T) {
		// Given an error message
		errorMsg := "Resource not found"

		// When we create an error response
		response := ErrorResponse(errorMsg)

		// Then it should have error and no data
		assert.NotNil(t, response.Errors)
		assert.Len(t, response.Errors, 1)
		assert.Equal(t, errorMsg, response.Errors[0].Message)
		assert.Nil(t, response.Data)
	})

	t.Run("handles empty error message", func(t *testing.T) {
		// Given an empty error message
		errorMsg := ""

		// When we create an error response
		response := ErrorResponse(errorMsg)

		// Then it should still create valid response
		assert.NotNil(t, response.Errors)
		assert.Len(t, response.Errors, 1)
		assert.Equal(t, "", response.Errors[0].Message)
	})

	t.Run("handles special characters in error message", func(t *testing.T) {
		// Given error message with special characters
		errorMsg := "Error: \"quoted\" and\nnewline"

		// When we create an error response
		response := ErrorResponse(errorMsg)

		// Then it should preserve the message
		assert.Equal(t, errorMsg, response.Errors[0].Message)
	})
}

func TestGithubv4InputStructToMap(t *testing.T) {
	t.Run("converts simple struct to map", func(t *testing.T) {
		// Given a simple struct
		type Input struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		input := Input{Title: "Test Title", Body: "Test Body"}

		// When we convert it to map
		result, err := githubv4InputStructToMap(input)

		// Then it should create correct map
		require.NoError(t, err)
		assert.Equal(t, "Test Title", result["title"])
		assert.Equal(t, "Test Body", result["body"])
	})

	t.Run("converts nested struct to map", func(t *testing.T) {
		// Given a nested struct
		type Nested struct {
			Value string `json:"value"`
		}
		type Input struct {
			Title  string `json:"title"`
			Nested Nested `json:"nested"`
		}
		input := Input{
			Title:  "Test",
			Nested: Nested{Value: "nested value"},
		}

		// When we convert it to map
		result, err := githubv4InputStructToMap(input)

		// Then it should handle nested structure
		require.NoError(t, err)
		assert.Equal(t, "Test", result["title"])
		nested, ok := result["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "nested value", nested["value"])
	})

	t.Run("respects json tags", func(t *testing.T) {
		// Given struct with json tags
		type Input struct {
			FieldName string `json:"field_name"`
			OtherName string `json:"other_name,omitempty"`
		}
		input := Input{FieldName: "value1", OtherName: "value2"}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then json tags should be used as keys
		require.NoError(t, err)
		assert.Equal(t, "value1", result["field_name"])
		assert.Equal(t, "value2", result["other_name"])
	})

	t.Run("handles omitempty tag", func(t *testing.T) {
		// Given struct with omitempty and empty value
		type Input struct {
			Required string `json:"required"`
			Optional string `json:"optional,omitempty"`
		}
		input := Input{Required: "value"}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then empty optional field should be omitted
		require.NoError(t, err)
		assert.Equal(t, "value", result["required"])
		assert.NotContains(t, result, "optional")
	})

	t.Run("handles pointer fields", func(t *testing.T) {
		// Given struct with pointer field
		type Input struct {
			Title *string `json:"title"`
			Body  *string `json:"body,omitempty"`
		}
		title := "Test Title"
		input := Input{Title: &title}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then pointers should be handled correctly
		require.NoError(t, err)
		assert.Equal(t, "Test Title", result["title"])
	})

	t.Run("handles nil pointer fields", func(t *testing.T) {
		// Given struct with nil pointer
		type Input struct {
			Title *string `json:"title,omitempty"`
		}
		input := Input{Title: nil}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then nil pointer should be omitted
		require.NoError(t, err)
		assert.NotContains(t, result, "title")
	})

	t.Run("handles arrays and slices", func(t *testing.T) {
		// Given struct with slice
		type Input struct {
			Tags []string `json:"tags"`
		}
		input := Input{Tags: []string{"tag1", "tag2"}}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then slice should be preserved
		require.NoError(t, err)
		tags, ok := result["tags"].([]any)
		require.True(t, ok)
		assert.Len(t, tags, 2)
	})

	t.Run("handles githubv4 types", func(t *testing.T) {
		// Given struct with githubv4 types
		type Input struct {
			ID    githubv4.ID     `json:"id"`
			Value githubv4.String `json:"value"`
		}
		input := Input{
			ID:    githubv4.ID("test-id"),
			Value: githubv4.String("test-value"),
		}

		// When we convert it
		result, err := githubv4InputStructToMap(input)

		// Then types should be converted
		require.NoError(t, err)
		assert.Equal(t, "test-id", result["id"])
		assert.Equal(t, "test-value", result["value"])
	})
}

func TestNewMockedHTTPClient(t *testing.T) {
	t.Run("creates HTTP client with matchers", func(t *testing.T) {
		// Given matchers
		matchers := []Matcher{
			NewQueryMatcher(
				"query { viewer { login } }",
				nil,
				DataResponse(map[string]any{"viewer": map[string]any{"login": "test"}}),
			),
		}

		// When we create a mocked client
		client := NewMockedHTTPClient(matchers...)

		// Then it should be valid
		assert.NotNil(t, client)
		assert.NotNil(t, client.Transport)
	})

	t.Run("client handles matching request", func(t *testing.T) {
		// Given a matcher
		query := "query{viewer{login}}"
		matchers := []Matcher{
			NewQueryMatcher(
				query,
				nil,
				DataResponse(map[string]any{"viewer": map[string]any{"login": "testuser"}}),
			),
		}
		client := NewMockedHTTPClient(matchers...)

		// When we make a matching request
		reqBody := map[string]any{
			"query":     query,
			"variables": nil,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "http://example.com/graphql", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Transport.RoundTrip(req)

		// Then it should return the mocked response
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result GQLResponse
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "testuser", result.Data["viewer"].(map[string]any)["login"])
	})

	t.Run("client returns error for non-matching request", func(t *testing.T) {
		// Given a matcher that won't match
		matchers := []Matcher{
			NewQueryMatcher(
				"query { viewer { login } }",
				nil,
				DataResponse(map[string]any{}),
			),
		}
		client := NewMockedHTTPClient(matchers...)

		// When we make a non-matching request
		reqBody := map[string]any{
			"query":     "query { different { query } }",
			"variables": nil,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "http://example.com/graphql", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Transport.RoundTrip(req)

		// Then it should return an error response
		require.NoError(t, err)
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("handles empty matchers list", func(t *testing.T) {
		// When we create client with no matchers
		client := NewMockedHTTPClient()

		// Then it should still be valid
		assert.NotNil(t, client)
	})
}

func TestMatcherIntegration(t *testing.T) {
	t.Run("full query workflow", func(t *testing.T) {
		// Given a complete query setup
		type Query struct {
			Repository struct {
				Name string
			} `graphql:"repository(owner: $owner, name: $name)"`
		}
		
		query := Query{}
		variables := map[string]any{
			"owner": githubv4.String("testowner"),
			"name":  githubv4.String("testrepo"),
		}
		response := DataResponse(map[string]any{
			"repository": map[string]any{
				"name": "testrepo",
			},
		})

		// When we create a matcher
		matcher := NewQueryMatcher(query, variables, response)

		// Then it should be properly configured
		assert.NotEmpty(t, matcher.Request)
		assert.NotNil(t, matcher.Variables)
		assert.NotNil(t, matcher.Response.Data)
	})

	t.Run("full mutation workflow", func(t *testing.T) {
		// Given a complete mutation setup
		type Mutation struct {
			CreateIssue struct {
				Issue struct {
					ID string
				}
			} `graphql:"createIssue(input: $input)"`
		}
		
		type Input struct {
			RepositoryID githubv4.ID     `json:"repositoryId"`
			Title        githubv4.String `json:"title"`
		}
		
		mutation := Mutation{}
		input := Input{
			RepositoryID: githubv4.ID("repo123"),
			Title:        githubv4.String("Test Issue"),
		}
		response := DataResponse(map[string]any{
			"createIssue": map[string]any{
				"issue": map[string]any{
					"id": "issue123",
				},
			},
		})

		// When we create a matcher
		matcher := NewMutationMatcher(mutation, input, nil, response)

		// Then it should be properly configured
		assert.NotEmpty(t, matcher.Request)
		assert.Contains(t, matcher.Request, "mutation")
		assert.NotNil(t, matcher.Variables)
		assert.Contains(t, matcher.Variables, "input")
	})
}