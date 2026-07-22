import AppKit
import WebKit

/// A standalone browser window with an embedded WKWebView.
/// Scoped to the Fleet server — external links open in the default browser.
///
/// Supports preloading: call `preload(url:)` at app launch to start loading the page
/// in the background, then call `show()` to display the window instantly.
/// The WebView is kept alive when the window is closed, so reopening is instant.
final class BrowserWindow: NSObject, NSWindowDelegate {
    private var window: NSWindow?
    private var webView: WKWebView?
    private var container: NSView?
    private var fleetHost: String?
    private var homeURL: URL?
    private var loadingOverlay: NSView?
    private var pageLoaded = false

    /// JavaScript to run on the next `didFinish` navigation. Consumed once.
    /// Used by `fleet://update_all` to click the in-page "Update all" button.
    private var pendingPostLoadJS: String?

    /// Host of the external IdP page an SSO/auth flow is currently on. Non-nil
    /// while a flow is in progress; external redirects are kept in the WebView
    /// so the full redirect chain completes in-app, but navigation is restricted
    /// to this host (hops to a new host are only allowed via server redirects
    /// or form submissions).
    private var ssoHost: String?

    /// When the current SSO flow started. Flows expire after `ssoFlowTimeout`
    /// so the chrome-less WebView can't render external sites indefinitely.
    private var ssoFlowStartedAt: Date?

    /// Whether an SSO/auth flow is in progress.
    private var ssoFlowActive: Bool { ssoHost != nil }

    /// True when an SSO flow has been running longer than `ssoFlowTimeout`.
    private var ssoFlowExpired: Bool {
        guard let started = ssoFlowStartedAt else { return false }
        return Date().timeIntervalSince(started) > Self.ssoFlowTimeout
    }

    /// How long an SSO flow may run before external navigation is cut off.
    private static let ssoFlowTimeout: TimeInterval = 10 * 60

    /// The window title used throughout the app.
    static let windowTitle = "Fleet Desktop"

    /// File extensions that should be downloaded rather than displayed.
    private static let downloadableExtensions: Set<String> = [
        "mobileconfig", "pkg", "dmg", "zip", "tar", "gz", "pdf"
    ]

    /// MIME types that should be downloaded rather than displayed.
    private static let downloadableMIMETypes: Set<String> = [
        "application/x-apple-aspen-config",
        "application/octet-stream",
        "application/zip",
        "application/x-tar",
        "application/gzip",
        "application/pdf",
        "application/vnd.apple.installer+xml",
    ]

    /// URL schemes that are safe to open externally.
    private static let allowedExternalSchemes: Set<String> = ["https", "http", "mailto"]

    /// Called when a navigation error occurs (e.g., expired token returns 401/403)
    /// or when the page content indicates an error (e.g., "Something went wrong").
    var onNavigationError: (() -> Void)?

    /// Called when the window is closed (so the timer can be paused).
    var onWindowClose: (() -> Void)?

    /// Called when the window is shown (so the timer can be resumed).
    var onWindowShow: (() -> Void)?

    /// Preload the WebView and start loading the URL without showing a window.
    /// Call `show()` later to display the window.
    func preload(url: URL) {
        fleetHost = url.host?.lowercased()
        homeURL = url

        // Configure WKWebView — non-persistent data store so no cookies/cache persist
        let config = WKWebViewConfiguration()
        config.websiteDataStore = .nonPersistent()
        let wv = WKWebView(frame: .zero, configuration: config)
        wv.navigationDelegate = self
        wv.uiDelegate = self
        wv.translatesAutoresizingMaskIntoConstraints = false
        webView = wv

        let cont = NSView()
        cont.addSubview(wv)
        container = cont

        NSLayoutConstraint.activate([
            wv.topAnchor.constraint(equalTo: cont.topAnchor),
            wv.leadingAnchor.constraint(equalTo: cont.leadingAnchor),
            wv.trailingAnchor.constraint(equalTo: cont.trailingAnchor),
            wv.bottomAnchor.constraint(equalTo: cont.bottomAnchor),
        ])

        wv.load(URLRequest(url: url))
    }

