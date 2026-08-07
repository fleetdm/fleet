import AppKit
import WebKit
import os

/// How a toast ended.
enum ToastOutcome {
    /// The user activated the page's primary action.
    case primaryAction(id: String?)

    /// The user dismissed the toast.
    case dismissed(reason: String?)

    /// The display timeout expired with no interaction.
    case timedOut

    /// The page never reached first paint. Nothing was shown.
    case loadFailed(String)

    /// The page returned an HTTP error status on the main frame. Nothing was shown.
    case httpError(Int)

    /// The page loaded with HTTP 200 but rendered Fleet's error copy, which is what an
    /// expired device token looks like. Nothing was shown.
    case contentError(String)

    /// No display is attached, so there is nothing to draw on. Nothing was shown.
    case noDisplay
}

/// A borderless, floating toast anchored bottom-right, hosting a `WKWebView` on a
/// rounded card. `onDisplayed` fires when it reaches the screen, `onFinish` when it is
/// gone — or immediately, with a failure, if it never appeared. Each fires once.
final class ToastWindow: NSObject {
    // MARK: - Layout

    /// Matches the Figma update card.
    private static let cardSize = NSSize(width: 525, height: 318)

    /// Gap from the screen's working area.
    private static let margin: CGFloat = 16

    /// Matches the `--radius` the page's CSS draws its card with.
    private static let cornerRadius: CGFloat = 20

    /// Transparent padding so the drop shadow isn't clipped at the window edge. Must
    /// exceed the shadow's blur + offset.
    private static let shadowPadding: CGFloat = 70

    private static let animationDuration: TimeInterval = 0.35

    // MARK: - Timeouts

    /// Only covers a page that connects then never renders; real network failures are
    /// reported by the navigation delegate long before this.
    static let loadTimeout: TimeInterval = 30

    /// Safety net, not the expected way to close: that's the page's dismiss action.
    /// Without it a page that never sends `dismiss` leaves an undismissable window.
    static let displayTimeout: TimeInterval = 600

    /// Last resort, in case a wedged WebKit means neither timeout above fires.
    static let watchdogLimit: TimeInterval = loadTimeout + displayTimeout + 5

    /// Smallest card worth showing. The maximum is the screen's working height less
    /// margins, computed per-screen in `clampHeight(_:on:)`.
    private static let minHeight: CGFloat = 200

    private static let defaultHTTPSPort = 443

    // MARK: - Bridge

    /// Message handler name the page posts to.
    static let bridgeChannel = "fleetDesktop"

    /// Classifies the loaded document as empty, an error page, or usable.
    ///
    /// Error detection is copy-based and so inherently brittle — the durable fix is for
    /// the page to post an `error` action over the bridge, which is also handled.
    private static var contentProbe: String {
        """
        (function () {
          if (!document.body || document.body.children.length === 0) { return "empty"; }
          var text = document.body.innerText || "";
          return \(FleetErrorPage.matchesExpression) ? "error" : "ok";
        })();
        """
    }

    /// Reports the document's content height whenever it changes.
    ///
    /// Requires the page to let content determine its height: a page that sets
    /// `html, body { height: 100% }` always measures exactly the current viewport, so
    /// it can never grow. `scrollHeight` on the body is what a normal document flow
    /// reports.
    private static let autoSizeScript = """
        (function () {
          var last = 0;
          function report() {
            if (!document.body) { return; }
            var height = Math.ceil(document.body.scrollHeight);
            if (!height || Math.abs(height - last) < 2) { return; }
            last = height;
            window.webkit.messageHandlers.\(bridgeChannel).postMessage({
              v: 1, action: "resize", payload: { height: height }
            });
          }
          if (window.ResizeObserver && document.body) {
            new ResizeObserver(report).observe(document.body);
          }
          window.addEventListener("load", report);
          report();
        })();
        """

    // MARK: - State

