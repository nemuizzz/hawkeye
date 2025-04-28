// Package hawkeye provides a simple API for monitoring web URLs for changes.
// This package wraps the lower-level functionality in pkg/monitor to provide
// a simpler, more convenient interface as shown in the README examples.
package hawkeye

import (
	"context"
	"time"

	"github.com/nemuizzz/hawkeye/pkg/monitor"
)

// Monitor watches a URL for changes
type Monitor struct {
	internal *monitor.Monitor
	ctx      context.Context
	cancel   context.CancelFunc
	url      string
	interval time.Duration
	headers  map[string]string
	ignore   []string
	timeout  time.Duration
	retries  int
	retryInt time.Duration
}

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

// NewMonitor creates a new monitor with the specified URL and check interval
func NewMonitor(url string, interval time.Duration) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	config := &monitor.Config{
		URL:             url,
		Interval:        interval,
		Timeout:         time.Second * 30, // default timeout
		Headers:         make(map[string]string),
		IgnoreSelectors: []string{},
		Method:          monitor.MethodHash,
		RetryCount:      3,                // default retry count
		RetryInterval:   time.Second * 10, // default retry interval
		FollowRedirects: true,
	}

	return &Monitor{
		internal: monitor.NewMonitorWithConfig(config),
		ctx:      ctx,
		cancel:   cancel,
		url:      url,
		interval: interval,
		headers:  make(map[string]string),
		ignore:   []string{},
		timeout:  time.Second * 30,
		retries:  3,
		retryInt: time.Second * 10,
	}
}

// Start begins monitoring the URL for changes
func (m *Monitor) Start() <-chan Change {
	internalChanges := m.internal.Start()
	changes := make(chan Change)

	go func() {
		defer close(changes)
		for {
			select {
			case change, ok := <-internalChanges:
				if !ok {
					return
				}

				// Convert from internal Change type to public API Change type
				changes <- Change{
					URL:         change.URL,
					Timestamp:   change.Timestamp,
					HasChanged:  change.HasChanged,
					StatusCode:  change.StatusCode,
					ContentType: change.ContentType,
					Error:       change.Error,
					Details:     change.Details,
				}
			case <-m.ctx.Done():
				return
			}
		}
	}()

	return changes
}

// Stop stops the monitoring
func (m *Monitor) Stop() {
	m.cancel()
	m.internal.Stop()
}

// recreateMonitor recreates the internal monitor with current settings
func (m *Monitor) recreateMonitor() {
	config := &monitor.Config{
		URL:             m.url,
		Interval:        m.interval,
		Timeout:         m.timeout,
		Headers:         m.headers,
		IgnoreSelectors: m.ignore,
		Method:          monitor.MethodHash,
		RetryCount:      m.retries,
		RetryInterval:   m.retryInt,
		FollowRedirects: true,
	}

	// Stop the existing monitor if it's running
	if m.internal != nil {
		m.internal.Stop()
	}

	m.internal = monitor.NewMonitorWithConfig(config)
}

// WithHeaders adds custom HTTP headers to the monitor
func (m *Monitor) WithHeaders(headers map[string]string) *Monitor {
	m.headers = headers
	m.recreateMonitor()
	return m
}

// WithIgnoreSelectors adds CSS selectors to ignore when checking for changes
func (m *Monitor) WithIgnoreSelectors(selectors []string) *Monitor {
	m.ignore = selectors
	m.recreateMonitor()
	return m
}

// WithTimeout sets the HTTP request timeout
func (m *Monitor) WithTimeout(timeout time.Duration) *Monitor {
	m.timeout = timeout
	m.recreateMonitor()
	return m
}

// WithRetries sets the number of retry attempts and interval between retries
func (m *Monitor) WithRetries(count int, interval time.Duration) *Monitor {
	m.retries = count
	m.retryInt = interval
	m.recreateMonitor()
	return m
}

// WithContext associates the monitor with a context
// This is a more Go 1.23-friendly approach to monitor lifecycle management
func (m *Monitor) WithContext(ctx context.Context) *Monitor {
	// Cancel the existing context
	m.cancel()

	// Create a new context that will be canceled when either the provided context
	// or our own internal context is canceled
	newCtx, newCancel := context.WithCancel(ctx)

	// Update the monitor's context
	m.ctx = newCtx
	m.cancel = newCancel

	return m
}

// GetURL returns the URL being monitored
func (m *Monitor) GetURL() string {
	return m.url
}

// Iterator returns an iterator that yields changes.
func (m *Monitor) Iterator() func(yield func(Change) bool) {
	changes := m.Start()

	return func(yield func(Change) bool) {
		for change := range changes {
			if !yield(change) {
				m.Stop()
				return
			}
		}
	}
}

// NewMonitorWithContext creates a new monitor with a context
func NewMonitorWithContext(ctx context.Context, url string, interval time.Duration) *Monitor {
	monitor := NewMonitor(url, interval)
	return monitor.WithContext(ctx)
}
