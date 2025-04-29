package monitor

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	url := "https://example.com"
	interval := time.Second * 5

	m := NewMonitor(url, interval)
	require.NotNil(t, m)
	require.Equal(t, url, m.config.URL)
	require.Equal(t, interval, m.config.Interval)
	require.Equal(t, time.Second*30, m.config.Timeout)
	require.Equal(t, MethodHash, m.config.Method)
}

func TestNewMonitorWithConfig(t *testing.T) {
	config := &Config{
		URL:             "https://example.com",
		Interval:        time.Second * 10,
		Timeout:         time.Second * 20,
		RetryCount:      5,
		RetryInterval:   time.Second * 3,
		Method:          MethodLength,
		FollowRedirects: false,
	}

	m := NewMonitorWithConfig(config)
	require.NotNil(t, m)
	require.Equal(t, config.URL, m.config.URL)
	require.Equal(t, config.Interval, m.config.Interval)
	require.Equal(t, config.Timeout, m.config.Timeout)
	require.Equal(t, config.RetryCount, m.config.RetryCount)
	require.Equal(t, config.RetryInterval, m.config.RetryInterval)
	require.Equal(t, config.Method, m.config.Method)
	require.Equal(t, config.FollowRedirects, m.config.FollowRedirects)
}

func TestMonitorFetchContent(t *testing.T) {
	// Create a test server
	content := "Hello, World!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(content))
	}))
	defer server.Close()

	// Create a monitor for the test server
	config := &Config{
		URL:        server.URL,
		Interval:   time.Millisecond * 100,
		Timeout:    time.Second,
		RetryCount: 1,
		Method:     MethodHash,
	}
	m := NewMonitorWithConfig(config)

	// Fetch content
	fetchedContent, change, err := m.fetchContent()
	require.NoError(t, err)
	require.Equal(t, content, string(fetchedContent))
	require.Equal(t, server.URL, change.URL)
	require.Equal(t, 200, change.StatusCode)
	require.Equal(t, "text/plain", change.ContentType)
}

func TestMonitorDetectChange(t *testing.T) {
	t.Run("test hash change detection", func(t *testing.T) {
		// Setup test monitor
		m := NewMonitor("https://example.com", time.Second)
		m.config.Method = MethodHash

		// First check, no change expected
		content1 := []byte("Initial content")
		changed, _ := m.detectChange(content1)
		require.False(t, changed)

		// Second check with same content, no change expected
		changed, _ = m.detectChange(content1)
		require.False(t, changed)

		// Third check with different content, change expected
		content2 := []byte("Changed content")
		changed, details := m.detectChange(content2)
		require.True(t, changed)
		require.Contains(t, details, "differs at position")
	})

	t.Run("test length change detection", func(t *testing.T) {
		// Setup test monitor
		m := NewMonitor("https://example.com", time.Second)
		m.config.Method = MethodLength

		// First check, no change expected
		content1 := []byte("Initial content")
		changed, _ := m.detectChange(content1)
		require.False(t, changed)

		// Second check with different length, change expected
		content2 := []byte("Different length content string")
		changed, details := m.detectChange(content2)
		require.True(t, changed)
		require.Contains(t, details, "length")
	})

	t.Run("test custom change detection", func(t *testing.T) {
		// Setup test monitor with custom comparison function
		m := NewMonitor("https://example.com", time.Second)
		m.config.Method = MethodCustom
		m.config.CustomCompareFn = func(old, new []byte) (bool, string) {
			// Just a simple example: Check if the first byte changed
			if len(old) > 0 && len(new) > 0 && old[0] != new[0] {
				return true, "First byte changed"
			}
			return false, ""
		}

		// First check, no change expected
		content1 := []byte("Same first letter")
		changed, _ := m.detectChange(content1)
		require.False(t, changed)

		// Second check with different first letter, change expected
		content2 := []byte("Different first letter")
		changed, details := m.detectChange(content2)
		require.True(t, changed)
		require.Equal(t, "First byte changed", details)
	})
}