    /// Show the browser window. If the page hasn't finished loading yet,
    /// a loading overlay with the Fleet logo is displayed until it does.
    func show(title: String = BrowserWindow.windowTitle) {
        guard webView != nil, let container = container else { return }

        // If window already exists, just bring it forward
        if let win = window {
            win.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        // Add loading overlay if page isn't loaded yet
        if !pageLoaded {
            addLoadingOverlay()
        }

        // Default window size — centered on screen
        let screenFrame = NSScreen.main?.visibleFrame ?? NSRect(x: 0, y: 0, width: 1200, height: 800)
        let windowWidth: CGFloat = min(1100, screenFrame.width * 0.8)
        let windowHeight: CGFloat = min(750, screenFrame.height * 0.8)
        let windowRect = NSRect(
            x: screenFrame.midX - windowWidth / 2,
            y: screenFrame.midY - windowHeight / 2,
            width: windowWidth,
            height: windowHeight
        )

        let win = NSWindow(
            contentRect: windowRect,
            styleMask: [.titled, .closable, .resizable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        win.title = title
        win.isReleasedWhenClosed = false
        win.tabbingMode = .disallowed
        win.representedURL = nil
        win.standardWindowButton(.documentIconButton)?.isHidden = true
        win.contentView = container
        win.delegate = self
        win.minSize = NSSize(width: 480, height: 360)

        // Restore previous window position and size (persisted by macOS automatically)
        win.setFrameAutosaveName("FleetDesktopMainWindow")

        centerTitleTextField(in: win)

        win.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        self.window = win

        // Clear any web element focus so nothing appears selected on open
        webView?.evaluateJavaScript("document.activeElement?.blur()", completionHandler: nil)

        onWindowShow?()
    }

    /// Whether the WebView has been created (preloaded or opened).
    var isAvailable: Bool {
        return webView != nil
    }

    /// Whether the browser window exists and is on-screen (not used for preloaded-only state).
    var isWindowVisible: Bool {
        window.map { $0.isVisible } ?? false
    }

    /// Reload the current page in the web view (e.g., Cmd+R).
    func reloadCurrent() {
        webView?.reload()
    }

    /// Navigate the existing web view to a new URL (e.g., after token refresh).
    func reload(url: URL) {
        fleetHost = url.host?.lowercased()
        homeURL = url
        webView?.load(URLRequest(url: url))
    }

    /// Queue JavaScript to run once, the next time a navigation finishes loading.
    /// Set this *before* calling `preload(url:)` or `reload(url:)`.
    func runOnNextLoad(_ js: String) {
        pendingPostLoadJS = js
    }

    // MARK: - Loading Overlay

    private func addLoadingOverlay() {
        guard loadingOverlay == nil, let container = container else { return }

        let overlay = LoadingOverlayView()
        overlay.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(overlay)

        NSLayoutConstraint.activate([
            overlay.topAnchor.constraint(equalTo: container.topAnchor),
            overlay.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            overlay.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            overlay.bottomAnchor.constraint(equalTo: container.bottomAnchor),
        ])

        // Fleet logo centered in overlay
        let iconView = NSImageView()
        iconView.translatesAutoresizingMaskIntoConstraints = false
        if let logoPath = Bundle.main.path(forResource: "fleet-logo", ofType: "png"),
           let logo = NSImage(contentsOfFile: logoPath) {
            iconView.image = logo
        } else {
            iconView.image = NSApp.applicationIconImage
        }
        iconView.imageScaling = .scaleProportionallyUpOrDown
        overlay.addSubview(iconView)

        // Spinner below the icon
        let spinner = NSProgressIndicator()
        spinner.translatesAutoresizingMaskIntoConstraints = false
        spinner.style = .spinning
        spinner.controlSize = .regular
        spinner.startAnimation(nil)
        overlay.addSubview(spinner)

        NSLayoutConstraint.activate([
            iconView.centerXAnchor.constraint(equalTo: overlay.centerXAnchor),
            iconView.centerYAnchor.constraint(equalTo: overlay.centerYAnchor, constant: -20),
            iconView.widthAnchor.constraint(equalToConstant: 64),
            iconView.heightAnchor.constraint(equalToConstant: 64),
            spinner.centerXAnchor.constraint(equalTo: overlay.centerXAnchor),
            spinner.topAnchor.constraint(equalTo: iconView.bottomAnchor, constant: 16),
        ])

        self.loadingOverlay = overlay
    }

    private func dismissLoadingOverlay() {
        guard let overlay = loadingOverlay else { return }
        NSAnimationContext.runAnimationGroup({ context in
            context.duration = 0.3
            overlay.animator().alphaValue = 0
        }, completionHandler: { [weak self] in
            overlay.removeFromSuperview()
            self?.loadingOverlay = nil
        })
    }

    // MARK: - SSO Flow Detection

    /// Resets SSO state. Called on window close, navigation errors, and
    /// when navigation returns to the Fleet host from an SSO flow.
    private func resetSSOFlow() {
        ssoHost = nil
        ssoFlowStartedAt = nil
    }

    /// Returns the WebView to the Fleet device page. Used to recover from a
    /// stranded state (expired/abandoned SSO flow on an external page).
    private func navigateHome() {
        guard let homeURL = homeURL else { return }
        webView?.load(URLRequest(url: homeURL))
    }

    // MARK: - External URL Safety

    /// Opens a URL externally only if it uses a safe scheme (https, http, mailto).
    private func openExternalURL(_ url: URL) {
        guard let scheme = url.scheme?.lowercased(),
              Self.allowedExternalSchemes.contains(scheme) else {
            return
        }
        NSWorkspace.shared.open(url)
    }

    // MARK: - Title Centering

    private func centerTitleTextField(in window: NSWindow) {
        guard let titlebarView = window.standardWindowButton(.closeButton)?.superview else { return }

        for subview in titlebarView.subviews {
            if let textField = subview as? NSTextField {
                textField.translatesAutoresizingMaskIntoConstraints = false
                NSLayoutConstraint.activate([
                    textField.centerXAnchor.constraint(equalTo: titlebarView.centerXAnchor),
                    textField.centerYAnchor.constraint(equalTo: titlebarView.centerYAnchor),
                ])
            } else if !(subview is NSButton) {
                subview.isHidden = true
            }
        }
    }

    // MARK: - NSWindowDelegate

    func windowWillClose(_ notification: Notification) {
        // Keep the WebView alive — just detach the window
        window = nil
        loadingOverlay?.removeFromSuperview()
        loadingOverlay = nil
        pendingPostLoadJS = nil
        resetSSOFlow()
        onWindowClose?()
    }
}

// MARK: - WKNavigationDelegate

extension BrowserWindow: WKNavigationDelegate {
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        pageLoaded = true
        window?.title = Self.windowTitle
        dismissLoadingOverlay()

        // If an SSO flow was active and we've finished loading a Fleet-host page,
        // the SSO callback is complete — reset the flow.
        if ssoFlowActive, webView.url?.host?.lowercased() == fleetHost {
            resetSSOFlow()
        }

        // Check if the page content indicates an error (Fleet returns 200 with error HTML
        // when the token is expired, rather than a 401/403 status code)
        checkPageForErrors(webView)

        // Only run queued JS on Fleet-host pages — avoids injecting into IdP
        // pages during SSO redirects and avoids consuming the slot on an
        // intermediate redirect before the real target finishes loading.
        if let js = pendingPostLoadJS, webView.url?.host?.lowercased() == fleetHost {
            pendingPostLoadJS = nil
            webView.evaluateJavaScript(js, completionHandler: nil)
        }
    }

    /// Inspects the page DOM for error indicators that suggest the device token has expired.
    /// Fleet returns HTTP 200 with specific error HTML when tokens expire, so we check for
    /// a combination of error phrases to reduce false positives from legitimate page content.
    private func checkPageForErrors(_ webView: WKWebView) {
        let js = """
        (function() {
            var body = document.body ? document.body.innerText : '';
            var errors = 0;
            if (body.indexOf('Something went wrong') !== -1) errors++;
            if (body.indexOf('Error loading software') !== -1) errors++;
            if (body.indexOf('Please contact your IT admin') !== -1) errors++;
            return errors >= 2 ? 'error' : 'ok';
        })();
        """
        webView.evaluateJavaScript(js) { [weak self] result, _ in
            if let status = result as? String, status == "error" {
                self?.onNavigationError?()
            }
        }
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        // Ignore cancelled navigations (e.g., user clicked a new link while loading)
        if (error as NSError).code == NSURLErrorCancelled { return }
        resetSSOFlow()
        onNavigationError?()
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        if (error as NSError).code == NSURLErrorCancelled { return }
        resetSSOFlow()
        onNavigationError?()
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationResponse: WKNavigationResponse,
        decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void
    ) {
        if let httpResponse = navigationResponse.response as? HTTPURLResponse,
           httpResponse.statusCode == 401 || httpResponse.statusCode == 403 {
            decisionHandler(.cancel)
            onNavigationError?()
            return
        }

        // Check if this response should be downloaded instead of displayed
        let shouldDownload: Bool = {
            let mimeType = navigationResponse.response.mimeType ?? ""
            let urlExtension = navigationResponse.response.url?.pathExtension.lowercased() ?? ""

            if Self.downloadableMIMETypes.contains(mimeType) { return true }
            if Self.downloadableExtensions.contains(urlExtension) { return true }
            if !navigationResponse.canShowMIMEType { return true }
            return false
        }()

        if shouldDownload {
            decisionHandler(.download)
            return
        }

        decisionHandler(.allow)
    }

    func webView(_ webView: WKWebView, navigationResponse: WKNavigationResponse, didBecome download: WKDownload) {
        download.delegate = self
    }

    func webView(_ webView: WKWebView, navigationAction: WKNavigationAction, didBecome download: WKDownload) {
        download.delegate = self
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        guard let requestURL = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }

        let requestHost = requestURL.host?.lowercased()

        // Always allow same-host and about: URLs
        if requestHost == fleetHost || requestURL.scheme == "about" {
            decisionHandler(.allow)
            return
        }

        // During an active SSO flow, keep external IdP redirects in the WebView
        // so the chain completes in-app — but only over HTTPS, only while the
        // flow is fresh, and only on the current IdP host. Hops to a *new*
        // external host are allowed via server redirects or form submissions
        // (multi-host IdP chains); link clicks to unrelated hosts open in the
        // default browser so the chrome-less WebView can't be steered to
        // arbitrary sites.
        if ssoFlowActive {
            guard requestURL.scheme?.lowercased() == "https", !ssoFlowExpired else {
                // Flow over (expired or degraded to non-HTTPS). Don't just cancel —
                // that would strand the WebView on the IdP page; return home.
                resetSSOFlow()
                decisionHandler(.cancel)
                navigateHome()
                return
            }
            if requestHost == ssoHost {
                decisionHandler(.allow)
            } else if navigationAction.navigationType == .other || navigationAction.navigationType == .formSubmitted {
                ssoHost = requestHost
                decisionHandler(.allow)
            } else {
                openExternalURL(requestURL)
                decisionHandler(.cancel)
            }
            return
        }

        // Detect SSO: if the Fleet server redirected us to an external host
        // (server redirect or form submission from Fleet page), start SSO flow.
        // This covers all SSO scenarios: MDM enrollment, IdP login, etc.
        if navigationAction.navigationType == .other || navigationAction.navigationType == .formSubmitted {
            if navigationAction.sourceFrame.request.url?.host?.lowercased() == fleetHost,
               requestURL.scheme?.lowercased() == "https" {
                ssoHost = requestHost
                ssoFlowStartedAt = Date()
                decisionHandler(.allow)
                return
            }
        }

        // External links — open in default browser (scheme-validated). But if
        // the WebView is stranded on an external page with no active flow (an
        // expired or abandoned SSO), navigate home instead — otherwise every
        // scripted retry on the stranded page would pop another browser tab.
        decisionHandler(.cancel)
        if webView.url?.host?.lowercased() == fleetHost {
            openExternalURL(requestURL)
        } else {
            navigateHome()
        }
    }
}

// MARK: - WKUIDelegate

extension BrowserWindow: WKUIDelegate {
    /// Handle links that request a new window (target="_blank", window.open, etc.).
    /// Same-host links are loaded in the current WebView; external links open in the default browser.
    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if let url = navigationAction.request.url {
            let host = url.host?.lowercased()
            if host == fleetHost || (ssoFlowActive && host == ssoHost && !ssoFlowExpired) {
                webView.load(URLRequest(url: url))
            } else {
                openExternalURL(url)
            }
        }
        return nil
    }
}

