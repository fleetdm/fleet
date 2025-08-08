package menu

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
)

// MockFactory implements Factory interface for testing
type MockFactory struct {
	Items      []MockMenuItem
	Separators int
}

// NewMockFactory creates a new mock factory for testing
func NewMockFactory() *MockFactory {
	return &MockFactory{
		Items:      []MockMenuItem{},
		Separators: 0,
	}
}

// AddMenuItem creates a new mock menu item
func (m *MockFactory) AddMenuItem(title string, _ string) Item {
	item := NewMockMenuItem(title)
	m.Items = append(m.Items, *item)
	return item
}

// AddSeparator increments the separator count
func (m *MockFactory) AddSeparator() {
	m.Separators++
}

// MockMenuItem provides a testable implementation of Item
type MockMenuItem struct {
	Title     string
	Enabled   bool
	Visible   bool
	clickedCh chan struct{}
	History   []string // Track all operations for testing
}

// NewMockMenuItem creates a new mock menu item
func NewMockMenuItem(title string) *MockMenuItem {
	return &MockMenuItem{
		Title:     title,
		Enabled:   true,
		Visible:   true,
		clickedCh: make(chan struct{}, 1),
		History:   []string{},
	}
}

func (m *MockMenuItem) SetTitle(title string) {
	m.Title = title
	m.History = append(m.History, "SetTitle:"+title)
}

func (m *MockMenuItem) Enable() {
	m.Enabled = true
	m.History = append(m.History, "Enable")
}

func (m *MockMenuItem) Disable() {
	m.Enabled = false
	m.History = append(m.History, "Disable")
}

func (m *MockMenuItem) Show() {
	m.Visible = true
	m.History = append(m.History, "Show")
}

func (m *MockMenuItem) Hide() {
	m.Visible = false
	m.History = append(m.History, "Hide")
}

func (m *MockMenuItem) ClickedCh() <-chan struct{} {
	return m.clickedCh
}

// SimulateClick simulates a user clicking the menu item
func (m *MockMenuItem) SimulateClick() {
	select {
	case m.clickedCh <- struct{}{}:
	default:
	}
}

