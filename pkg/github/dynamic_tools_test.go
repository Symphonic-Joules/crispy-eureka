package github

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/github-mcp-server/pkg/toolsets"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsetEnum(t *testing.T) {
	t.Run("returns enum with all toolset names", func(t *testing.T) {
		// Given a toolset group with multiple toolsets
		tsg := toolsets.NewToolsetGroup(false)
		ts1 := toolsets.NewToolset("toolset1", "Description 1")
		ts2 := toolsets.NewToolset("toolset2", "Description 2")
		tsg.Toolsets["toolset1"] = ts1
		tsg.Toolsets["toolset2"] = ts2

		// When we create an enum
		enum := ToolsetEnum(tsg)

		// Then it should be a valid property option
		assert.NotNil(t, enum)
	})

	t.Run("handles empty toolset group", func(t *testing.T) {
		// Given an empty toolset group
		tsg := toolsets.NewToolsetGroup(false)

		// When we create an enum
		enum := ToolsetEnum(tsg)

		// Then it should still be valid
		assert.NotNil(t, enum)
	})

	t.Run("includes all toolset names", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		tsg.Toolsets["repos"] = toolsets.NewToolset("repos", "Repo tools")
		tsg.Toolsets["issues"] = toolsets.NewToolset("issues", "Issue tools")
		tsg.Toolsets["prs"] = toolsets.NewToolset("prs", "PR tools")

		// When we create an enum
		enum := ToolsetEnum(tsg)

		// Then it should contain all names
		assert.NotNil(t, enum)
	})
}