    private let panel: NSPanel
    private let webView: WKWebView
    private let root: HaloView
    private let shadowView: ShadowBackingView
    private let card: NSView
    private let url: URL
    private let logger: Logger

    /// Host the page must post from, captured from the URL we load.
    private let expectedHost: String?

    /// Port that goes with `expectedHost`, defaulted to https's 443 when the URL omits
    /// it. Without this, a Fleet server on :8080 would also trust :9999 on that host.
    private let expectedPort: Int

    private var cardSize: NSSize

    /// Fires once, when the toast is on screen.
    var onDisplayed: (() -> Void)?

    /// Fires once, when the toast is gone or has failed before appearing.
    var onFinish: ((ToastOutcome) -> Void)?

    private var didDisplay = false
    private var didFinish = false

    private var loadDeadline: DispatchWorkItem?
    private var readyGrace: DispatchWorkItem?
    private var displayDeadline: DispatchWorkItem?

    /// The page signalled `ready`, or `didFinish` fired and the grace period lapsed.
    private var hasPainted = false

    init(url: URL, logger: Logger) {
        self.url = url
        self.logger = logger
        self.cardSize = Self.cardSize

        self.expectedHost = url.host?.lowercased()
        self.expectedPort = url.port ?? Self.defaultHTTPSPort

        let pad = Self.shadowPadding
        let cardRect = NSRect(origin: NSPoint(x: pad, y: pad), size: Self.cardSize)
        let fullRect = NSRect(
            x: 0, y: 0,
            width: Self.cardSize.width + 2 * pad,
            height: Self.cardSize.height + 2 * pad
        )

        // Non-activating so it doesn't steal focus. Joins all Spaces and survives
        // deactivation, so it follows the user rather than sticking to one desktop.
        panel = KeyablePanel(
            contentRect: fullRect,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.level = .floating
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = false // we draw a softer, alert-like shadow ourselves
        // Would otherwise be draggable by the invisible halo.
        panel.isMovableByWindowBackground = false
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        panel.hidesOnDeactivate = false

        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = .nonPersistent()
        let contentController = WKUserContentController()
        configuration.userContentController = contentController
        // Auto-size to content, so a page listing an arbitrary number of items isn't
        // clipped. Injected rather than left to the page, so it works without the page
        // implementing anything. The page can still post `resize` itself.
        contentController.addUserScript(
            WKUserScript(
                source: Self.autoSizeScript,
                injectionTime: .atDocumentEnd,
                forMainFrameOnly: true
            ))

        webView = WKWebView(frame: NSRect(origin: .zero, size: Self.cardSize), configuration: configuration)
        // Let the card's fill show through instead of the webview's opaque backdrop,
        // which is the wrong colour in dark mode. No public API for this; if Apple
        // removes the key the ObjC exception is uncatchable from Swift.
        webView.setValue(false, forKey: "drawsBackground")
        webView.wantsLayer = true
        webView.layer?.backgroundColor = NSColor.clear.cgColor
        // No mask here: the card already clips, and masking twice shaves its border.
        if #available(macOS 12.0, *) {
            // Avoids a black flash before the first paint composites.
            webView.underPageBackgroundColor = .clear
        }
        webView.autoresizingMask = [.width, .height]

        // Does NOT clip, so the shadow can render into the surrounding padding.
        root = HaloView(frame: fullRect)
        root.wantsLayer = true
        root.layer?.masksToBounds = false
        root.cardRect = cardRect

        // Separate view because a layer that clips its bounds clips its own shadow.
        // Filled with the card colour: the two rounded rects are coincident, so along
        // the curve the card is only partly opaque and any other colour shows through.
        shadowView = ShadowBackingView(frame: cardRect)
        shadowView.wantsLayer = true
        shadowView.layer?.cornerRadius = Self.cornerRadius
        shadowView.layer?.masksToBounds = false
        // NSShadow, not layer.shadow* — layer shadows don't render reliably here.
        let dropShadow = NSShadow()
        dropShadow.shadowColor = NSColor.black.withAlphaComponent(0.45)
        dropShadow.shadowBlurRadius = 24
        dropShadow.shadowOffset = NSSize(width: 0, height: -10)
        shadowView.shadow = dropShadow

        let solid = SolidCardView(frame: cardRect)
        solid.wantsLayer = true
        solid.layer?.cornerRadius = Self.cornerRadius
        solid.layer?.masksToBounds = true
        solid.addSubview(webView)
        card = solid

        super.init()

        // Weak proxy: WKUserContentController retains handlers strongly, and we own it
        // through the webview, so registering self directly is an unbreakable cycle.
        contentController.add(WeakScriptMessageProxy(self), name: Self.bridgeChannel)
        webView.navigationDelegate = self
        webView.uiDelegate = self

        root.addSubview(shadowView)
        root.addSubview(card)
        panel.contentView = root
    }

