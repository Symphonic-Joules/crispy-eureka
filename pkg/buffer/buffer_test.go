package buffer

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessResponseAsRingBufferToEnd(t *testing.T) {
	t.Run("returns all lines when total is less than max", func(t *testing.T) {
		// Given an HTTP response with 3 lines and a buffer size of 5
		body := "line1\nline2\nline3"
		resp := createHTTPResponse(body)
		maxLines := 5

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then all lines should be returned
		require.NoError(t, err)
		assert.Equal(t, "line1\nline2\nline3", result)
		assert.Equal(t, 3, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("returns last N lines when total exceeds max", func(t *testing.T) {
		// Given an HTTP response with 7 lines and a buffer size of 3
		body := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
		resp := createHTTPResponse(body)
		maxLines := 3

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then only the last 3 lines should be returned
		require.NoError(t, err)
		assert.Equal(t, "line5\nline6\nline7", result)
		assert.Equal(t, 7, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles exactly max lines", func(t *testing.T) {
		// Given an HTTP response with exactly maxLines
		body := "line1\nline2\nline3\nline4\nline5"
		resp := createHTTPResponse(body)
		maxLines := 5

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then all lines should be returned in order
		require.NoError(t, err)
		assert.Equal(t, "line1\nline2\nline3\nline4\nline5", result)
		assert.Equal(t, 5, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles single line response", func(t *testing.T) {
		// Given an HTTP response with a single line
		body := "single line"
		resp := createHTTPResponse(body)
		maxLines := 10

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then the single line should be returned
		require.NoError(t, err)
		assert.Equal(t, "single line", result)
		assert.Equal(t, 1, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles empty response", func(t *testing.T) {
		// Given an HTTP response with no content
		body := ""
		resp := createHTTPResponse(body)
		maxLines := 10

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then an empty string should be returned
		require.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Equal(t, 0, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles maxLines of 1", func(t *testing.T) {
		// Given an HTTP response with multiple lines and maxLines of 1
		body := "line1\nline2\nline3"
		resp := createHTTPResponse(body)
		maxLines := 1

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then only the last line should be returned
		require.NoError(t, err)
		assert.Equal(t, "line3", result)
		assert.Equal(t, 3, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles lines with various content", func(t *testing.T) {
		// Given an HTTP response with lines containing special characters
		body := "line with spaces\n\tline with tab\nline-with-dashes\nline_with_underscores\n"
		resp := createHTTPResponse(body)
		maxLines := 10

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then all lines should be preserved correctly
		require.NoError(t, err)
		assert.Equal(t, "line with spaces\n\tline with tab\nline-with-dashes\nline_with_underscores\n", result)
		assert.Equal(t, 5, totalLines) // Empty line at end counts as a line
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles large number of lines", func(t *testing.T) {
		// Given an HTTP response with 1000 lines and a buffer size of 100
		var lines []string
		for i := 1; i <= 1000; i++ {
			lines = append(lines, "line"+string(rune('0'+i%10)))
		}
		body := strings.Join(lines, "\n")
		resp := createHTTPResponse(body)
		maxLines := 100

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then only the last 100 lines should be returned
		require.NoError(t, err)
		resultLines := strings.Split(result, "\n")
		assert.Equal(t, 100, len(resultLines))
		assert.Equal(t, 1000, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("ring buffer wraps correctly at boundary", func(t *testing.T) {
		// Given a response with lines that will cause multiple ring buffer wraps
		body := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13"
		resp := createHTTPResponse(body)
		maxLines := 4

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then the last 4 lines should be returned in correct order
		require.NoError(t, err)
		assert.Equal(t, "10\n11\n12\n13", result)
		assert.Equal(t, 13, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles lines with only whitespace", func(t *testing.T) {
		// Given an HTTP response with whitespace-only lines
		body := "line1\n   \n\t\t\nline4"
		resp := createHTTPResponse(body)
		maxLines := 10

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then whitespace lines should be preserved
		require.NoError(t, err)
		assert.Equal(t, "line1\n   \n\t\t\nline4", result)
		assert.Equal(t, 4, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles very long individual lines", func(t *testing.T) {
		// Given an HTTP response with a very long line
		longLine := strings.Repeat("a", 100000)
		body := "short\n" + longLine + "\nshort2"
		resp := createHTTPResponse(body)
		maxLines := 5

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then all lines including the long one should be processed
		require.NoError(t, err)
		assert.Contains(t, result, longLine)
		assert.Equal(t, 3, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("preserves original response object", func(t *testing.T) {
		// Given an HTTP response with headers and status
		resp := &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Header: http.Header{
				"Content-Type": []string{"text/plain"},
			},
			Body: io.NopCloser(strings.NewReader("line1\nline2")),
		}
		maxLines := 5

		// When we process the response
		_, _, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then the original response should be returned unchanged (except body read)
		require.NoError(t, err)
		assert.Equal(t, resp, returnedResp)
		assert.Equal(t, 200, returnedResp.StatusCode)
		assert.Equal(t, "200 OK", returnedResp.Status)
		assert.Equal(t, "text/plain", returnedResp.Header.Get("Content-Type"))
	})

	t.Run("returns correct line count even when buffer is smaller", func(t *testing.T) {
		// Given a response with many more lines than buffer size
		var lines []string
		for i := 1; i <= 50; i++ {
			lines = append(lines, "line")
		}
		body := strings.Join(lines, "\n")
		resp := createHTTPResponse(body)
		maxLines := 5

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then totalLines should reflect actual count, not buffer size
		require.NoError(t, err)
		resultLines := strings.Split(result, "\n")
		assert.Equal(t, 5, len(resultLines))
		assert.Equal(t, 50, totalLines)
		assert.Equal(t, resp, returnedResp)
	})

	t.Run("handles lines ending without newline", func(t *testing.T) {
		// Given a response where last line doesn't end with newline
		body := "line1\nline2\nline3"
		resp := createHTTPResponse(body)
		maxLines := 5

		// When we process the response
		result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, maxLines)

		// Then all lines should be captured correctly
		require.NoError(t, err)
		assert.Equal(t, "line1\nline2\nline3", result)
		assert.Equal(t, 3, totalLines)
		assert.Equal(t, resp, returnedResp)
	})
}

// createHTTPResponse is a helper function to create an HTTP response with the given body
func createHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}