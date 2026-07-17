import AppKit

@main
struct FleetDesktopMain {
    static func main() {
        let app = NSApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        app.run()
    }
}

/// Pure AppKit app delegate — no SwiftUI status window.
/// Runs FleetService on launch and opens the browser window directly.
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let fleetService = FleetService()

    /// True if another instance of this app was already running when we launched.
    /// A secondary instance forwards any fleet:// URL to the primary and exits,
    /// so macOS never shows a second Dock icon.
    private var isSecondaryInstance = false

    /// Distributed notifications used to hand work from a short-lived duplicate
    /// instance to the already-running primary before the duplicate exits.
    private static let forwardURLNotification = Notification.Name("com.fleetdm.fleet-desktop.openURL")
    private static let forwardReopenNotification = Notification.Name("com.fleetdm.fleet-desktop.reopen")

    func applicationWillFinishLaunching(_ notification: Notification) {
        // Register the fleet:// handler before the system delivers the launch URL.
        // macOS delivers the Apple Event between willFinishLaunching and
        // didFinishLaunching, so registering here captures it even for a duplicate
        // instance that exists only to forward the URL to the primary.
        NSAppleEventManager.shared().setEventHandler(
            self,
            andSelector: #selector(handleURLEvent(_:withReply:)),
            forEventClass: AEEventClass(kInternetEventClass),
            andEventID: AEEventID(kAEGetURL)
        )

        // Single-instance guard. macOS normally coalesces launches of the same
        // bundle, but that breaks when the binary is exec'd directly, the app runs
        // from a translocated path, or multiple bundle copies are registered — any
        // of which leaves duplicate Dock icons. If a primary is already running we
        // become a secondary: hand off (URL or reopen) and exit before showing UI.
        if isAlreadyRunningElsewhere() {
            isSecondaryInstance = true
            return
        }

        // Primary: listen for hand-offs from any future duplicate instance.
        let dnc = DistributedNotificationCenter.default()
        dnc.addObserver(self, selector: #selector(handleForwardedURL(_:)),
                        name: Self.forwardURLNotification, object: nil)
        dnc.addObserver(self, selector: #selector(handleForwardedReopen(_:)),
                        name: Self.forwardReopenNotification, object: nil)
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        // A secondary instance with no fleet:// URL (a plain relaunch): tell the
        // primary to reopen its window, then exit so we leave no Dock icon behind.
        if isSecondaryInstance {
            forwardReopenToPrimary()
            activatePrimaryAndTerminate()
        }
        setupMainMenu()
        fleetService.run()
    }

    func applicationDidBecomeActive(_ notification: Notification) {
        fleetService.onApplicationDidBecomeActive()
    }

    @objc private func handleURLEvent(_ event: NSAppleEventDescriptor, withReply reply: NSAppleEventDescriptor) {
        guard let urlString = event.paramDescriptor(forKeyword: AEKeyword(keyDirectObject))?.stringValue,
              let url = URL(string: urlString),
              url.scheme?.lowercased() == "fleet" else {
            return
        }
        // Secondary instance: forward the deep link to the primary and exit so the
        // duplicate never materializes as its own Dock icon.
        if isSecondaryInstance {
            forwardURLToPrimary(url)
            activatePrimaryAndTerminate()
        }
        fleetService.handleFleetURL(url)
    }

    // MARK: - Single-Instance Handoff

    /// Whether another instance of this bundle (with a lower PID — the primary) is
    /// already running. Comparing PIDs makes the choice deterministic if two
    /// instances ever launch simultaneously: the lowest PID stays, the rest exit.
    private func isAlreadyRunningElsewhere() -> Bool {
        guard let bundleID = Bundle.main.bundleIdentifier else { return false }
        let mine = NSRunningApplication.current.processIdentifier
        return NSRunningApplication.runningApplications(withBundleIdentifier: bundleID)
            .contains { $0.processIdentifier < mine }
    }

    /// Hands a fleet:// deep link to the primary instance over a distributed
    /// notification. `deliverImmediately` flushes it before we exit(0).
    private func forwardURLToPrimary(_ url: URL) {
        DistributedNotificationCenter.default().postNotificationName(
            Self.forwardURLNotification,
            object: nil,
            userInfo: ["url": url.absoluteString],
            deliverImmediately: true
        )
    }

    /// Asks the primary instance to reopen its window (plain relaunch with no URL).
    private func forwardReopenToPrimary() {
        DistributedNotificationCenter.default().postNotificationName(
            Self.forwardReopenNotification,
            object: nil,
            userInfo: nil,
            deliverImmediately: true
        )
    }

    /// Bring the primary instance forward, then exit immediately. A secondary has
    /// created no windows or timers, so exit(0) is clean and skips any further
    /// delegate callbacks.
    private func activatePrimaryAndTerminate() -> Never {
        if let bundleID = Bundle.main.bundleIdentifier {
            let mine = NSRunningApplication.current.processIdentifier
            NSRunningApplication.runningApplications(withBundleIdentifier: bundleID)
                .filter { $0.processIdentifier != mine }
                .min(by: { $0.processIdentifier < $1.processIdentifier })?
                .activate(options: [.activateAllWindows])
        }
        exit(0)
    }

    @objc private func handleForwardedURL(_ notification: Notification) {
        guard let urlString = notification.userInfo?["url"] as? String,
              let url = URL(string: urlString) else { return }
        fleetService.handleFleetURL(url)
    }

    @objc private func handleForwardedReopen(_ notification: Notification) {
        fleetService.run()
    }

    // MARK: - Main Menu

    private func setupMainMenu() {
        let mainMenu = NSMenu()

        // App menu (Fleet Desktop)
        let appMenuItem = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(withTitle: "About Fleet Desktop", action: #selector(NSApplication.orderFrontStandardAboutPanel(_:)), keyEquivalent: "")
        appMenu.addItem(.separator())
        appMenu.addItem(withTitle: "Hide Fleet Desktop", action: #selector(NSApplication.hide(_:)), keyEquivalent: "h")
        let hideOthersItem = appMenu.addItem(withTitle: "Hide Others", action: #selector(NSApplication.hideOtherApplications(_:)), keyEquivalent: "h")
        hideOthersItem.keyEquivalentModifierMask = [.command, .option]
        appMenu.addItem(withTitle: "Show All", action: #selector(NSApplication.unhideAllApplications(_:)), keyEquivalent: "")
        appMenu.addItem(.separator())
        appMenu.addItem(withTitle: "Quit Fleet Desktop", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        appMenuItem.submenu = appMenu
        mainMenu.addItem(appMenuItem)

        // Edit menu (enables copy/paste/select-all in the web view)
        let editMenuItem = NSMenuItem()
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(withTitle: "Undo", action: NSSelectorFromString("undo:"), keyEquivalent: "z")
        editMenu.addItem(withTitle: "Redo", action: NSSelectorFromString("redo:"), keyEquivalent: "Z")
        editMenu.addItem(.separator())
        editMenu.addItem(withTitle: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        editMenuItem.submenu = editMenu
        mainMenu.addItem(editMenuItem)

        // View menu
        let viewMenuItem = NSMenuItem()
        let viewMenu = NSMenu(title: "View")
        viewMenu.addItem(withTitle: "Reload Page", action: #selector(reloadPage(_:)), keyEquivalent: "r")
        viewMenuItem.submenu = viewMenu
        mainMenu.addItem(viewMenuItem)

        // Window menu
        let windowMenuItem = NSMenuItem()
        let windowMenu = NSMenu(title: "Window")
        windowMenu.addItem(withTitle: "Minimize", action: #selector(NSWindow.performMiniaturize(_:)), keyEquivalent: "m")
        windowMenu.addItem(withTitle: "Close", action: #selector(NSWindow.performClose(_:)), keyEquivalent: "w")
        windowMenuItem.submenu = windowMenu
        mainMenu.addItem(windowMenuItem)

        NSApp.mainMenu = mainMenu
        NSApp.windowsMenu = windowMenu
    }

    @objc private func reloadPage(_ sender: Any?) {
        fleetService.reloadCurrentPage()
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return false
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        // Re-open the browser window when the user clicks the Dock icon
        if !flag {
            fleetService.run()
        }
        return true
    }
}