    deinit {
        teardown()
    }

    // MARK: - Presenting

    /// Loads the page and shows the toast once it has painted. Nothing is shown before
    /// first paint, and a toast the user never saw is never reported as displayed.
    func present() {
        webView.load(URLRequest(url: url))

        let deadline = DispatchWorkItem { [weak self] in
            guard let self = self, !self.hasPainted else { return }
            self.finish(.loadFailed("Page did not render within \(Int(Self.loadTimeout))s."))
        }
        loadDeadline = deadline
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.loadTimeout, execute: deadline)
    }

    /// Anchors bottom-right on the active screen and fades in.
    private func show() {
        guard !didDisplay, !didFinish else { return }

        // The screen under the cursor is where the user is actually looking.
        let mouse = NSEvent.mouseLocation
        let screen = NSScreen.screens.first(where: { NSMouseInRect(mouse, $0.frame, false) })
            ?? NSScreen.main
            ?? NSScreen.screens.first
        guard let screen = screen else {
            finish(.noDisplay)
            return
        }

        didDisplay = true
        cancelTimers()

        panel.setFrame(frame(on: screen), display: true)
        panel.alphaValue = 0
        panel.orderFrontRegardless()
        panel.makeFirstResponder(webView)
        // No NSApp.activate: the panel is non-activating so it doesn't steal focus.

        // Fade, not slide: a frame slide silently no-ops for borderless non-activating
        // panels, and animating the layer looks janky against the shadow.
        NSAnimationContext.runAnimationGroup { context in
            context.duration = Self.animationDuration
            context.timingFunction = CAMediaTimingFunction(name: .easeOut)
            panel.animator().alphaValue = 1
        }

        logger.log("toast displayed")
        onDisplayed?()

        armDisplayTimeout()
    }

    /// Anchored bottom-right. The origin is offset by `shadowPadding` so the visible
    /// card lands at the anchor, not the transparent padding. `visibleFrame` excludes
    /// the Dock.
    private func frame(on screen: NSScreen) -> NSRect {
        let pad = Self.shadowPadding
        let visible = screen.visibleFrame
        let fullSize = NSSize(
            width: cardSize.width + 2 * pad,
            height: cardSize.height + 2 * pad
        )
        let origin = NSPoint(
            x: visible.maxX - cardSize.width - Self.margin - pad,
            y: visible.minY + Self.margin - pad
        )
        return NSRect(origin: origin, size: fullSize)
    }

    private func armDisplayTimeout() {
        let deadline = DispatchWorkItem { [weak self] in
            self?.fadeOutAndFinish(.timedOut)
        }
        displayDeadline = deadline
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.displayTimeout, execute: deadline)
    }

    // MARK: - Finishing

    /// For outcomes the user caused, so the fade is visible before the process exits.
    private func fadeOutAndFinish(_ outcome: ToastOutcome) {
        guard !didFinish else { return }
        guard didDisplay else {
            finish(outcome)
            return
        }
        cancelTimers()

        NSAnimationContext.runAnimationGroup({ context in
            context.duration = Self.animationDuration
            context.timingFunction = CAMediaTimingFunction(name: .easeIn)
            panel.animator().alphaValue = 0
        }, completionHandler: { [weak self] in
            self?.panel.orderOut(nil)
            self?.finish(outcome)
        })
    }

    /// Single exit point, guarded so nothing reports twice.
    private func finish(_ outcome: ToastOutcome) {
        guard !didFinish else { return }
        didFinish = true
        cancelTimers()
        teardown()
        onFinish?(outcome)
    }

    private func cancelTimers() {
        loadDeadline?.cancel()
        readyGrace?.cancel()
        displayDeadline?.cancel()
        loadDeadline = nil
        readyGrace = nil
        displayDeadline = nil
    }

    /// Breaks the webview's references to this object.
    private func teardown() {
        webView.stopLoading()
        webView.navigationDelegate = nil
        webView.uiDelegate = nil
        webView.configuration.userContentController
            .removeScriptMessageHandler(forName: Self.bridgeChannel)
    }

    // MARK: - Resizing

    /// Applies a requested content height, keeping the toast anchored bottom-right.
    private func resize(toHeight requested: CGFloat) {
        let screen = panel.screen ?? NSScreen.main
        let height = clampHeight(requested, on: screen)
        // The observer fires again after we resize the webview, so ignoring an unchanged
        // height is what stops that becoming a feedback loop.
        guard abs(height - cardSize.height) > 0.5 else { return }

        logger.debug("resizing card to \(Int(height))pt (requested \(Int(requested)))")
        cardSize = NSSize(width: cardSize.width, height: height)

        let pad = Self.shadowPadding
        let cardRect = NSRect(origin: NSPoint(x: pad, y: pad), size: cardSize)
        shadowView.frame = cardRect
        card.frame = cardRect
        webView.frame = NSRect(origin: .zero, size: cardSize)
        root.cardRect = cardRect

        guard let screen = screen else { return }
        panel.setFrame(frame(on: screen), display: true)
    }

    /// Never taller than the screen's working area less margins, never smaller than
    /// `minHeight`.
    private func clampHeight(_ requested: CGFloat, on screen: NSScreen?) -> CGFloat {
        let available = (screen?.visibleFrame.height ?? Self.minHeight) - 2 * Self.margin
        let maxHeight = max(Self.minHeight, available)
        return min(max(requested, Self.minHeight), maxHeight)
    }

}

