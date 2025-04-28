package monitor

import (
	"context"
	"fmt"
	"sync"
)

// MonitorMap represents a map of URLs to Monitors
type MonitorMap map[string]*Monitor

// MonitorGroup represents a group of monitors with the same configuration
type MonitorGroup struct {
	Name        string
	Description string
	Monitors    MonitorMap
}

// Manager handles multiple monitors
type Manager struct {
	monitors      MonitorMap
	groups        map[string]*MonitorGroup
	changeChannel chan Change
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new Manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		monitors:      make(MonitorMap),
		groups:        make(map[string]*MonitorGroup),
		changeChannel: make(chan Change),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// AddMonitor adds a new monitor to the manager
func (m *Manager) AddMonitor(monitor *Monitor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	url := monitor.GetURL()
	if url == "" {
		return ErrURLEmpty
	}

	if _, exists := m.monitors[url]; exists {
		return fmt.Errorf("monitor for URL '%s' already exists", url)
	}

	m.monitors[url] = monitor
	return nil
}

// AddMonitorWithConfig creates and adds a new monitor with the given configuration
func (m *Manager) AddMonitorWithConfig(config *Config) (*Monitor, error) {
	if config.URL == "" {
		return nil, ErrURLEmpty
	}

	if config.Interval <= 0 {
		return nil, ErrInvalidInterval
	}

	monitor := NewMonitorWithConfig(config)
	err := m.AddMonitor(monitor)
	if err != nil {
		return nil, err
	}

	return monitor, nil
}

// CreateGroup creates a new monitor group
func (m *Manager) CreateGroup(name, description string) (*MonitorGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.groups[name]; exists {
		return nil, fmt.Errorf("group '%s' already exists", name)
	}

	group := &MonitorGroup{
		Name:        name,
		Description: description,
		Monitors:    make(MonitorMap),
	}

	m.groups[name] = group
	return group, nil
}

// AddToGroup adds a monitor to a group
func (m *Manager) AddToGroup(url, groupName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	monitor, exists := m.monitors[url]
	if !exists {
		return fmt.Errorf("no monitor found for URL '%s'", url)
	}

	group, exists := m.groups[groupName]
	if !exists {
		return fmt.Errorf("group '%s' does not exist", groupName)
	}

	group.Monitors[url] = monitor
	return nil
}

// RemoveMonitor removes a monitor
func (m *Manager) RemoveMonitor(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	monitor, exists := m.monitors[url]
	if !exists {
		return fmt.Errorf("no monitor found for URL '%s'", url)
	}

	// Stop the monitor
	monitor.Stop()

	// Remove from all groups
	for _, group := range m.groups {
		delete(group.Monitors, url)
	}

	// Remove from manager
	delete(m.monitors, url)
	return nil
}

// GetMonitor returns a monitor by URL
func (m *Manager) GetMonitor(url string) (*Monitor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	monitor, exists := m.monitors[url]
	if !exists {
		return nil, fmt.Errorf("no monitor found for URL '%s'", url)
	}

	return monitor, nil
}

// GetGroup returns a group by name
func (m *Manager) GetGroup(name string) (*MonitorGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[name]
	if !exists {
		return nil, fmt.Errorf("group '%s' does not exist", name)
	}

	return group, nil
}

// ListMonitors returns a list of all monitored URLs
func (m *Manager) ListMonitors() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	urls := make([]string, 0, len(m.monitors))
	for url := range m.monitors {
		urls = append(urls, url)
	}

	return urls
}

// ListGroups returns a list of all group names
func (m *Manager) ListGroups() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make([]string, 0, len(m.groups))
	for name := range m.groups {
		groups = append(groups, name)
	}

	return groups
}

// Start starts all monitors and returns a channel for all changes
func (m *Manager) Start() <-chan Change {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, monitor := range m.monitors {
		changes := monitor.Start()
		go m.forwardChanges(changes)
	}

	return m.changeChannel
}

// forwardChanges forwards changes from a monitor to the manager's change channel
func (m *Manager) forwardChanges(changes <-chan Change) {
	for change := range changes {
		select {
		case m.changeChannel <- change:
		case <-m.ctx.Done():
			return
		}
	}
}

// StartMonitor starts a specific monitor
func (m *Manager) StartMonitor(url string) (<-chan Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	monitor, exists := m.monitors[url]
	if !exists {
		return nil, fmt.Errorf("no monitor found for URL '%s'", url)
	}

	changes := monitor.Start()
	go m.forwardChanges(changes)

	return m.changeChannel, nil
}

// StartGroup starts all monitors in a group
func (m *Manager) StartGroup(groupName string) (<-chan Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupName]
	if !exists {
		return nil, fmt.Errorf("group '%s' does not exist", groupName)
	}

	for _, monitor := range group.Monitors {
		changes := monitor.Start()
		go m.forwardChanges(changes)
	}

	return m.changeChannel, nil
}

// Stop stops all monitors
func (m *Manager) Stop() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, monitor := range m.monitors {
		monitor.Stop()
	}

	close(m.changeChannel)
}

// StopMonitor stops a specific monitor
func (m *Manager) StopMonitor(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	monitor, exists := m.monitors[url]
	if !exists {
		return fmt.Errorf("no monitor found for URL '%s'", url)
	}

	monitor.Stop()
	return nil
}

// StopGroup stops all monitors in a group
func (m *Manager) StopGroup(groupName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupName]
	if !exists {
		return fmt.Errorf("group '%s' does not exist", groupName)
	}

	for _, monitor := range group.Monitors {
		monitor.Stop()
	}

	return nil
}