func TestByteSliceEqual(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		a := []byte{1, 2, 3, 4, 5}
		b := []byte{1, 2, 3, 4, 5}
		require.True(t, byteSliceEqual(a, b))
	})

	t.Run("different length", func(t *testing.T) {
		a := []byte{1, 2, 3}
		b := []byte{1, 2, 3, 4}
		require.False(t, byteSliceEqual(a, b))
	})

	t.Run("different content", func(t *testing.T) {
		a := []byte{1, 2, 3, 4, 5}
		b := []byte{1, 2, 3, 5, 5}
		require.False(t, byteSliceEqual(a, b))
	})

	t.Run("nil slices", func(t *testing.T) {
		var a, b []byte
		require.True(t, byteSliceEqual(a, b))
	})
}

func TestMonitorTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Delayed response"))
	}))
	defer server.Close()

	// Create a monitor with a short timeout
	config := &Config{
		URL:      server.URL,
		Interval: time.Millisecond * 100,
		Timeout:  time.Millisecond * 50, // Set timeout shorter than server delay
	}
	m := NewMonitorWithConfig(config)

	// Fetch should fail with timeout
	_, _, err := m.fetchContent()
	require.Error(t, err)
	require.Contains(t, err.Error(), "deadline exceeded")
}

func TestFindDifference(t *testing.T) {
	monitor := &Monitor{}

	tests := []struct {
		name     string
		old      string
		new      string
		expected string
	}{
		{
			name:     "identical contents",
			old:      "hello world",
			new:      "hello world",
			expected: "Content changed but no specific difference found",
		},
		{
			name:     "different length",
			old:      "hello",
			new:      "hello world",
			expected: "Content differs at position 5",
		},
		{
			name:     "single character difference",
			old:      "hello world",
			new:      "hello woRld",
			expected: "Content differs at position 8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := monitor.findDifference([]byte(tc.old), []byte(tc.new))
			require.Contains(t, result, tc.expected)
		})
	}
}

