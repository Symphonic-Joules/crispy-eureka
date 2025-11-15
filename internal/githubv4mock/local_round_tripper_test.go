package githubv4mock

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalRoundTripper(t *testing.T) {
	t.Run("executes HTTP request using handler directly", func(t *testing.T) {
		// Given a handler that returns a specific response
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test response"))
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request through the round tripper
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		// Then it should execute the handler and return the response
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "test response", string(body))
	})

	t.Run("passes request to handler correctly", func(t *testing.T) {
		// Given a handler that checks the request
		var receivedMethod string
		var receivedPath string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a POST request
		req := httptest.NewRequest("POST", "http://example.com/api/endpoint", nil)
		_, err := rt.RoundTrip(req)

		// Then the handler should receive the correct request details
		require.NoError(t, err)
		assert.Equal(t, "POST", receivedMethod)
		assert.Equal(t, "/api/endpoint", receivedPath)
	})

	t.Run("handles request body", func(t *testing.T) {
		// Given a handler that reads the request body
		var receivedBody string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, _ := io.ReadAll(r.Body)
			receivedBody = string(bodyBytes)
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request with a body
		requestBody := "test request body"
		req := httptest.NewRequest("POST", "http://example.com/test", strings.NewReader(requestBody))
		_, err := rt.RoundTrip(req)

		// Then the handler should receive the body
		require.NoError(t, err)
		assert.Equal(t, requestBody, receivedBody)
	})

	t.Run("handles request headers", func(t *testing.T) {
		// Given a handler that checks headers
		var receivedContentType string
		var receivedAuth string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request with headers
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")
		_, err := rt.RoundTrip(req)

		// Then the handler should receive the headers
		require.NoError(t, err)
		assert.Equal(t, "application/json", receivedContentType)
		assert.Equal(t, "Bearer token123", receivedAuth)
	})

	t.Run("returns response headers", func(t *testing.T) {
		// Given a handler that sets response headers
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		// Then the response should include the headers
		require.NoError(t, err)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})

	t.Run("handles different status codes", func(t *testing.T) {
		testCases := []struct {
			name       string
			statusCode int
		}{
			{"OK", http.StatusOK},
			{"Created", http.StatusCreated},
			{"Bad Request", http.StatusBadRequest},
			{"Not Found", http.StatusNotFound},
			{"Internal Server Error", http.StatusInternalServerError},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Given a handler that returns a specific status code
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				})
				rt := localRoundTripper{handler: handler}

				// When we make a request
				req := httptest.NewRequest("GET", "http://example.com/test", nil)
				resp, err := rt.RoundTrip(req)

				// Then the response should have the correct status code
				require.NoError(t, err)
				assert.Equal(t, tc.statusCode, resp.StatusCode)
			})
		}
	})

	t.Run("handles empty response body", func(t *testing.T) {
		// Given a handler that returns no body
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		// Then it should handle gracefully
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("handles large response body", func(t *testing.T) {
		// Given a handler that returns a large body
		largeBody := strings.Repeat("x", 10000)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(largeBody))
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		// Then it should handle the large body correctly
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, largeBody, string(body))
	})

	t.Run("preserves request URL", func(t *testing.T) {
		// Given a handler that checks the URL
		var receivedURL string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURL = r.URL.String()
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request with query parameters
		req := httptest.NewRequest("GET", "http://example.com/test?param1=value1&param2=value2", nil)
		_, err := rt.RoundTrip(req)

		// Then the handler should receive the full URL
		require.NoError(t, err)
		assert.Contains(t, receivedURL, "param1=value1")
		assert.Contains(t, receivedURL, "param2=value2")
	})

	t.Run("works with nil request body", func(t *testing.T) {
		// Given a handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make a request with nil body
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		// Then it should work correctly
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("can be used multiple times", func(t *testing.T) {
		// Given a round tripper
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})
		rt := localRoundTripper{handler: handler}

		// When we make multiple requests
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			_, err := rt.RoundTrip(req)
			require.NoError(t, err)
		}

		// Then all requests should be handled
		assert.Equal(t, 3, callCount)
	})
}