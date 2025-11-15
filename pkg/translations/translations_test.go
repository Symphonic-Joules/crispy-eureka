package translations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullTranslationHelper(t *testing.T) {
	t.Run("always returns default value", func(t *testing.T) {
		// Given a null translation helper
		helper := NullTranslationHelper

		// When we request any key with a default value
		result := helper("SOME_KEY", "default value")

		// Then the default value should be returned
		assert.Equal(t, "default value", result)
	})

	t.Run("ignores key parameter", func(t *testing.T) {
		// Given a null translation helper
		helper := NullTranslationHelper

		// When we use different keys with same default
		result1 := helper("KEY1", "same default")
		result2 := helper("KEY2", "same default")

		// Then both should return the default
		assert.Equal(t, "same default", result1)
		assert.Equal(t, "same default", result2)
	})

	t.Run("handles empty strings", func(t *testing.T) {
		// Given a null translation helper
		helper := NullTranslationHelper

		// When we use empty key and default
		result := helper("", "")

		// Then empty string should be returned
		assert.Equal(t, "", result)
	})
}

func TestTranslationHelper(t *testing.T) {
	t.Run("returns default value when no override exists", func(t *testing.T) {
		// Given a translation helper with no config file
		helper, cleanup := TranslationHelper()
		defer cleanup()

		// When we request a key with a default value
		result := helper("TEST_KEY", "default value")

		// Then the default value should be returned
		assert.Equal(t, "default value", result)
	})

	t.Run("converts key to uppercase", func(t *testing.T) {
		// Given a translation helper
		helper, cleanup := TranslationHelper()
		defer cleanup()

		// When we use a lowercase key
		result := helper("test_key", "default")

		// Then it should work the same as uppercase
		assert.Equal(t, "default", result)
	})

	t.Run("caches values after first call", func(t *testing.T) {
		// Given a translation helper
		helper, cleanup := TranslationHelper()
		defer cleanup()

		// When we call the helper twice with the same key
		result1 := helper("CACHE_TEST", "first")
		result2 := helper("CACHE_TEST", "second")

		// Then both calls should return the first default value (cached)
		assert.Equal(t, "first", result1)
		assert.Equal(t, "first", result2)
	})

	t.Run("reads from environment variable", func(t *testing.T) {
		// Given an environment variable with a translation override
		key := "ENV_TEST_KEY"
		expectedValue := "env override value"
		envVar := "GITHUB_MCP_" + key
		err := os.Setenv(envVar, expectedValue)
		require.NoError(t, err)
		defer os.Unsetenv(envVar)

		// When we create a new helper and request the key
		helper, cleanup := TranslationHelper()
		defer cleanup()
		result := helper(key, "default value")

		// Then the environment variable value should be returned
		assert.Equal(t, expectedValue, result)
	})

	t.Run("environment variable takes precedence over default", func(t *testing.T) {
		// Given an environment variable override
		key := "PRECEDENCE_TEST"
		envValue := "from environment"
		envVar := "GITHUB_MCP_" + key
		err := os.Setenv(envVar, envValue)
		require.NoError(t, err)
		defer os.Unsetenv(envVar)

		// When we create a helper and request the key
		helper, cleanup := TranslationHelper()
		defer cleanup()
		result := helper(key, "default value")

		// Then the environment value should take precedence
		assert.Equal(t, envValue, result)
	})

	t.Run("handles multiple keys independently", func(t *testing.T) {
		// Given a translation helper
		helper, cleanup := TranslationHelper()
		defer cleanup()

		// When we request multiple different keys
		result1 := helper("KEY1", "default1")
		result2 := helper("KEY2", "default2")
		result3 := helper("KEY3", "default3")

		// Then each should return its own default value
		assert.Equal(t, "default1", result1)
		assert.Equal(t, "default2", result2)
		assert.Equal(t, "default3", result3)
	})

	t.Run("cleanup dumps translation map to file", func(t *testing.T) {
		// Given a translation helper that has been used
		helper, cleanup := TranslationHelper()
		helper("TEST_DUMP_KEY", "test value")

		// When we call cleanup
		cleanup()

		// Then a config file should be created
		_, err := os.Stat("github-mcp-server-config.json")
		assert.NoError(t, err)

		// Cleanup the created file
		defer os.Remove("github-mcp-server-config.json")

		// And it should contain our key
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var config map[string]string
		err = json.Unmarshal(data, &config)
		require.NoError(t, err)

		assert.Contains(t, config, "TEST_DUMP_KEY")
		assert.Equal(t, "test value", config["TEST_DUMP_KEY"])
	})

	t.Run("reads from config file if present", func(t *testing.T) {
		// Given a config file with translation overrides
		configData := map[string]string{
			"FILE_KEY": "value from file",
		}
		data, err := json.MarshalIndent(configData, "", "  ")
		require.NoError(t, err)

		configFile := "github-mcp-server-config.json"
		err = os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)
		defer os.Remove(configFile)

		// When we create a new helper
		helper, cleanup := TranslationHelper()
		defer cleanup()

		// Then it should read from the config file
		result := helper("FILE_KEY", "default")
		assert.Equal(t, "value from file", result)
	})
}