// MARK: - JS bridge

extension ToastWindow: WKScriptMessageHandler {
    /// Receives `window.webkit.messageHandlers.fleetDesktop.postMessage(...)`.
    /// Unknown actions and a missing `payload` are tolerated, so a newer page keeps
    /// working against an older binary.
    func userContentController(
        _ userContentController: WKUserContentController,
        didReceive message: WKScriptMessage
    ) {
        // Without the main-frame check, an iframe on the Fleet page (an embedded doc,
        // an OAuth widget) could claim the primary action or dismiss the toast, and
        // that flows straight into our exit code.
        guard message.frameInfo.isMainFrame else {
            logger.debug("bridge: dropped message from a subframe")
            return
        }
        guard isTrusted(message.frameInfo.securityOrigin) else {
            logger.warning("bridge: dropped message from an untrusted origin")
            return
        }
        guard let body = message.body as? [String: Any],
              let action = body["action"] as? String else {
            logger.debug("bridge: dropped malformed message")
            return
        }
        let payload = body["payload"] as? [String: Any]

        switch action {
        case "ready":
            markPainted()
        case "primary":
            fadeOutAndFinish(.primaryAction(id: payload?["id"] as? String))
        case "dismiss":
            fadeOutAndFinish(.dismissed(reason: payload?["reason"] as? String))
        case "error":
            let detail = payload?["message"] as? String ?? "The page reported an error."
            finish(.contentError(detail))
        case "resize":
            if let height = payload?["height"] as? Double {
                resize(toHeight: CGFloat(height))
            }
        case "log":
            if let text = payload?["message"] as? String {
                logger.debug("page: \(text, privacy: .public)")
            }
        default:
            logger.debug("bridge: ignoring unknown action \(action, privacy: .public)")
        }
    }

