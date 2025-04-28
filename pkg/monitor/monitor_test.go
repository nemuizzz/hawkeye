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
		require.Contains(t, details, "hash")
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