// MARK: - WKDownloadDelegate

extension BrowserWindow: WKDownloadDelegate {
    func download(
        _ download: WKDownload,
        decideDestinationUsing response: URLResponse,
        suggestedFilename: String,
        completionHandler: @escaping (URL?) -> Void
    ) {
        let downloadsDir = FileManager.default.urls(for: .downloadsDirectory, in: .userDomainMask).first!
        // The suggested name is server-supplied — keep only the final path
        // component so the file always lands directly in Downloads.
        var safeFilename = (suggestedFilename as NSString).lastPathComponent
        if safeFilename.isEmpty || safeFilename == "." || safeFilename == ".." { safeFilename = "download" }
        var destination = downloadsDir.appendingPathComponent(safeFilename)

        // Avoid overwriting existing files — append a number if needed (max 999)
        var counter = 1
        let baseName = destination.deletingPathExtension().lastPathComponent
        let ext = destination.pathExtension
        while FileManager.default.fileExists(atPath: destination.path), counter < 1000 {
            let newName = ext.isEmpty ? "\(baseName) (\(counter))" : "\(baseName) (\(counter)).\(ext)"
            destination = downloadsDir.appendingPathComponent(newName)
            counter += 1
        }

        completionHandler(destination)
    }

    func downloadDidFinish(_ download: WKDownload) {
        guard let url = download.progress.fileURL else { return }

        // Only auto-open .mobileconfig files (MDM enrollment profiles).
        // All other file types are saved to Downloads without opening,
        // to avoid automatically executing potentially unsafe files.
        if url.pathExtension.lowercased() == "mobileconfig" {
            NSWorkspace.shared.open(url)

            // Navigate back to the Fleet self-service homepage
            navigateHome()
        }
    }

    func download(_ download: WKDownload, didFailWithError error: Error, resumeData: Data?) {
        NSLog("Fleet Desktop: Download failed: %@", error.localizedDescription)
    }
}

// MARK: - Loading Overlay

/// Draws the window background color, automatically adapting when the
/// user switches between dark and light mode.
private final class LoadingOverlayView: NSView {
    override var wantsUpdateLayer: Bool { true }

    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        wantsLayer = true
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        wantsLayer = true
    }

    override func updateLayer() {
        layer?.backgroundColor = NSColor.windowBackgroundColor.cgColor
    }
}
