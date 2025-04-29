package monitor

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	customhttp "github.com/nemuizzz/hawkeye/pkg/http"
	"github.com/nemuizzz/hawkeye/pkg/utils"
	"github.com/nemuizzz/hawkeye/pkg/version"
)

// ChangeDetectionMethod represents the method used to detect changes
type ChangeDetectionMethod int

const (
	// MethodHash compares the hash of the content
	MethodHash ChangeDetectionMethod = iota
	// MethodLength compares the content length
	MethodLength
	// MethodCustom uses a custom comparison function
	MethodCustom
)

// Error definitions
var (
	ErrURLEmpty        = errors.New("URL cannot be empty")
	ErrInvalidInterval = errors.New("interval must be greater than zero")
	ErrMonitorStopped  = errors.New("monitor has been stopped")
)

// Change represents a detected change in a monitored URL
type Change struct {
	URL         string    `json:"url"`
	Timestamp   time.Time `json:"timestamp"`
	HasChanged  bool      `json:"has_changed"`
	StatusCode  int       `json:"status_code,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	Error       string    `json:"error,omitempty"`
	Details     string    `json:"details,omitempty"`
}

// Config holds the configuration for a monitor
type Config struct {
	URL                 string
	Interval            time.Duration
	Timeout             time.Duration
	Headers             map[string]string
	IgnoreSelectors     []string
	Method              ChangeDetectionMethod
	CustomCompareFn     func([]byte, []byte) (bool, string)
	RetryCount          int
	RetryInterval       time.Duration
	FollowRedirects     bool
	IncludeResponseBody bool
	NormalizeWhitespace bool
	ContentFilters      ContentFilterList
	IgnoreTimestamps    bool
}

// Monitor watches a URL for changes
type Monitor struct {
	config       Config
	client       *http.Client
	lastContent  []byte
	lastCheck    time.Time
	changes      chan Change
	stop         chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	checkCount   int64
	status       string
	isFirstCheck bool
	filters      ContentFilterList
}

// DefaultConfig returns a default configuration
func DefaultConfig(url string) *Config {
	return &Config{
		URL:                 url,
		Interval:            time.Minute * 5,
		Timeout:             time.Second * 30,
		Method:              MethodHash,
		RetryCount:          3,
		RetryInterval:       time.Second * 10,
		FollowRedirects:     true,
		NormalizeWhitespace: false,
		IgnoreTimestamps:    false,
	}
}

// NewMonitor creates a new monitor with default settings
func NewMonitor(url string, interval time.Duration) *Monitor {
	config := DefaultConfig(url)
	config.Interval = interval
	return NewMonitorWithConfig(config)
}

// NewMonitorWithConfig creates a new monitor with the given configuration
func NewMonitorWithConfig(config *Config) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	clientOpts := &customhttp.ClientOptions{
		Timeout:         config.Timeout,
		FollowRedirects: config.FollowRedirects,
	}

	client := customhttp.NewClient(clientOpts)

	// Set up filters
	var filters ContentFilterList

	// Add the provided filters
	if config.ContentFilters != nil {
		filters = append(filters, config.ContentFilters...)
	}

	// Add default timestamp filter if configured
	if config.IgnoreTimestamps {
		tsFilter, _ := NewTimestampFilter()
		if tsFilter != nil {
			filters = append(filters, tsFilter)
		}
	}

	return &Monitor{
		config:       *config,
		client:       client,
		changes:      make(chan Change),
		stop:         make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
		isFirstCheck: true,
		filters:      filters,
	}
}

// Start begins monitoring the URL for changes
func (m *Monitor) Start() <-chan Change {
	go m.run()
	return m.changes
}

// Stop stops the monitoring
func (m *Monitor) Stop() {
	m.cancel()
	close(m.stop)
}

// run is the main monitoring loop
func (m *Monitor) run() {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()
	defer close(m.changes)

	// Perform first check immediately
	m.performCheck()

	for {
		select {
		case <-ticker.C:
			m.performCheck()
		case <-m.ctx.Done():
			return
		}
	}
}

// performCheck checks the URL for changes
func (m *Monitor) performCheck() {
	m.mu.Lock()
	m.checkCount++
	m.status = "checking"
	m.mu.Unlock()

	var change Change
	var content []byte
	var err error

	for i := 0; i <= m.config.RetryCount; i++ {
		if i > 0 {
			time.Sleep(m.config.RetryInterval)
		}

		content, change, err = m.fetchContent()
		if err == nil {
			break
		}

		// Last attempt, report the error
		if i == m.config.RetryCount {
			change = Change{
				URL:       m.config.URL,
				Timestamp: time.Now(),
				Error:     err.Error(),
			}
		}
	}

	if err != nil {
		m.changes <- change
		return
	}

	changed, details := m.detectChange(content)

	m.mu.Lock()
	m.lastCheck = time.Now()
	m.status = "idle"
	isFirst := m.isFirstCheck
	m.isFirstCheck = false
	m.mu.Unlock()

	// Don't report a change on the first check
	if isFirst {
		return
	}

	if changed {
		change.HasChanged = true
		change.Details = details
		m.changes <- change
	}
}

// fetchContent retrieves the content from the URL
func (m *Monitor) fetchContent() ([]byte, Change, error) {
	req, err := http.NewRequestWithContext(m.ctx, "GET", m.config.URL, nil)
	if err != nil {
		return nil, Change{}, err
	}

	// Add custom headers
	customhttp.AddHeaders(req, m.config.Headers, version.UserAgent())

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, Change{}, err
	}
	defer resp.Body.Close()

	change := Change{
		URL:         m.config.URL,
		Timestamp:   time.Now(),
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, change, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, change, err
	}

	return content, change, nil
}

// detectChange checks if the content has changed
func (m *Monitor) detectChange(content []byte) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If this is the first check, just store the content
	if m.lastContent == nil {
		m.lastContent = content
		return false, ""
	}

	// Apply filters to content if any are defined
	compareContent := content
	compareLast := m.lastContent

	// Apply content filters
	if len(m.filters) > 0 {
		compareContent = m.filters.Apply(compareContent)
		compareLast = m.filters.Apply(compareLast)
	}

	// Normalize content if configured
	if m.config.NormalizeWhitespace {
		compareContent = m.normalizeContent(compareContent)
		compareLast = m.normalizeContent(compareLast)
	}

	switch m.config.Method {
	case MethodHash:
		currentHash := m.calculateHash(compareContent)
		lastHash := m.calculateHash(compareLast)
		changed := !byteSliceEqual(currentHash, lastHash)

		if changed {
			details := m.findDifference(compareLast, compareContent)
			m.lastContent = content // Store the original content
			return true, details
		}

	case MethodLength:
		oldLen := len(compareLast)
		newLen := len(compareContent)

		if oldLen != newLen {
			details := m.findDifference(compareLast, compareContent)
			m.lastContent = content // Store the original content
			return true, details
		}

	case MethodCustom:
		if m.config.CustomCompareFn != nil {
			changed, details := m.config.CustomCompareFn(compareLast, compareContent)

			if changed {
				m.lastContent = content // Store the original content
				return true, details
			}
		}
	}

	return false, ""
}

// calculateHash calculates the SHA-256 hash of the content
func (m *Monitor) calculateHash(content []byte) []byte {
	hash := sha256.Sum256(content)
	return hash[:]
}

// findDifference finds the difference between old and new content
// It returns a description of what changed
func (m *Monitor) findDifference(oldContent, newContent []byte) string {
	// Convert to string for easier comparison
	oldStr := string(oldContent)
	newStr := string(newContent)

	// Find the first different character
	diffPos := -1
	for i := 0; i < len(oldStr) && i < len(newStr); i++ {
		if oldStr[i] != newStr[i] {
			diffPos = i
			break
		}
	}

	// If we found a difference or lengths are different
	if diffPos >= 0 || len(oldStr) != len(newStr) {
		// If no specific difference found but lengths differ,
		// set the position to the length of the shorter string
		if diffPos < 0 {
			diffPos = min(len(oldStr), len(newStr))
		}

		// Get context around the difference
		start := diffPos - 20
		if start < 0 {
			start = 0
		}

		oldEnd := diffPos + 20
		if oldEnd > len(oldStr) {
			oldEnd = len(oldStr)
		}

		newEnd := diffPos + 20
		if newEnd > len(newStr) {
			newEnd = len(newStr)
		}

		// Use 1-based position for human readability
		return fmt.Sprintf("Content differs at position %d\nOld: ...%s...\nNew: ...%s...",
			diffPos, oldStr[start:oldEnd], newStr[start:newEnd])
	}

	return "Content changed but no specific difference found"
}

// normalizeContent normalizes content to prevent false positives
// It handles common cases like whitespace differences
func (m *Monitor) normalizeContent(content []byte) []byte {
	if len(content) == 0 {
		return content
	}

	// Convert to string for easier manipulation
	str := string(content)

	// Replace carriage returns to normalize line endings
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "\r", "\n")

	// Normalize consecutive whitespace if configured
	// This is a simplification as proper regex would require the regexp package
	if m.config.NormalizeWhitespace {
		str = strings.Join(strings.Fields(str), " ")
		// Trim leading/trailing whitespace
		str = strings.TrimSpace(str)
	}

	return []byte(str)
}

// GetStatus returns the current status of the monitor
func (m *Monitor) GetStatus() (time.Time, string, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastCheck, m.status, m.checkCount
}

// GetURL returns the URL being monitored
func (m *Monitor) GetURL() string {
	return m.config.URL
}

// byteSliceEqual compares two byte slices for equality
func byteSliceEqual(a, b []byte) bool {
	return utils.ByteSliceEqual(a, b)
}
