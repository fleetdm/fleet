import AppKit
import WebKit

/// Shows Fleet's MDM enrollment page (`<server>/enroll`) in a sheet over the device page.
///
/// "Turn on MDM" opens the enrollment page in a new tab. The app has no tabs, so loading
/// it in the main WebView replaced the device page with one that has no way back. In a
/// sheet the device page stays loaded underneath and Close always returns to it.
///
/// The sheet is its own navigation delegate, with enrollment-only rules, so the main
/// WebView's navigation and SSO handling never see its traffic.
final class EnrollmentSheet: NSObject {
    let webView: WKWebView

    /// Called once the sheet has been dismissed.
    var onClose: (() -> Void)?

    /// Called when a navigation in the sheet becomes a download, so the owner can
    /// set its delegate to save and open the enrollment profile.
    var onDownload: ((WKDownload) -> Void)?

    private let fleetHost: String
    private let panel: NSWindow
    private let buttonBar = NSView()
    private weak var parent: NSWindow?
    private var contentHeight: CGFloat?
    private var closed = false

    private static let preferredContentSize = NSSize(width: 980, height: 640)
    private static let minimumContentSize = NSSize(width: 320, height: 200)

    /// Space kept between the sheet and the parent window's edges.
    private static let parentMargin: CGFloat = 40

    private static let contentHeightMessage = "enrollmentContentHeight"

    /// The enrollment page is laid out for a full browser window. At sheet height its
    /// later steps fall below the fold with no visible scrollbar, so users only see
    /// Download. These rules tighten its spacing so every step fits; they target the
    /// page's own classes, so if the page changes they stop matching and the sheet
    /// scrolls instead.
    private static let compactCSS = """
    #main-content { padding: 28px 32px; }
    .content-with-sidebar { gap: 40px; }
    .device-instructions-content { gap: 20px; }
    .device-instructions-content h1 { font-size: 24px; line-height: 1.3; }
    .device-instructions-content ol { gap: 14px; }
    .device-instructions-content li { gap: 10px; }
    """

    init(fleetHost: String, websiteDataStore: WKWebsiteDataStore) {
        self.fleetHost = fleetHost
        let config = WKWebViewConfiguration()
        config.websiteDataStore = websiteDataStore
        webView = WKWebView(frame: .zero, configuration: config)
        panel = NSWindow(
            contentRect: NSRect(origin: .zero, size: Self.preferredContentSize),
            styleMask: [.titled],
            backing: .buffered,
            defer: true
        )
        super.init()

        // An isolated world keeps the handler out of reach of page scripts, including
        // IdP pages shown during end-user authentication.
        let controller = config.userContentController
        controller.addUserScript(WKUserScript(
            source: Self.contentScript(fleetHost: fleetHost),
            injectionTime: .atDocumentEnd,
            forMainFrameOnly: true,
            in: .defaultClient
        ))
        controller.add(self, contentWorld: .defaultClient, name: Self.contentHeightMessage)

        webView.navigationDelegate = self
        webView.uiDelegate = self
        buildContent()
    }

    /// Attaches the sheet to `parent` and loads `request` in it.
    func present(on parent: NSWindow, request: URLRequest) {
        self.parent = parent
        fit()
        parent.beginSheet(panel) { [weak self] _ in
            self?.didEnd()
        }
        webView.load(request)
    }

    /// Dismisses the sheet. Safe to call more than once.
    func close() {
        guard !closed else { return }
        if let parent = panel.sheetParent {
            parent.endSheet(panel)
        } else {
            didEnd()
        }
    }

    /// Re-fits the sheet after the parent window is resized.
    func parentDidResize() {
        fit()
    }

    @objc private func closeClicked(_ sender: Any?) {
        close()
    }

    private func didEnd() {
        guard !closed else { return }
        closed = true
        webView.stopLoading()
        // The content controller retains its message handlers, which would keep this
        // sheet alive.
        webView.configuration.userContentController.removeScriptMessageHandler(
            forName: Self.contentHeightMessage, contentWorld: .defaultClient)
        onClose?()
    }