func TestDumpTranslationKeyMap(t *testing.T) {
	t.Run("creates file with valid JSON", func(t *testing.T) {
		// Given a translation map
		translationMap := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		// When we dump it to a file
		err := DumpTranslationKeyMap(translationMap)
		require.NoError(t, err)
		defer os.Remove("github-mcp-server-config.json")

		// Then the file should exist
		_, err = os.Stat("github-mcp-server-config.json")
		require.NoError(t, err)

		// And contain valid JSON with our data
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, translationMap, result)
	})

	t.Run("handles empty map", func(t *testing.T) {
		// Given an empty translation map
		translationMap := map[string]string{}

		// When we dump it
		err := DumpTranslationKeyMap(translationMap)
		require.NoError(t, err)
		defer os.Remove("github-mcp-server-config.json")

		// Then file should exist with empty JSON object
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Empty(t, result)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		// Given an existing config file with old data
		oldData := map[string]string{"OLD_KEY": "old value"}
		oldJSON, _ := json.Marshal(oldData)
		err := os.WriteFile("github-mcp-server-config.json", oldJSON, 0644)
		require.NoError(t, err)

		// When we dump a new translation map
		newMap := map[string]string{"NEW_KEY": "new value"}
		err = DumpTranslationKeyMap(newMap)
		require.NoError(t, err)
		defer os.Remove("github-mcp-server-config.json")

		// Then the file should contain only the new data
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, newMap, result)
		assert.NotContains(t, result, "OLD_KEY")
	})

	t.Run("creates properly formatted JSON", func(t *testing.T) {
		// Given a translation map
		translationMap := map[string]string{
			"KEY1": "value1",
		}

		// When we dump it
		err := DumpTranslationKeyMap(translationMap)
		require.NoError(t, err)
		defer os.Remove("github-mcp-server-config.json")

		// Then the JSON should be indented
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		// Verify it's formatted (contains newlines)
		assert.Contains(t, string(data), "\n")
	})

	t.Run("handles special characters in values", func(t *testing.T) {
		// Given a map with special characters
		translationMap := map[string]string{
			"SPECIAL": "value with \"quotes\" and\nnewlines\tand\ttabs",
		}

		// When we dump and read it back
		err := DumpTranslationKeyMap(translationMap)
		require.NoError(t, err)
		defer os.Remove("github-mcp-server-config.json")

		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Then special characters should be preserved
		assert.Equal(t, translationMap["SPECIAL"], result["SPECIAL"])
	})

	t.Run("fails gracefully with invalid directory", func(t *testing.T) {
		// Given we're in a directory we can't write to
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		// Create a temporary directory and remove write permissions
		tmpDir := filepath.Join(os.TempDir(), "readonly-test")
		err := os.MkdirAll(tmpDir, 0755)
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		err = os.Chmod(tmpDir, 0444)
		require.NoError(t, err)
		defer os.Chmod(tmpDir, 0755)

		// When we try to dump a translation map
		translationMap := map[string]string{"KEY": "value"}
		err = DumpTranslationKeyMap(translationMap)

		// Then it should return an error
		assert.Error(t, err)
	})
}

func TestTranslationHelperIntegration(t *testing.T) {
	t.Run("full workflow with env vars and config file", func(t *testing.T) {
		// Setup: Clean environment
		defer os.Remove("github-mcp-server-config.json")

		// Given environment variables and config file
		os.Setenv("GITHUB_MCP_ENV_KEY", "from env")
		defer os.Unsetenv("GITHUB_MCP_ENV_KEY")

		configData := map[string]string{
			"FILE_KEY": "from file",
		}
		data, _ := json.MarshalIndent(configData, "", "  ")
		os.WriteFile("github-mcp-server-config.json", data, 0644)

		// When we create a helper and use multiple keys
		helper, cleanup := TranslationHelper()
		envResult := helper("ENV_KEY", "env default")
		fileResult := helper("FILE_KEY", "file default")
		defaultResult := helper("DEFAULT_KEY", "just default")

		// Then each should use the appropriate source
		assert.Equal(t, "from env", envResult)
		assert.Equal(t, "from file", fileResult)
		assert.Equal(t, "just default", defaultResult)

		// And cleanup should update the config file
		cleanup()

		// Verify the dumped file contains all keys
		data, err := os.ReadFile("github-mcp-server-config.json")
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Contains(t, result, "ENV_KEY")
		assert.Contains(t, result, "FILE_KEY")
		assert.Contains(t, result, "DEFAULT_KEY")
	})
}