    private func isTrusted(_ origin: WKSecurityOrigin) -> Bool {
        guard let expectedHost = expectedHost else { return false }
        // WebKit reports 0 for a scheme's default port.
        let port = origin.port == 0 ? Self.defaultHTTPSPort : origin.port
        return origin.protocol.lowercased() == "https"
            && origin.host.lowercased() == expectedHost
            && port == expectedPort
    }
}

// MARK: - Navigation

extension ToastWindow: WKNavigationDelegate, WKUIDelegate {
    /// A React page mounts after this fires, which is why `ready` is preferred. Wait a
    /// grace period for it, then show anyway so a page that doesn't use the bridge
    /// still appears.
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        guard !hasPainted else { return }
        logger.debug("navigation finished; checking the document has content")

        // didFinish is not proof there is anything worth showing. WebKit reports some
        // refusals as successful navigations to an empty document, an empty 200 looks the
        // same, and Fleet serves 200 with error copy when the device token has expired.
        // All three would otherwise be reported as displayed.
        webView.evaluateJavaScript(Self.contentProbe) { [weak self] result, _ in
            guard let self = self, !self.hasPainted, !self.didFinish else { return }

            switch result as? String {
            case "empty":
                self.finish(.loadFailed("Page loaded but rendered no content."))
                return
            case "error":
                self.finish(.contentError("Page reported an error; the device token may have expired."))
                return
            default:
                break
            }

            self.logger.debug("document has content; waiting up to 1.5s for a ready message")
            let grace = DispatchWorkItem { [weak self] in
                self?.markPainted()
            }
            self.readyGrace = grace
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.5, execute: grace)
        }
    }

    func webView(
        _ webView: WKWebView,
        didFailProvisionalNavigation navigation: WKNavigation!,
        withError error: Error
    ) {
        handleNavigationFailure(error)
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        handleNavigationFailure(error)
    }

    /// Rejects HTTP errors rather than displaying an error page. The main-frame guard
    /// matters: without it a 401 on any subresource would fail the whole toast.
    ///
    /// Known gap: Fleet serves 200 with error HTML for an expired token, so a rotated
    /// token still reports success.
    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationResponse: WKNavigationResponse,
        decisionHandler: @escaping (WKNavigationResponsePolicy) -> Void
    ) {
        guard navigationResponse.isForMainFrame,
              let http = navigationResponse.response as? HTTPURLResponse,
              http.statusCode >= 400 else {
            decisionHandler(.allow)
            return
        }
        decisionHandler(.cancel)
        finish(.httpError(http.statusCode))
    }

    /// Confines the toast to the origin it was opened with. A chromeless window with
    /// no address bar must never end up somewhere the user can't identify, so link
    /// clicks go to the browser and anything else is refused.
    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }

        if isSameOrigin(url) || url.absoluteString == "about:blank" {
            decisionHandler(.allow)
            return
        }

        if navigationAction.navigationType == .linkActivated,
           let scheme = url.scheme?.lowercased(),
           ["https", "http", "mailto"].contains(scheme) {
            decisionHandler(.cancel)
            NSWorkspace.shared.open(url)
            return
        }

        logger.debug("blocked navigation to \(url.scheme ?? "?", privacy: .public) URL")
        decisionHandler(.cancel)
    }

    /// No `window.open` from a notification.
    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if let url = navigationAction.request.url,
           let scheme = url.scheme?.lowercased(),
           ["https", "http"].contains(scheme) {
            NSWorkspace.shared.open(url)
        }
        return nil
    }

    private func isSameOrigin(_ candidate: URL) -> Bool {
        candidate.scheme?.lowercased() == "https"
            && candidate.host?.lowercased() == expectedHost
            && (candidate.port ?? Self.defaultHTTPSPort) == expectedPort
    }

    private func handleNavigationFailure(_ error: Error) {
        let nsError = error as NSError
        logger.debug("navigation failed: \(nsError.domain, privacy: .public) \(nsError.code)")
        // A cancelled load is what our own policy decisions produce, so it isn't a
        // failure in itself — the decision that caused it already reported.
        guard nsError.code != NSURLErrorCancelled else { return }
        finish(.loadFailed(error.localizedDescription))
    }

    /// The page has rendered: show it, if it isn't up already.
    private func markPainted() {
        guard !hasPainted else { return }
        hasPainted = true
        readyGrace?.cancel()
        readyGrace = nil
        show()
    }
}

