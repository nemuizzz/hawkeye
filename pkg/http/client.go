package http

import (
	"net/http"
	"time"

	"github.com/nemuizzz/hawkeye/pkg/version"
)

// ClientOptions configures the HTTP client
type ClientOptions struct {
	Timeout         time.Duration
	FollowRedirects bool
	Headers         map[string]string
	UserAgent       string
}

// DefaultClientOptions returns default HTTP client options
func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Timeout:         time.Second * 30,
		FollowRedirects: true,
		UserAgent:       version.UserAgent(),
	}
}

// NewClient creates a new HTTP client with the provided options
func NewClient(opts *ClientOptions) *http.Client {
	if opts == nil {
		opts = DefaultClientOptions()
	}

	client := &http.Client{
		Timeout: opts.Timeout,
	}

	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}

// AddHeaders adds custom headers to an HTTP request
func AddHeaders(req *http.Request, headers map[string]string, defaultUserAgent string) {
	// Set default User-Agent if not already set
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}