// TestManagerWithMockFactory tests the Manager using a mock factory
func TestManagerWithMockFactory(t *testing.T) {
	factory := NewMockFactory()
	manager := NewManager("1.0.0", factory)

	t.Run("initial setup", func(t *testing.T) {
		// Check that all items were created
		assert.NotNil(t, manager.Items.Version)
		assert.NotNil(t, manager.Items.MigrateMDM)
		assert.NotNil(t, manager.Items.MyDevice)
		assert.NotNil(t, manager.Items.HostOffline)
		assert.NotNil(t, manager.Items.SelfService)
		assert.NotNil(t, manager.Items.Transparency)

		// Check that correct number of separators were added
		assert.Equal(t, 2, factory.Separators)

		// Check that correct number of items were created
		assert.Equal(t, 6, len(factory.Items)) // Version, MigrateMDM, MyDevice, 1x HostOffline, SelfService, Transparency
	})

	t.Run("set connecting state", func(t *testing.T) {
		manager.SetConnecting()

		// Check MyDevice state
		myDevice := manager.Items.MyDevice.(*MockMenuItem)
		assert.Equal(t, "Connecting...", myDevice.Title)
		assert.True(t, myDevice.Visible)
		assert.False(t, myDevice.Enabled)

		// Check other items are hidden/disabled
		transparency := manager.Items.Transparency.(*MockMenuItem)
		assert.False(t, transparency.Enabled)

		selfService := manager.Items.SelfService.(*MockMenuItem)
		assert.False(t, selfService.Visible)
		assert.False(t, selfService.Enabled)

		migrateMDM := manager.Items.MigrateMDM.(*MockMenuItem)
		assert.False(t, migrateMDM.Visible)
		assert.False(t, migrateMDM.Enabled)
	})

	t.Run("set connected state", func(t *testing.T) {
		summary := &fleet.DesktopSummary{
			SelfService: ptr.Bool(true),
		}
		manager.SetConnected(summary, false)

		// Check MyDevice state
		myDevice := manager.Items.MyDevice.(*MockMenuItem)
		assert.Equal(t, "My device", myDevice.Title)
		assert.True(t, myDevice.Visible)
		assert.True(t, myDevice.Enabled)

		// Check transparency is enabled
		transparency := manager.Items.Transparency.(*MockMenuItem)
		assert.True(t, transparency.Enabled)
		assert.True(t, transparency.Visible)

		// Check self-service is shown (not free tier)
		selfService := manager.Items.SelfService.(*MockMenuItem)
		assert.True(t, selfService.Visible)
		assert.True(t, selfService.Enabled)
	})

	t.Run("set offline state", func(t *testing.T) {
		// First, set connected state with self-service enabled
		summary := &fleet.DesktopSummary{
			SelfService: ptr.Bool(true),
		}
		manager.SetConnected(summary, false)

		// Verify self-service is enabled when connected
		selfService := manager.Items.SelfService.(*MockMenuItem)
		assert.True(t, selfService.Enabled, "Self-service should be enabled when connected")
		assert.True(t, selfService.Visible, "Self-service should be visible when connected")

		// Verify offline indicator is not displayed after connecting
		assert.False(t, manager.IsOfflineIndicatorDisplayed(), "Offline indicator should not be displayed when connected")

		// Now set offline state
		manager.SetOffline()

		// Check MyDevice is hidden
		myDevice := manager.Items.MyDevice.(*MockMenuItem)
		assert.False(t, myDevice.Visible)

		// Check transparency is disabled and hidden
		transparency := manager.Items.Transparency.(*MockMenuItem)
		assert.False(t, transparency.Enabled)
		assert.False(t, transparency.Visible)

		// Check self-service is disabled when offline
		assert.False(t, selfService.Enabled, "Self-service should be disabled when offline")
		assert.False(t, selfService.Visible, "Self-service should be hidden when offline")

		// Check offline warning is shown
		offlineItem := manager.Items.HostOffline.(*MockMenuItem)
		assert.True(t, offlineItem.Visible)

		// Check offline indicator is displayed
		assert.True(t, manager.IsOfflineIndicatorDisplayed(), "Offline indicator should be displayed when offline")
	})

	t.Run("update failing policies", func(t *testing.T) {
		// Test with failing policies
		failingCount := uint(3)
		manager.UpdateFailingPolicies(&failingCount)

		myDevice := manager.Items.MyDevice.(*MockMenuItem)
		assert.Contains(t, myDevice.Title, "3")

		// Test with no failing policies
		failingCount = uint(0)
		manager.UpdateFailingPolicies(&failingCount)
		assert.Contains(t, myDevice.Title, "My device")
	})

	t.Run("MDM migrator visibility", func(t *testing.T) {
		// Initially should be hidden
		assert.False(t, manager.GetMDMMigratorVisibility())

		// Show MDM migrator
		manager.SetMDMMigratorVisibility(true)
		assert.True(t, manager.GetMDMMigratorVisibility())

		migrateMDM := manager.Items.MigrateMDM.(*MockMenuItem)
		assert.True(t, migrateMDM.Visible)
		assert.True(t, migrateMDM.Enabled)

		// Hide MDM migrator
		manager.SetMDMMigratorVisibility(false)
		assert.False(t, manager.GetMDMMigratorVisibility())
		assert.False(t, migrateMDM.Visible)
		assert.False(t, migrateMDM.Enabled)
	})

	t.Run("offline indicator display state", func(t *testing.T) {
		// Create a fresh manager for this test
		testFactory := NewMockFactory()
		testManager := NewManager("1.0.0", testFactory)

		// Initially should not be displayed
		assert.False(t, testManager.IsOfflineIndicatorDisplayed())

		// Set offline indicator displayed
		testManager.SetOfflineIndicatorDisplayed(true)
		assert.True(t, testManager.IsOfflineIndicatorDisplayed())

		// Clear offline indicator displayed
		testManager.SetOfflineIndicatorDisplayed(false)
		assert.False(t, testManager.IsOfflineIndicatorDisplayed())
	})
}