// MARK: - Views

/// Borderless panels can't become key by default, which would stop the webview from
/// receiving keyboard input.
private final class KeyablePanel: NSPanel {
    override var canBecomeKey: Bool { true }
}

/// The window's root view, larger than the card so the shadow has room. A layer-backed
/// view hit-tests its whole frame, so without the override it would swallow clicks well
/// outside the visible edge.
private final class HaloView: NSView {
    /// The visible card, in this view's coordinates.
    var cardRect: NSRect = .zero

    override func hitTest(_ point: NSPoint) -> NSView? {
        // `point` arrives in the superview's coordinate space.
        let local = convert(point, from: superview)
        guard cardRect.contains(local) else { return nil }
        return super.hitTest(point)
    }
}

/// Shared by the card and the shadow backing behind it, which must match: the rects are
/// coincident, so any colour difference shows as a fringe at the rounded corners.
private func toastCardFill(for appearance: NSAppearance) -> NSColor {
    let isDark = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua
    return isDark ? NSColor(white: 0.16, alpha: 1) : .white
}

/// Opaque card backing for the `solid` style: white in light mode, dark grey in dark
/// mode, with a hairline rim. Repaints automatically when the system appearance
/// changes.
private final class SolidCardView: NSView {
    /// The dark value is stronger on purpose: against a dark desktop the black shadow
    /// is invisible, so the rim is the only thing separating the card.
    private static let lightBorder = NSColor(white: 0, alpha: 0.16)
    private static let darkBorder = NSColor(white: 1, alpha: 0.24)

    override var wantsUpdateLayer: Bool { true }

    override func updateLayer() {
        let isDark = effectiveAppearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua
        layer?.backgroundColor = toastCardFill(for: effectiveAppearance).cgColor
        // Drawn inside the rounded edge, since the layer is masked to its corner
        // radius. 1pt renders crisply on Retina.
        layer?.borderWidth = 1
        layer?.borderColor = (isDark ? Self.darkBorder : Self.lightBorder).cgColor
    }
}

/// Gives the drop shadow an opaque rounded shape to cast from, filled to match the card
/// so it never shows through the card's antialiased corners.
private final class ShadowBackingView: NSView {
    override var wantsUpdateLayer: Bool { true }

    override func updateLayer() {
        layer?.backgroundColor = toastCardFill(for: effectiveAppearance).cgColor
    }
}

/// Forwards bridge messages without retaining the target, so `ToastWindow` can be
/// released even though `WKUserContentController` holds its handlers strongly.
private final class WeakScriptMessageProxy: NSObject, WKScriptMessageHandler {
    private weak var target: WKScriptMessageHandler?

    init(_ target: WKScriptMessageHandler) {
        self.target = target
    }

    func userContentController(
        _ userContentController: WKUserContentController,
        didReceive message: WKScriptMessage
    ) {
        target?.userContentController(userContentController, didReceive: message)
    }
}
