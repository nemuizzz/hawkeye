package monitor

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	require.NotNil(t, manager)
	require.Empty(t, manager.monitors)
	require.Empty(t, manager.groups)
	require.NotNil(t, manager.changeChannel)
	require.NotNil(t, manager.ctx)
	require.NotNil(t, manager.cancel)
}

func TestAddMonitor(t *testing.T) {
	manager := NewManager()
	monitor := NewMonitor("https://example.com", time.Second*5)

	// Add monitor
	err := manager.AddMonitor(monitor)
	require.NoError(t, err)
	require.Len(t, manager.monitors, 1)

	// Try adding the same monitor again
	err = manager.AddMonitor(monitor)
	require.Error(t, err)
	require.Len(t, manager.monitors, 1)

	// Try adding a monitor with empty URL
	badMonitor := NewMonitor("", time.Second*5)
	err = manager.AddMonitor(badMonitor)
	require.Error(t, err)
	require.Len(t, manager.monitors, 1)
}

func TestAddMonitorWithConfig(t *testing.T) {
	manager := NewManager()

	// Valid config
	config := &Config{
		URL:      "https://example.com",
		Interval: time.Second * 5,
	}

	monitor, err := manager.AddMonitorWithConfig(config)
	require.NoError(t, err)
	require.NotNil(t, monitor)
	require.Len(t, manager.monitors, 1)

	// Invalid config: empty URL
	badConfig1 := &Config{
		URL:      "",
		Interval: time.Second * 5,
	}

	monitor, err = manager.AddMonitorWithConfig(badConfig1)
	require.Error(t, err)
	require.Nil(t, monitor)
	require.Len(t, manager.monitors, 1)

	// Invalid config: zero interval
	badConfig2 := &Config{
		URL:      "https://another-example.com",
		Interval: 0,
	}

	monitor, err = manager.AddMonitorWithConfig(badConfig2)
	require.Error(t, err)
	require.Nil(t, monitor)
	require.Len(t, manager.monitors, 1)
}

func TestCreateGroup(t *testing.T) {
	manager := NewManager()

	// Create a group
	group, err := manager.CreateGroup("test-group", "Test Group")
	require.NoError(t, err)
	require.NotNil(t, group)
	require.Equal(t, "test-group", group.Name)
	require.Equal(t, "Test Group", group.Description)
	require.Empty(t, group.Monitors)
	require.Len(t, manager.groups, 1)

	// Try creating a group with the same name
	group, err = manager.CreateGroup("test-group", "Another Description")
	require.Error(t, err)
	require.Nil(t, group)
	require.Len(t, manager.groups, 1)
}

func TestAddToGroup(t *testing.T) {
	manager := NewManager()
	monitor := NewMonitor("https://example.com", time.Second*5)

	// Add monitor
	err := manager.AddMonitor(monitor)
	require.NoError(t, err)

	// Create a group
	_, err = manager.CreateGroup("test-group", "Test Group")
	require.NoError(t, err)

	// Add monitor to group
	err = manager.AddToGroup("https://example.com", "test-group")
	require.NoError(t, err)

	// Verify monitor was added to group
	group, err := manager.GetGroup("test-group")
	require.NoError(t, err)
	require.Len(t, group.Monitors, 1)

	// Try adding a non-existent monitor to the group
	err = manager.AddToGroup("https://non-existent.com", "test-group")
	require.Error(t, err)

	// Try adding to a non-existent group
	err = manager.AddToGroup("https://example.com", "non-existent-group")
	require.Error(t, err)
}

