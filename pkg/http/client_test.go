package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		options *ClientOptions
	}{
		{
			name:    "nil options",
			options: nil,
		},
		{
			name: "custom timeout",
			options: &ClientOptions{
				Timeout: time.Second * 10,
			},
		},
		{
			name: "no redirects",
			options: &ClientOptions{
				FollowRedirects: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.options)
			require.NotNil(t, client)

			if tt.options == nil {
				require.Equal(t, DefaultClientOptions().Timeout, client.Timeout)
			} else {
				if tt.options.Timeout > 0 {
					require.Equal(t, tt.options.Timeout, client.Timeout)
				}

				if !tt.options.FollowRedirects {
					require.NotNil(t, client.CheckRedirect)
				}
			}
		})
	}
}

func TestAddHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	headers := map[string]string{
		"X-Test":       "test-value",
		"Content-Type": "application/json",
	}
	userAgent := "TestAgent/1.0"

	AddHeaders(req, headers, userAgent)

	// Check User-Agent is set
	require.Equal(t, userAgent, req.Header.Get("User-Agent"))

	// Check custom headers are set
	for key, value := range headers {
		require.Equal(t, value, req.Header.Get(key))
	}

	// Test with existing User-Agent
	req, _ = http.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("User-Agent", "ExistingAgent/1.0")

	AddHeaders(req, headers, userAgent)

	// Original User-Agent should be preserved
	require.Equal(t, "ExistingAgent/1.0", req.Header.Get("User-Agent"))
}