func TestEnableToolset(t *testing.T) {
	t.Run("returns valid tool definition", func(t *testing.T) {
		// Given a toolset group and translator
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")

		// When we create the enable toolset tool
		tool, handler := EnableToolset(srv, tsg, translator)

		// Then it should have correct metadata
		assert.Equal(t, "enable_toolset", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, handler)
	})

	t.Run("tool has required toolset parameter", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")

		// When we create the tool
		tool, _ := EnableToolset(srv, tsg, translator)

		// Then it should have toolset parameter
		assert.Contains(t, tool.InputSchema.Properties, "toolset")
		assert.Contains(t, tool.InputSchema.Required, "toolset")
	})

	t.Run("tool is marked as read-only", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")

		// When we create the tool
		tool, _ := EnableToolset(srv, tsg, translator)

		// Then it should be marked read-only
		assert.NotNil(t, tool.Annotations)
		assert.True(t, *tool.Annotations.ReadOnlyHint)
	})

	t.Run("handler enables a disabled toolset", func(t *testing.T) {
		// Given a toolset group with a disabled toolset
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("test-toolset", "Test toolset")
		ts.Enabled = false
		tsg.Toolsets["test-toolset"] = ts
		
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")
		_, handler := EnableToolset(srv, tsg, translator)

		// When we call the handler to enable it
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "test-toolset",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then the toolset should be enabled
		require.NoError(t, err)
		assert.True(t, ts.Enabled)
		assert.Contains(t, getTextContent(result), "enabled")
	})

	t.Run("handler returns error for non-existent toolset", func(t *testing.T) {
		// Given a toolset group without the requested toolset
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")
		_, handler := EnableToolset(srv, tsg, translator)

		// When we request a non-existent toolset
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "non-existent",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return an error message
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, getTextContent(result), "not found")
	})

	t.Run("handler handles already enabled toolset", func(t *testing.T) {
		// Given an already enabled toolset
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("enabled-toolset", "Already enabled")
		ts.Enabled = true
		tsg.Toolsets["enabled-toolset"] = ts
		
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")
		_, handler := EnableToolset(srv, tsg, translator)

		// When we try to enable it again
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "enabled-toolset",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should indicate it's already enabled
		require.NoError(t, err)
		assert.Contains(t, getTextContent(result), "already enabled")
	})

	t.Run("handler returns error for missing toolset parameter", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		srv := server.NewMCPServer("test", "1.0.0")
		_, handler := EnableToolset(srv, tsg, translator)

		// When we call without toolset parameter
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return an error
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestListAvailableToolsets(t *testing.T) {
	t.Run("returns valid tool definition", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper

		// When we create the list tool
		tool, handler := ListAvailableToolsets(tsg, translator)

		// Then it should have correct metadata
		assert.Equal(t, "list_available_toolsets", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, handler)
	})

	t.Run("tool is marked as read-only", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper

		// When we create the tool
		tool, _ := ListAvailableToolsets(tsg, translator)

		// Then it should be marked read-only
		assert.NotNil(t, tool.Annotations)
		assert.True(t, *tool.Annotations.ReadOnlyHint)
	})

	t.Run("handler returns all toolsets with metadata", func(t *testing.T) {
		// Given a toolset group with multiple toolsets
		tsg := toolsets.NewToolsetGroup(false)
		ts1 := toolsets.NewToolset("repos", "Repository tools")
		ts1.Enabled = true
		ts2 := toolsets.NewToolset("issues", "Issue tools")
		ts2.Enabled = false
		tsg.Toolsets["repos"] = ts1
		tsg.Toolsets["issues"] = ts2

		translator := translations.NullTranslationHelper
		_, handler := ListAvailableToolsets(tsg, translator)

		// When we call the handler
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		// Then it should return JSON with all toolsets
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse the JSON response
		var toolsets []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &toolsets)
		require.NoError(t, err)

		assert.Len(t, toolsets, 2)
		
		// Check that both toolsets are present
		names := make([]string, len(toolsets))
		for i, ts := range toolsets {
			names[i] = ts["name"]
		}
		assert.Contains(t, names, "repos")
		assert.Contains(t, names, "issues")
	})

	t.Run("handler includes enabled status for each toolset", func(t *testing.T) {
		// Given toolsets with different enabled states
		tsg := toolsets.NewToolsetGroup(false)
		ts1 := toolsets.NewToolset("enabled-ts", "Enabled toolset")
		ts1.Enabled = true
		ts2 := toolsets.NewToolset("disabled-ts", "Disabled toolset")
		ts2.Enabled = false
		tsg.Toolsets["enabled-ts"] = ts1
		tsg.Toolsets["disabled-ts"] = ts2

		translator := translations.NullTranslationHelper
		_, handler := ListAvailableToolsets(tsg, translator)

		// When we call the handler
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		// Then each toolset should have enabled status
		require.NoError(t, err)
		
		var toolsets []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &toolsets)
		require.NoError(t, err)

		for _, ts := range toolsets {
			assert.Contains(t, ts, "currently_enabled")
			if ts["name"] == "enabled-ts" {
				assert.Equal(t, "true", ts["currently_enabled"])
			} else if ts["name"] == "disabled-ts" {
				assert.Equal(t, "false", ts["currently_enabled"])
			}
		}
	})

	t.Run("handler includes description for each toolset", func(t *testing.T) {
		// Given toolsets with descriptions
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("test-ts", "This is a test toolset")
		tsg.Toolsets["test-ts"] = ts

		translator := translations.NullTranslationHelper
		_, handler := ListAvailableToolsets(tsg, translator)

		// When we call the handler
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		// Then descriptions should be included
		require.NoError(t, err)
		
		var toolsets []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &toolsets)
		require.NoError(t, err)

		assert.Len(t, toolsets, 1)
		assert.Equal(t, "This is a test toolset", toolsets[0]["description"])
	})

	t.Run("handler returns empty array for empty toolset group", func(t *testing.T) {
		// Given an empty toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		_, handler := ListAvailableToolsets(tsg, translator)

		// When we call the handler
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		// Then it should return an empty array
		require.NoError(t, err)
		
		var toolsets []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &toolsets)
		require.NoError(t, err)

		assert.Empty(t, toolsets)
	})
}