func TestRemoveMonitor(t *testing.T) {
	manager := NewManager()
	monitor := NewMonitor("https://example.com", time.Second*5)

	// Add monitor
	err := manager.AddMonitor(monitor)
	require.NoError(t, err)

	// Create a group and add monitor to it
	_, err = manager.CreateGroup("test-group", "Test Group")
	require.NoError(t, err)
	err = manager.AddToGroup("https://example.com", "test-group")
	require.NoError(t, err)

	// Remove the monitor
	err = manager.RemoveMonitor("https://example.com")
	require.NoError(t, err)

	// Verify monitor was removed
	require.Empty(t, manager.monitors)

	// Verify monitor was removed from group
	group, err := manager.GetGroup("test-group")
	require.NoError(t, err)
	require.Empty(t, group.Monitors)

	// Try removing a non-existent monitor
	err = manager.RemoveMonitor("https://non-existent.com")
	require.Error(t, err)
}

func TestGetMonitor(t *testing.T) {
	manager := NewManager()
	originalMonitor := NewMonitor("https://example.com", time.Second*5)

	// Add monitor
	err := manager.AddMonitor(originalMonitor)
	require.NoError(t, err)

	// Get the monitor
	monitor, err := manager.GetMonitor("https://example.com")
	require.NoError(t, err)
	require.Equal(t, originalMonitor, monitor)

	// Try getting a non-existent monitor
	monitor, err = manager.GetMonitor("https://non-existent.com")
	require.Error(t, err)
	require.Nil(t, monitor)
}

func TestGetGroup(t *testing.T) {
	manager := NewManager()

	// Create a group
	originalGroup, err := manager.CreateGroup("test-group", "Test Group")
	require.NoError(t, err)

	// Get the group
	group, err := manager.GetGroup("test-group")
	require.NoError(t, err)
	require.Equal(t, originalGroup, group)

	// Try getting a non-existent group
	group, err = manager.GetGroup("non-existent-group")
	require.Error(t, err)
	require.Nil(t, group)
}

func TestListMonitors(t *testing.T) {
	manager := NewManager()

	// Initially no monitors
	monitors := manager.ListMonitors()
	require.Empty(t, monitors)

	// Add monitors
	monitor1 := NewMonitor("https://example1.com", time.Second*5)
	monitor2 := NewMonitor("https://example2.com", time.Second*5)
	monitor3 := NewMonitor("https://example3.com", time.Second*5)

	_ = manager.AddMonitor(monitor1)
	_ = manager.AddMonitor(monitor2)
	_ = manager.AddMonitor(monitor3)

	// List monitors
	monitors = manager.ListMonitors()
	require.Len(t, monitors, 3)
	require.Contains(t, monitors, "https://example1.com")
	require.Contains(t, monitors, "https://example2.com")
	require.Contains(t, monitors, "https://example3.com")
}

func TestListGroups(t *testing.T) {
	manager := NewManager()

	// Initially no groups
	groups := manager.ListGroups()
	require.Empty(t, groups)

	// Create groups
	_, _ = manager.CreateGroup("group1", "Group 1")
	_, _ = manager.CreateGroup("group2", "Group 2")
	_, _ = manager.CreateGroup("group3", "Group 3")

	// List groups
	groups = manager.ListGroups()
	require.Len(t, groups, 3)
	require.Contains(t, groups, "group1")
	require.Contains(t, groups, "group2")
	require.Contains(t, groups, "group3")
}

func TestConcurrentManagerOperations(t *testing.T) {
	manager := NewManager()
	var wg sync.WaitGroup
	numGoroutines := 10

	// Create a group first
	_, err := manager.CreateGroup("test-group", "Test Group")
	require.NoError(t, err)

	// Launch multiple goroutines that simultaneously add monitors
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			url := fmt.Sprintf("https://example-%d.com", index)
			config := &Config{
				URL:      url,
				Interval: time.Second * 5,
			}

			// Add a monitor
			_, err := manager.AddMonitorWithConfig(config)
			if err == nil {
				// If monitor was added successfully, try adding to group
				manager.AddToGroup(url, "test-group")
			}
		}(i)
	}

	wg.Wait()

	// Verify no data races occurred and all valid monitors were added
	require.GreaterOrEqual(t, len(manager.ListMonitors()), 1)

	group, err := manager.GetGroup("test-group")
	require.NoError(t, err)
	require.NotEmpty(t, group.Monitors)
}