func TestNormalizeContent(t *testing.T) {
	monitor := &Monitor{
		config: Config{
			NormalizeWhitespace: true,
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normalize line endings",
			input:    "hello\r\nworld\r",
			expected: "hello\nworld\n",
		},
		{
			name:     "normalize whitespace",
			input:    "  hello   world  ",
			expected: "hello world",
		},
		{
			name:     "normalize mixed whitespace",
			input:    "  hello \r\n world  \r test",
			expected: "hello world test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Empty string と normalize line endings のテストでは whitespace normalization が不要
			if tc.name == "empty string" || tc.name == "normalize line endings" {
				monitor.config.NormalizeWhitespace = false
			} else {
				monitor.config.NormalizeWhitespace = true
			}

			result := string(monitor.normalizeContent([]byte(tc.input)))
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectChangeWithWhitespaceNormalization(t *testing.T) {
	// Test when NormalizeWhitespace is false
	config := DefaultConfig("https://example.com")
	config.NormalizeWhitespace = false
	monitor1 := NewMonitorWithConfig(config)

	// Set initial content
	monitor1.mu.Lock()
	monitor1.lastContent = []byte("hello world")
	monitor1.mu.Unlock()

	// Test with whitespace difference
	changed, _ := monitor1.detectChange([]byte("hello  world"))
	require.True(t, changed, "Should detect change when whitespace normalization is disabled")

	// Test when NormalizeWhitespace is true
	config.NormalizeWhitespace = true
	monitor2 := NewMonitorWithConfig(config)

	// Set initial content
	monitor2.mu.Lock()
	monitor2.lastContent = []byte("hello world")
	monitor2.mu.Unlock()

	// Test with whitespace difference
	changed, _ = monitor2.detectChange([]byte("hello  world"))
	require.False(t, changed, "Should not detect change when whitespace normalization is enabled")

	// Test with actual content difference
	changed, details := monitor2.detectChange([]byte("hello universe"))
	require.True(t, changed, "Should detect change with different content")
	require.Contains(t, details, "differs at position")
}

func TestMonitorWithTimestampFiltering(t *testing.T) {
	// Set up a monitor with timestamp filtering
	config := DefaultConfig("https://example.com")
	config.IgnoreTimestamps = true
	monitor := NewMonitorWithConfig(config)

	// Set initial content with a timestamp
	initialContent := []byte("Last updated: 2023-05-01T12:00:00Z")
	monitor.mu.Lock()
	monitor.lastContent = initialContent
	monitor.mu.Unlock()

	// New content with only the timestamp changed
	updatedContent := []byte("Last updated: 2023-05-01T13:00:00Z")

	// Should not detect a change since we're ignoring timestamps
	changed, _ := monitor.detectChange(updatedContent)
	require.False(t, changed, "Should not detect a change when only timestamps differ and filtering is enabled")

	// New content with other changes
	otherContent := []byte("Last updated: 2023-05-01T13:00:00Z and new content")

	// Should detect a change since other content changed
	changed, details := monitor.detectChange(otherContent)
	require.True(t, changed, "Should detect changes in non-timestamp content")
	require.Contains(t, details, "differs at position")
}

func TestMonitorWithCustomFilters(t *testing.T) {
	// Create a custom regex filter to ignore specific pattern
	customFilter, err := NewRegexFilter("version: [0-9.]+", "version: X.Y.Z", "Ignore version numbers")
	require.NoError(t, err)

	// Set up a monitor with the custom filter
	config := DefaultConfig("https://example.com")
	config.ContentFilters = ContentFilterList{customFilter}
	monitor := NewMonitorWithConfig(config)

	// Set initial content with a version number
	initialContent := []byte("Software version: 1.2.3")
	monitor.mu.Lock()
	monitor.lastContent = initialContent
	monitor.mu.Unlock()

	// New content with only the version changed
	updatedContent := []byte("Software version: 1.2.4")

	// Should not detect a change since we're filtering out version numbers
	changed, _ := monitor.detectChange(updatedContent)
	require.False(t, changed, "Should not detect a change when only version numbers differ")

	// New content with other changes
	otherContent := []byte("Software version: 1.2.4 with new features")

	// Should detect a change since other content changed
	changed, details := monitor.detectChange(otherContent)
	require.True(t, changed, "Should detect changes in non-filtered content")
	require.Contains(t, details, "differs at position")
}

func TestMonitorWithMultipleFilters(t *testing.T) {
	// Create multiple filters
	tsFilter, err := NewTimestampFilter()
	require.NoError(t, err)

	versionFilter, err := NewRegexFilter("version: [0-9.]+", "version: X.Y.Z", "Ignore version numbers")
	require.NoError(t, err)

	// Set up a monitor with multiple filters
	config := DefaultConfig("https://example.com")
	config.ContentFilters = ContentFilterList{tsFilter, versionFilter}
	monitor := NewMonitorWithConfig(config)

	// Set initial content
	initialContent := []byte("Updated: 2023-05-01T12:00:00Z, version: 1.2.3")
	monitor.mu.Lock()
	monitor.lastContent = initialContent
	monitor.mu.Unlock()

	// New content with only filtered elements changed
	updatedContent := []byte("Updated: 2023-05-01T13:00:00Z, version: 1.2.4")

	// Should not detect a change since we're filtering both timestamps and versions
	changed, _ := monitor.detectChange(updatedContent)
	require.False(t, changed, "Should not detect a change when only filtered elements differ")

	// New content with other changes
	otherContent := []byte("Updated: 2023-05-01T13:00:00Z, version: 1.2.4, new feature added")

	// Should detect a change
	changed, details := monitor.detectChange(otherContent)
	require.True(t, changed, "Should detect changes in non-filtered content")
	require.Contains(t, details, "differs at position")
}
