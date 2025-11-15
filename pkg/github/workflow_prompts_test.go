package github

import (
	"context"
	"testing"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueToFixWorkflowPrompt(t *testing.T) {
	t.Run("returns valid prompt definition", func(t *testing.T) {
		// Given a translation helper
		translator := translations.NullTranslationHelper

		// When we create the workflow prompt
		tool, handler := IssueToFixWorkflowPrompt(translator)

		// Then the prompt should have correct metadata
		assert.Equal(t, "IssueToFixWorkflow", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, handler)
	})

	t.Run("prompt has required arguments", func(t *testing.T) {
		// Given a translation helper
		translator := translations.NullTranslationHelper

		// When we create the workflow prompt
		tool, _ := IssueToFixWorkflowPrompt(translator)

		// Then it should have the required arguments
		assert.Contains(t, tool.Input.Properties, "owner")
		assert.Contains(t, tool.Input.Properties, "repo")
		assert.Contains(t, tool.Input.Properties, "title")
		assert.Contains(t, tool.Input.Properties, "description")

		// And required arguments should be marked as such
		assert.Contains(t, tool.Input.Required, "owner")
		assert.Contains(t, tool.Input.Required, "repo")
		assert.Contains(t, tool.Input.Required, "title")
		assert.Contains(t, tool.Input.Required, "description")
	})

	t.Run("prompt has optional arguments", func(t *testing.T) {
		// Given a translation helper
		translator := translations.NullTranslationHelper

		// When we create the workflow prompt
		tool, _ := IssueToFixWorkflowPrompt(translator)

		// Then it should have optional arguments
		assert.Contains(t, tool.Input.Properties, "labels")
		assert.Contains(t, tool.Input.Properties, "assignees")

		// And they should not be in the required list
		assert.NotContains(t, tool.Input.Required, "labels")
		assert.NotContains(t, tool.Input.Required, "assignees")
	})

	t.Run("handler returns messages with required arguments only", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with only required arguments
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "testowner",
					"repo":        "testrepo",
					"title":       "Test Issue",
					"description": "Test description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should succeed and return messages
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
	})

	t.Run("handler returns correct number of messages", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "testowner",
					"repo":        "testrepo",
					"title":       "Test Issue",
					"description": "Test description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return multiple messages (conversation flow)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Messages), 4) // At least 4 messages in the workflow
	})

	t.Run("handler includes owner and repo in messages", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with specific owner/repo
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "myowner",
					"repo":        "myrepo",
					"title":       "Bug Fix",
					"description": "Fix the bug",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the messages should mention the owner and repo
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}
		assert.Contains(t, allContent, "myowner/myrepo")
	})

	t.Run("handler includes title in messages", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with a specific title
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "Unique Title 12345",
					"description": "description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the messages should mention the title
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}
		assert.Contains(t, allContent, "Unique Title 12345")
	})

	t.Run("handler includes description in messages", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with a specific description
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "This is a unique description xyz789",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the messages should mention the description
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}
		assert.Contains(t, allContent, "This is a unique description xyz789")
	})

	t.Run("handler includes labels when provided", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with labels
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
					"labels":      "bug,urgent,frontend",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the messages should mention the labels
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}
		assert.Contains(t, allContent, "bug,urgent,frontend")
	})

	t.Run("handler includes assignees when provided", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with assignees
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
					"assignees":   "user1,user2",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the messages should mention the assignees
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}
		assert.Contains(t, allContent, "user1,user2")
	})

	t.Run("handler works without optional arguments", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it without labels or assignees
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should still work correctly
		require.NoError(t, err)
		assert.NotEmpty(t, result.Messages)
	})

	t.Run("messages have correct roles", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then messages should have alternating user/assistant roles
		require.NoError(t, err)
		for _, msg := range result.Messages {
			assert.Contains(t, []string{"user", "assistant"}, msg.Role)
		}
	})

	t.Run("messages are properly structured conversation", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then messages should form a coherent conversation
		require.NoError(t, err)
		assert.Greater(t, len(result.Messages), 0)

		// First message should set context
		firstMsg := result.Messages[0]
		assert.Equal(t, "user", firstMsg.Role)
		if textContent, ok := firstMsg.Content.(mcp.TextContent); ok {
			assert.Contains(t, textContent.Text, "workflow")
		}
	})

	t.Run("handler uses custom translation function", func(t *testing.T) {
		// Given a custom translator that prefixes all strings
		customTranslator := func(key string, defaultValue string) string {
			return "CUSTOM_" + defaultValue
		}

		// When we create the prompt with custom translator
		tool, _ := IssueToFixWorkflowPrompt(customTranslator)

		// Then the description should use the custom translator
		assert.Contains(t, tool.Description, "CUSTOM_")
	})

	t.Run("handler handles non-string optional parameters", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it with non-string optional params (edge case)
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
					"labels":      123, // non-string
					"assignees":   true, // non-string
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should handle gracefully by converting to string
		require.NoError(t, err)
		assert.NotEmpty(t, result.Messages)
	})

	t.Run("messages provide actionable workflow steps", func(t *testing.T) {
		// Given a prompt handler
		translator := translations.NullTranslationHelper
		_, handler := IssueToFixWorkflowPrompt(translator)

		// When we invoke it
		request := mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Arguments: map[string]interface{}{
					"owner":       "owner",
					"repo":        "repo",
					"title":       "title",
					"description": "description",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then messages should contain workflow steps
		require.NoError(t, err)
		var allContent string
		for _, msg := range result.Messages {
			if textContent, ok := msg.Content.(mcp.TextContent); ok {
				allContent += textContent.Text
			}
		}

		// Should mention key workflow steps
		assert.Contains(t, allContent, "issue")
		assert.Contains(t, allContent, "Copilot")
	})
}