    /// Opens a URL in the default browser if it uses a safe scheme.
    private func openExternalURL(_ url: URL) {
        guard let scheme = url.scheme?.lowercased(),
              BrowserWindow.allowedExternalSchemes.contains(scheme) else {
            return
        }
        NSWorkspace.shared.open(url)
    }

    // MARK: - Layout

    private func buildContent() {
        webView.translatesAutoresizingMaskIntoConstraints = false

        let separator = NSBox()
        separator.boxType = .separator
        separator.translatesAutoresizingMaskIntoConstraints = false

        let closeButton = NSButton(title: "Close", target: self, action: #selector(closeClicked(_:)))
        closeButton.bezelStyle = .rounded
        closeButton.keyEquivalent = "\u{1b}"
        closeButton.translatesAutoresizingMaskIntoConstraints = false

        buttonBar.translatesAutoresizingMaskIntoConstraints = false
        buttonBar.addSubview(separator)
        buttonBar.addSubview(closeButton)

        let content = NSView()
        content.addSubview(webView)
        content.addSubview(buttonBar)
        panel.contentView = content

        NSLayoutConstraint.activate([
            webView.topAnchor.constraint(equalTo: content.topAnchor),
            webView.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            webView.bottomAnchor.constraint(equalTo: buttonBar.topAnchor),

            buttonBar.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            buttonBar.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            buttonBar.bottomAnchor.constraint(equalTo: content.bottomAnchor),

            separator.topAnchor.constraint(equalTo: buttonBar.topAnchor),
            separator.leadingAnchor.constraint(equalTo: buttonBar.leadingAnchor),
            separator.trailingAnchor.constraint(equalTo: buttonBar.trailingAnchor),

            closeButton.topAnchor.constraint(equalTo: separator.bottomAnchor, constant: 12),
            closeButton.trailingAnchor.constraint(equalTo: buttonBar.trailingAnchor, constant: -20),
            closeButton.bottomAnchor.constraint(equalTo: buttonBar.bottomAnchor, constant: -12),
        ])
    }

    /// Sizes the sheet to show the whole enrollment page without scrolling. Before the
    /// page reports its height (or while an IdP page is showing), uses the preferred
    /// size within the parent window.
    private func fit() {
        guard let parent = parent else { return }
        let available = parent.contentLayoutRect.size
        let minimum = Self.minimumContentSize
        let width = min(Self.preferredContentSize.width, max(available.width - Self.parentMargin, minimum.width))

        let height: CGFloat
        if let contentHeight = contentHeight {
            // Every step should be visible, so the sheet may hang below a short parent
            // window, as sheets can, but it stays on screen.
            let parentTop = parent.convertToScreen(parent.contentLayoutRect).maxY
            let screenBottom = (parent.screen ?? NSScreen.main)?.visibleFrame.minY ?? 0
            let maxHeight = max(parentTop - screenBottom - Self.parentMargin, minimum.height)
            height = min(max(contentHeight + buttonBar.fittingSize.height, minimum.height), maxHeight)
        } else {
            height = min(Self.preferredContentSize.height, max(available.height - Self.parentMargin, minimum.height))
        }
        let size = NSSize(width: width, height: height)

        guard panel.sheetParent != nil else {
            panel.setContentSize(size)
            return
        }
        // Keep the sheet attached to the top of the parent and centered on it.
        let top = panel.frame.maxY
        panel.setContentSize(size)
        panel.setFrameTopLeftPoint(NSPoint(x: parent.frame.midX - panel.frame.width / 2, y: top))
    }

    // MARK: - Content script

    /// Compacts the enrollment page and reports its content height whenever it changes
    /// (fonts loading, switching between Personal and Company-owned). Only acts on
    /// Fleet's enrollment page: the sheet also shows IdP pages during end-user
    /// authentication.
    private static func contentScript(fleetHost: String) -> String {
        """
        (() => {
          if (location.hostname.toLowerCase() !== \(jsString(fleetHost)) ||
              !/\\/enroll\\/?$/.test(location.pathname)) {
            return;
          }
          const style = document.createElement("style");
          // Under Fleet's CSP (style-src 'self' 'nonce-…') an inline style needs the
          // page's nonce. The attribute is hidden once the page loads; the property
          // still returns it.
          const nonce = document.querySelector("style[nonce], script[nonce]")?.nonce;
          if (nonce) {
            style.nonce = nonce;
          }
          style.textContent = \(jsString(compactCSS));
          document.head.appendChild(style);
          const content = document.getElementById("main-content");
          if (!content) {
            return;
          }
          new ResizeObserver(() => {
            window.webkit.messageHandlers.\(contentHeightMessage).postMessage(
              Math.ceil(content.getBoundingClientRect().bottom + window.scrollY));
          }).observe(content);
        })();
        """
    }

    /// Encodes `value` as a JavaScript string literal.
    private static func jsString(_ value: String) -> String {
        guard let data = try? JSONEncoder().encode(value),
              let literal = String(data: data, encoding: .utf8) else {
            return "\"\""
        }
        return literal
    }
}

// MARK: - WKNavigationDelegate

extension EnrollmentSheet: WKNavigationDelegate {
    /// Fleet pages load in the sheet. When end-user authentication is required,
    /// `/enroll` redirects to the IdP, so HTTPS redirects, form posts, and scripted
    /// navigations may move between hosts. Link clicks stay on the current page's
    /// host; others open in the default browser. Close is always available, so
    /// nothing here can strand the user.
    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }
        let host = url.host?.lowercased()
        let scheme = url.scheme?.lowercased()

