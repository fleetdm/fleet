package menu

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// Factory is the interface for creating menu items
type Factory interface {
	AddMenuItem(title string, tooltip string) Item
	AddSeparator()
}

// Item interface abstracts systray.MenuItem for testing
// Note: systray.MenuItem.ClickedCh is a field, not a method,
// so we need a wrapper to provide method-based access for testability
type Item interface {
	SetTitle(title string)
	Enable()
	Disable()
	Show()
	Hide()
	ClickedCh() <-chan struct{}
}

// Items holds all the menu items for the Fleet Desktop systray
type Items struct {
	Version      Item
	MigrateMDM   Item
	MyDevice     Item
	HostOffline  Item
	SelfService  Item
	Transparency Item
}

// Manager handles the state and behavior of the Fleet Desktop menu
type Manager struct {
	Items *Items
	// Track the current state of the MDM Migrate item so that on, e.g. token refreshes we can
	// immediately begin showing the migrator again if we were showing it prior.
	showMDMMigrator atomic.Bool
	// Track whether the offline indicator is currently displayed
	offlineIndicatorDisplayed atomic.Bool
}

// NewManager creates a new menu manager with initialized menu items
func NewManager(version string, factory Factory) *Manager {
	items := &Items{}

	// Add version item (always disabled)
	items.Version = factory.AddMenuItem(fmt.Sprintf("Fleet Desktop v%s", version), "")
	items.Version.Disable()
	factory.AddSeparator()

	// Add MDM migration item
	items.MigrateMDM = factory.AddMenuItem("Migrate to Fleet", "")
	items.MigrateMDM.Disable()
	items.MigrateMDM.Hide()

	// Add my device item
	items.MyDevice = factory.AddMenuItem("My device", "")
	items.MyDevice.Disable()
	items.MyDevice.Hide()

	// Add offline warning item
	items.HostOffline = factory.AddMenuItem("🛜🚫 Your computer is not connected to Fleet.", "")
	items.HostOffline.Disable()

	// Add self-service item
	items.SelfService = factory.AddMenuItem("Self-service", "")
	items.SelfService.Disable()
	items.SelfService.Hide()
	factory.AddSeparator()

	// Add transparency item
	items.Transparency = factory.AddMenuItem("About Fleet", "")
	items.Transparency.Disable()
	items.Transparency.Hide()

	m := &Manager{
		Items: items,
	}
	// Initialize atomic fields
	m.showMDMMigrator.Store(false)
	m.offlineIndicatorDisplayed.Store(false)
	return m
}

// SetConnecting sets the menu to the connecting state
func (m *Manager) SetConnecting() {
	log.Debug().Msg("displaying Connecting...")
	m.Items.MyDevice.SetTitle("Connecting...")
	m.Items.MyDevice.Show()
	m.Items.MyDevice.Disable()

	m.Items.Transparency.Disable()
	m.Items.SelfService.Disable()
	m.Items.SelfService.Hide()

	m.Items.MigrateMDM.Disable()
	if m.showMDMMigrator.Load() {
		m.Items.MigrateMDM.Show()
	} else {
		m.Items.MigrateMDM.Hide()
	}

	m.hideOfflineWarning()
}

// SetConnected sets the menu to the connected state
func (m *Manager) SetConnected(summary *fleet.DesktopSummary, isFreeTier bool) {
	m.Items.MyDevice.SetTitle("My device")
	m.Items.MyDevice.Show()
	m.Items.MyDevice.Enable()

	m.Items.Transparency.Enable()
	m.Items.Transparency.Show()

	m.hideOfflineWarning()
	m.offlineIndicatorDisplayed.Store(false)

	// Handle self-service visibility. Check for null for backward compatibility with an old Fleet server
	if isFreeTier || (summary.SelfService != nil && !*summary.SelfService) {
		m.Items.SelfService.Disable()
		m.Items.SelfService.Hide()
	} else {
		m.Items.SelfService.Enable()
		m.Items.SelfService.Show()
	}

	// Show MDM migrator if it was previously shown
	if m.showMDMMigrator.Load() {
		m.Items.MigrateMDM.Enable()
		m.Items.MigrateMDM.Show()
	}
}

// SetOffline sets the menu to the offline state
func (m *Manager) SetOffline() {
	m.Items.MyDevice.Hide()
	m.Items.SelfService.Disable()
	m.Items.SelfService.Hide()
	m.Items.Transparency.Disable()
	m.Items.Transparency.Hide()
	m.Items.MigrateMDM.Disable()
	m.Items.MigrateMDM.Hide()
	m.showOfflineWarning()
	m.offlineIndicatorDisplayed.Store(true)
}

// UpdateFailingPolicies updates the my device item based on failing policies count
func (m *Manager) UpdateFailingPolicies(failingPolicies *uint) {
	count := 0
	if failingPolicies != nil {
		count = int(*failingPolicies) // nolint:gosec // dismiss G115
	}

	if count > 0 {
		if runtime.GOOS == "windows" {
			// Windows doesn't support color emoji in system tray
			if count == 1 {
				m.Items.MyDevice.SetTitle("My device (1 issue)")
			} else {
				m.Items.MyDevice.SetTitle(fmt.Sprintf("My device (%d issues)", count))
			}
		} else {
			m.Items.MyDevice.SetTitle(fmt.Sprintf("🔴 My device (%d)", count))
		}
	} else {
		if runtime.GOOS == "windows" {
			m.Items.MyDevice.SetTitle("My device")
		} else {
			m.Items.MyDevice.SetTitle("🟢 My device")
		}
	}
}

// SetMDMMigratorVisibility controls the visibility of the MDM migration menu item
func (m *Manager) SetMDMMigratorVisibility(show bool) {
	m.showMDMMigrator.Store(show)
	if show {
		m.Items.MigrateMDM.Enable()
		m.Items.MigrateMDM.Show()
	} else {
		m.Items.MigrateMDM.Disable()
		m.Items.MigrateMDM.Hide()
	}
}

// GetMDMMigratorVisibility returns whether the MDM migrator is currently shown
func (m *Manager) GetMDMMigratorVisibility() bool {
	return m.showMDMMigrator.Load()
}

// IsOfflineIndicatorDisplayed returns whether the offline indicator is currently displayed
func (m *Manager) IsOfflineIndicatorDisplayed() bool {
	return m.offlineIndicatorDisplayed.Load()
}

// SetOfflineIndicatorDisplayed sets the offline indicator display state
func (m *Manager) SetOfflineIndicatorDisplayed(displayed bool) {
	m.offlineIndicatorDisplayed.Store(displayed)
}

// showOfflineWarning displays the offline warning item
func (m *Manager) showOfflineWarning() {
	m.Items.HostOffline.Show()
}

// hideOfflineWarning hides the offline warning item
func (m *Manager) hideOfflineWarning() {
	m.Items.HostOffline.Hide()
}