func TestGetToolsetsTools(t *testing.T) {
	t.Run("returns valid tool definition", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper

		// When we create the get tools tool
		tool, handler := GetToolsetsTools(tsg, translator)

		// Then it should have correct metadata
		assert.Equal(t, "get_toolset_tools", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, handler)
	})

	t.Run("tool has required toolset parameter", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper

		// When we create the tool
		tool, _ := GetToolsetsTools(tsg, translator)

		// Then it should have toolset parameter
		assert.Contains(t, tool.InputSchema.Properties, "toolset")
		assert.Contains(t, tool.InputSchema.Required, "toolset")
	})

	t.Run("tool is marked as read-only", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper

		// When we create the tool
		tool, _ := GetToolsetsTools(tsg, translator)

		// Then it should be marked read-only
		assert.NotNil(t, tool.Annotations)
		assert.True(t, *tool.Annotations.ReadOnlyHint)
	})

	t.Run("handler returns tools for specified toolset", func(t *testing.T) {
		// Given a toolset with tools
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("test-toolset", "Test")
		
		mockTool := mcp.NewTool("mock_tool", mcp.WithDescription("A mock tool"))
		st := toolsets.NewServerTool(mockTool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("result"), nil
		})
		ts.AddReadTools(st)
		tsg.Toolsets["test-toolset"] = ts

		translator := translations.NullTranslationHelper
		_, handler := GetToolsetsTools(tsg, translator)

		// When we request tools for this toolset
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "test-toolset",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return the tools
		require.NoError(t, err)
		assert.False(t, result.IsError)
		
		var tools []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &tools)
		require.NoError(t, err)

		assert.NotEmpty(t, tools)
		assert.Equal(t, "mock_tool", tools[0]["name"])
	})

	t.Run("handler returns error for non-existent toolset", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		_, handler := GetToolsetsTools(tsg, translator)

		// When we request a non-existent toolset
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "non-existent",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return an error
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, getTextContent(result), "not found")
	})

	t.Run("handler returns empty array for toolset with no tools", func(t *testing.T) {
		// Given a toolset with no tools
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("empty-toolset", "Empty")
		tsg.Toolsets["empty-toolset"] = ts

		translator := translations.NullTranslationHelper
		_, handler := GetToolsetsTools(tsg, translator)

		// When we request its tools
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "empty-toolset",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return an empty array
		require.NoError(t, err)
		
		var tools []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &tools)
		require.NoError(t, err)

		assert.Empty(t, tools)
	})

	t.Run("handler includes tool descriptions", func(t *testing.T) {
		// Given a toolset with a tool that has a description
		tsg := toolsets.NewToolsetGroup(false)
		ts := toolsets.NewToolset("test-toolset", "Test")
		
		mockTool := mcp.NewTool("described_tool", 
			mcp.WithDescription("This tool has a description"))
		st := toolsets.NewServerTool(mockTool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("result"), nil
		})
		ts.AddReadTools(st)
		tsg.Toolsets["test-toolset"] = ts

		translator := translations.NullTranslationHelper
		_, handler := GetToolsetsTools(tsg, translator)

		// When we request tools
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"toolset": "test-toolset",
				},
			},
		}

		result, err := handler(context.Background(), request)

		// Then descriptions should be included
		require.NoError(t, err)
		
		var tools []map[string]string
		err = json.Unmarshal([]byte(getTextContent(result)), &tools)
		require.NoError(t, err)

		assert.NotEmpty(t, tools)
		assert.Equal(t, "This tool has a description", tools[0]["description"])
	})

	t.Run("handler returns error for missing toolset parameter", func(t *testing.T) {
		// Given a toolset group
		tsg := toolsets.NewToolsetGroup(false)
		translator := translations.NullTranslationHelper
		_, handler := GetToolsetsTools(tsg, translator)

		// When we call without toolset parameter
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{},
			},
		}

		result, err := handler(context.Background(), request)

		// Then it should return an error
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

// Helper function to extract text content from CallToolResult
func getTextContent(result *mcp.CallToolResult) string {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	return ""
}