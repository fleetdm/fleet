package menu

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// MaxPendingUpdates is the maximum number of app update items shown in the menu.
const MaxPendingUpdates = 10

// PendingUpdate represents an app that has an update available for self-service install.
type PendingUpdate struct {
	TitleID uint
	Name    string
	Version string
}

// isOpenSUSE detects if the system is running OpenSUSE
func isOpenSUSE() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check /etc/os-release for OpenSUSE identification
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		return strings.Contains(strings.ToLower(content), "opensuse")
	}

	return false
}

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
	Version       Item
	MigrateMDM    Item
	MyDevice      Item
	HostOffline   Item
	SelfService   Item
	UpdatesHeader Item
	UpdateItems   [MaxPendingUpdates]Item
	InstallAll    Item
	Transparency  Item
}

// Manager handles the state and behavior of the Fleet Desktop menu
type Manager struct {
	Items *Items
	// Track the current state of the MDM Migrate item so that on, e.g. token refreshes we can
	// immediately begin showing the migrator again if we were showing it prior.
	showMDMMigrator atomic.Bool
	// Track whether the offline indicator is currently displayed
	offlineIndicatorDisplayed atomic.Bool
	// pendingUpdates is the current list of app updates; protected by pendingUpdatesMu
	pendingUpdates   []PendingUpdate
	pendingUpdatesMu sync.RWMutex
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

	// Add updates section (shown when self-service software is available to install)
	items.UpdatesHeader = factory.AddMenuItem("Updates", "")
	items.UpdatesHeader.Disable()
	items.UpdatesHeader.Hide()
	for i := 0; i < MaxPendingUpdates; i++ {
		items.UpdateItems[i] = factory.AddMenuItem("", "")
		items.UpdateItems[i].Disable()
		items.UpdateItems[i].Hide()
	}
	items.InstallAll = factory.AddMenuItem("Install all", "")
	items.InstallAll.Disable()
	items.InstallAll.Hide()
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

	m.hideUpdatesSection()
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
	m.hideUpdatesSection()
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
		if runtime.GOOS == "windows" || isOpenSUSE() {
			// Windows and OpenSUSE don't reliably support color emoji in system tray
			if count == 1 {
				m.Items.MyDevice.SetTitle("My device (1 issue)")
			} else {
				m.Items.MyDevice.SetTitle(fmt.Sprintf("My device (%d issues)", count))
			}
		} else {
			m.Items.MyDevice.SetTitle(fmt.Sprintf("🔴 My device (%d)", count))
		}
	} else {
		if runtime.GOOS == "windows" || isOpenSUSE() {
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

// SetPendingUpdates shows or hides the Updates section based on the list of apps
// that need an update. When updates is nil (e.g. self-service disabled), the section
// is hidden. When updates is empty (0 updates), the header "Updates (0)" is shown
// as a disabled item. When updates is non-empty, the header, per-app items, and
// "Install all" (if more than one) are shown.
func (m *Manager) SetPendingUpdates(updates []PendingUpdate) {
	m.pendingUpdatesMu.Lock()
	if updates != nil && len(updates) > MaxPendingUpdates {
		updates = updates[:MaxPendingUpdates]
	}
	if updates == nil {
		m.pendingUpdates = nil
	} else {
		m.pendingUpdates = make([]PendingUpdate, len(updates))
		copy(m.pendingUpdates, updates)
	}
	m.pendingUpdatesMu.Unlock()

	if updates == nil {
		m.hideUpdatesSection()
		return
	}

	m.Items.UpdatesHeader.SetTitle(fmt.Sprintf("Updates (%d)", len(updates)))
	m.Items.UpdatesHeader.Show()

	for i := 0; i < MaxPendingUpdates; i++ {
		if i < len(updates) {
			title := updates[i].Name
			if updates[i].Version != "" {
				title = fmt.Sprintf("%s (%s) - Install", updates[i].Name, updates[i].Version)
			} else {
				title = fmt.Sprintf("%s - Install", updates[i].Name)
			}
			m.Items.UpdateItems[i].SetTitle(title)
			m.Items.UpdateItems[i].Enable()
			m.Items.UpdateItems[i].Show()
		} else {
			m.Items.UpdateItems[i].Hide()
		}
	}

	// Only show "Install all" when there is more than one update
	if len(updates) > 1 {
		m.Items.InstallAll.SetTitle("Install all")
		m.Items.InstallAll.Enable()
		m.Items.InstallAll.Show()
	} else {
		m.Items.InstallAll.Hide()
	}
}

// GetPendingUpdateTitleID returns the software title ID for the update at index i.
// It returns (0, false) if the index is out of range.
func (m *Manager) GetPendingUpdateTitleID(i int) (uint, bool) {
	m.pendingUpdatesMu.RLock()
	defer m.pendingUpdatesMu.RUnlock()
	if i < 0 || i >= len(m.pendingUpdates) {
		return 0, false
	}
	return m.pendingUpdates[i].TitleID, true
}

// hideUpdatesSection hides the Updates header, all update items, and Install all.
func (m *Manager) hideUpdatesSection() {
	m.Items.UpdatesHeader.Hide()
	for i := 0; i < MaxPendingUpdates; i++ {
		m.Items.UpdateItems[i].Hide()
	}
	m.Items.InstallAll.Hide()
}