        if host == fleetHost || scheme == "about" {
            decisionHandler(.allow)
            return
        }

        // Okta FastPass during the IdP sign-in; see BrowserWindow.authenticatorSchemes.
        if let scheme = scheme, BrowserWindow.authenticatorSchemes.contains(scheme) {
            NSWorkspace.shared.open(url)
            decisionHandler(.cancel)
            return
        }

        let isHop = navigationAction.navigationType == .other || navigationAction.navigationType == .formSubmitted
        if scheme == "https", isHop || host == webView.url?.host?.lowercased() {
            decisionHandler(.allow)
            return
        }

        decisionHandler(.cancel)
        if navigationAction.navigationType == .linkActivated {
            openExternalURL(url)
        }
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationResponse: WKNavigationResponse,
        decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void
    ) {
        let response = navigationResponse.response
        let isProfile = response.mimeType == "application/x-apple-aspen-config"
            || response.url?.pathExtension.lowercased() == "mobileconfig"
        decisionHandler(isProfile || !navigationResponse.canShowMIMEType ? .download : .allow)
    }

    func webView(_ webView: WKWebView, navigationResponse: WKNavigationResponse, didBecome download: WKDownload) {
        onDownload?(download)
    }

    func webView(_ webView: WKWebView, navigationAction: WKNavigationAction, didBecome download: WKDownload) {
        onDownload?(download)
    }
}

// MARK: - WKUIDelegate

extension EnrollmentSheet: WKUIDelegate {
    /// New-window links: Fleet pages load in the sheet; anything else (such as the
    /// page's "Learn more" link) opens in the default browser.
    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if let url = navigationAction.request.url {
            if url.host?.lowercased() == fleetHost {
                webView.load(URLRequest(url: url))
            } else {
                openExternalURL(url)
            }
        }
        return nil
    }
}

// MARK: - WKScriptMessageHandler

extension EnrollmentSheet: WKScriptMessageHandler {
    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == Self.contentHeightMessage,
              message.frameInfo.isMainFrame,
              let height = (message.body as? NSNumber).map({ CGFloat(truncating: $0) }),
              height.isFinite, height > 0 else {
            return
        }
        contentHeight = height
        fit()
    }
}
