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

    /// There is no screen to draw on. Unexpected — the caller has already
    /// established that someone is logged in.
    case noDisplay
}

/// A borderless, floating toast anchored to the bottom-right of the active screen,
/// hosting a `WKWebView` on a rounded card.
///
/// Ported from the standalone prototype at github.com/marko-lisica/fleet-desktop-toast-prototype,
/// which is where the window-level, material and animation choices were worked out.
/// The comments below preserve the reasoning for the non-obvious ones, because
/// several look arbitrary until you try the alternative.
///
/// Reports through two callbacks, each firing at most once: `onDisplayed` when the
/// toast is actually on screen, and `onFinish` when it is gone — or immediately, with
/// a failure, if it never appeared.
final class ToastWindow: NSObject {
    /// Where the toast content comes from. Determines which origins the JS bridge
    /// will accept messages from.
    enum Content {
        case remote(URL)
        case localFile(URL)
    }

    // MARK: - Layout

    /// Matches the Figma update card.
    private static let cardSize = NSSize(width: 525, height: 318)

    /// Gap from the screen's working area.
    private static let margin: CGFloat = 16

    /// Matches the `--radius` the page's CSS draws its card with.
    private static let cornerRadius: CGFloat = 20

    /// Transparent padding around the card so the soft drop shadow has room to
    /// render without clipping at the window edge (the window itself is invisible;
    /// only the card and its shadow show). Must exceed the shadow's blur + offset.
    private static let shadowPadding: CGFloat = 70

    private static let animationDuration: TimeInterval = 0.35

    /// Bounds for a page-requested resize, so a bad `height` can't produce a window
    /// taller than the screen or too small to read.
    private static let heightBounds: ClosedRange<CGFloat> = 200...700

    // MARK: - Bridge

    /// Message handler name the page posts to. Deliberately not the prototype's
    /// generic "toast": this will live on Fleet's My device page, where the same
    /// channel will plausibly carry non-toast messages later.
    static let bridgeChannel = "fleetDesktop"

    // MARK: - State

    private let panel: NSPanel
    private let webView: WKWebView
    private let root: HaloView
    private let shadowView: ShadowBackingView
    private let card: NSView
    private let content: Content
    private let displayTimeout: TimeInterval
    private let loadTimeout: TimeInterval
    private let logger: Logger

    /// Origin the page must post from. Captured at init from what we actually load.
    private let expectedHost: String?

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

    init(
        content: Content,
        loadTimeout: TimeInterval,
        displayTimeout: TimeInterval,
        logger: Logger
    ) {
        self.content = content
        self.loadTimeout = loadTimeout
        self.displayTimeout = displayTimeout
        self.logger = logger
        self.cardSize = Self.cardSize

        switch content {
        case .remote(let url):
            self.expectedHost = url.host?.lowercased()
        case .localFile:
            self.expectedHost = nil
        }

        let pad = Self.shadowPadding
        let cardRect = NSRect(origin: NSPoint(x: pad, y: pad), size: Self.cardSize)
        let fullRect = NSRect(
            x: 0, y: 0,
            width: Self.cardSize.width + 2 * pad,
            height: Self.cardSize.height + 2 * pad
        )

        // Borderless, floating, non-activating — shows without stealing focus, like a
        // real notification. Joins all Spaces and survives deactivation so it follows
        // the user rather than being stranded on one desktop.
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
        // A system notification shouldn't be draggable by its body, and allowing it
        // would let the user drag the window by its invisible halo.
        panel.isMovableByWindowBackground = false
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        panel.hidesOnDeactivate = false

        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = .nonPersistent()
        let contentController = WKUserContentController()
        configuration.userContentController = contentController

        webView = WKWebView(frame: NSRect(origin: .zero, size: Self.cardSize), configuration: configuration)
        // Let the card's own fill show through instead of the webview's default opaque
        // backdrop, which would be the wrong colour in dark mode and would cover the
        // card entirely. There is no public API for a transparent WKWebView, so this
        // private key is the only option; if Apple ever removes it the resulting ObjC
        // exception is uncatchable from Swift.
        webView.setValue(false, forKey: "drawsBackground")
        webView.wantsLayer = true
        webView.layer?.backgroundColor = NSColor.clear.cgColor
        // No rounded mask on the webview: the card already clips to its corner radius,
        // and masking here as well would shave the card's border at the corners.
        if #available(macOS 12.0, *) {
            // Avoids an opaque (black in dark mode) flash before the first paint
            // composites, e.g. when switching Spaces.
            webView.underPageBackgroundColor = .clear
        }
        webView.autoresizingMask = [.width, .height]

        // Transparent root filling the window. Does NOT clip, so the card's soft
        // shadow can render into the surrounding padding.
        root = HaloView(frame: fullRect)
        root.wantsLayer = true
        root.layer?.masksToBounds = false
        root.cardRect = cardRect

        // Dedicated shadow view BEHIND the card to elevate the toast off the desktop.
        // A separate view is needed because the card masks to its corner radius, and a
        // layer that clips its bounds clips its own shadow too.
        //
        // It is filled with the card colour rather than black. The two rounded rects
        // are coincident, so along the curve the card antialiases to partial coverage
        // and whatever sits behind it shows through — an opaque black backing reads as
        // a dark fringe hugging each corner.
        shadowView = ShadowBackingView(frame: cardRect)
        shadowView.wantsLayer = true
        shadowView.layer?.cornerRadius = Self.cornerRadius
        shadowView.layer?.masksToBounds = false
        // Use NSShadow, not layer.shadow* — AppKit layer-backed views don't render
        // manual layer shadows reliably; NSView.shadow does.
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

        // A weak proxy, because WKUserContentController retains its handlers strongly
        // and this object owns the controller through the webview — registering self
        // directly is an unbreakable cycle.
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

    /// Loads the page and shows the toast once it has painted.
    ///
    /// Nothing is ordered in before first paint: a remote page against a slow server
    /// would otherwise flash an empty card. If the load deadline passes with no
    /// paint, this reports `.loadFailed` and shows nothing at all — a toast the user
    /// never saw must not be reported as displayed.
    func present() {
        switch content {
        case .remote(let url):
            webView.load(URLRequest(url: url))
        case .localFile(let url):
            // Read access scoped to the containing directory so the page's relative
            // asset paths resolve.
            webView.loadFileURL(url, allowingReadAccessTo: url.deletingLastPathComponent())
        }

        guard loadTimeout > 0 else { return }
        let deadline = DispatchWorkItem { [weak self] in
            guard let self = self, !self.hasPainted else { return }
            self.finish(.loadFailed("Page did not render within \(Int(self.loadTimeout))s."))
        }
        loadDeadline = deadline
        DispatchQueue.main.asyncAfter(deadline: .now() + loadTimeout, execute: deadline)
    }

    /// Anchors bottom-right on the active screen and fades in.
    private func show() {
        guard !didDisplay, !didFinish else { return }

        // Prefer the screen under the mouse cursor — with multiple displays that's
        // where the user is actually looking.
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
        // Deliberately no NSApp.activate here. The panel is non-activating precisely
        // so the toast doesn't steal focus from whatever the user is doing.

        // Fade rather than slide. A window-frame slide silently no-ops for borderless
        // non-activating panels, and animating the layer looked janky next to the
        // shadow, which has to be recomposited every frame.
        NSAnimationContext.runAnimationGroup { context in
            context.duration = Self.animationDuration
            context.timingFunction = CAMediaTimingFunction(name: .easeOut)
            panel.animator().alphaValue = 1
        }

        logger.log("toast displayed")
        onDisplayed?()

        armDisplayTimeout()
    }

    /// Window frame for the current card size, anchored bottom-right.
    ///
    /// The window is larger than the card by `shadowPadding` on every side, so the
    /// origin is offset to put the *visible card* at the anchor rather than the
    /// transparent padding. `visibleFrame` already excludes the Dock and menu bar, so
    /// the toast sits above the Dock rather than behind it.
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
        guard displayTimeout > 0 else { return }
        let deadline = DispatchWorkItem { [weak self] in
            self?.fadeOutAndFinish(.timedOut)
        }
        displayDeadline = deadline
        DispatchQueue.main.asyncAfter(deadline: .now() + displayTimeout, execute: deadline)
    }

    // MARK: - Finishing

    /// Fades the toast out, then reports. Used for outcomes the user caused, so the
    /// fade is actually visible before the process goes away.
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

    /// Single exit point. Guarded so a `dismiss` arriving mid-fade, or a navigation
    /// failure after a successful action, can't report twice.
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

    /// Breaks the webview's references to this object. Barely matters for a
    /// short-lived process, but this class is the reusable piece — the GUI app is
    /// expected to host it for in-app notifications later, where a leaked toast would
    /// keep a floating panel alive.
    private func teardown() {
        webView.stopLoading()
        webView.navigationDelegate = nil
        webView.uiDelegate = nil
        webView.configuration.userContentController
            .removeScriptMessageHandler(forName: Self.bridgeChannel)
    }

    // MARK: - Resizing

    /// Applies a page-requested height, keeping the toast anchored.
    private func resize(toHeight requested: CGFloat) {
        let height = min(max(requested, Self.heightBounds.lowerBound), Self.heightBounds.upperBound)
        guard abs(height - cardSize.height) > 0.5 else { return }

        cardSize = NSSize(width: cardSize.width, height: height)

        let pad = Self.shadowPadding
        let cardRect = NSRect(origin: NSPoint(x: pad, y: pad), size: cardSize)
        shadowView.frame = cardRect
        card.frame = cardRect
        webView.frame = NSRect(origin: .zero, size: cardSize)
        root.cardRect = cardRect

        guard let screen = panel.screen ?? NSScreen.main else { return }
        panel.setFrame(frame(on: screen), display: true)
    }

}

// MARK: - JS bridge

extension ToastWindow: WKScriptMessageHandler {
    /// Receives `window.webkit.messageHandlers.fleetDesktop.postMessage({v, action, payload})`.
    ///
    /// The envelope is versioned because the page ships on a different cadence than
    /// this app: unknown actions and a missing `payload` are tolerated, never fatal,
    /// so a newer page keeps working against an older binary.
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
        switch content {
        case .remote:
            guard let expectedHost = expectedHost else { return false }
            return origin.protocol.lowercased() == "https"
                && origin.host.lowercased() == expectedHost
        case .localFile:
            // For file:// origins `host` is the empty string, so only the scheme is
            // meaningful here.
            return origin.protocol.lowercased() == "file"
        }
    }
}

// MARK: - Navigation

extension ToastWindow: WKNavigationDelegate, WKUIDelegate {
    /// First paint is done, but the real page is React and mounts after this fires —
    /// which is why `ready` from the bridge is preferred. Give the page a short grace
    /// period to send it, then show anyway: a page that loaded but doesn't speak our
    /// protocol (the placeholder, or any Fleet version predating the bridge) should
    /// still be displayed.
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        guard !hasPainted else { return }
        logger.debug("navigation finished; checking the document has content")

        // didFinish is not proof there is anything worth showing. WebKit completes
        // some refusals — a blocked port, for instance — as a *successful* navigation
        // to an empty document rather than as an error, and an empty 200 from the
        // server looks the same. Either would otherwise put a blank card on screen and
        // report it as displayed. Any real page has at least one element in its body
        // by this point, including a React shell that hasn't mounted yet.
        webView.evaluateJavaScript("document.body ? document.body.children.length : 0") { [weak self] result, _ in
            guard let self = self, !self.hasPainted, !self.didFinish else { return }

            let children = (result as? Int) ?? 0
            guard children > 0 else {
                self.finish(.loadFailed("Page loaded but rendered no content."))
                return
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

    /// Rejects HTTP error statuses so a missing notification or an auth failure is
    /// reported rather than displayed as an error page.
    ///
    /// Note the main-frame guard: without it a 401 on any subresource would fail the
    /// whole toast.
    ///
    /// Known gap: Fleet serves HTTP 200 with error HTML when the device token has
    /// expired, so a rotated token still reports success. Fixing that properly needs
    /// the page to report it over the bridge.
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

        if isSameOrigin(url) {
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

    private func isSameOrigin(_ url: URL) -> Bool {
        switch content {
        case .remote:
            return url.scheme?.lowercased() == "https"
                && url.host?.lowercased() == expectedHost
        case .localFile:
            return url.isFileURL
        }
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

/// The window's root view. Larger than the visible card so the drop shadow has room,
/// and transparent — but a layer-backed view still hit-tests its whole frame, so
/// without the override below the toast would swallow clicks up to `shadowPadding`
/// outside its visible edge.
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

/// The card's opaque fill for the current appearance.
///
/// Shared by the visible card and the shadow backing behind it, which have to match:
/// the two rounded rects are coincident, so the card antialiases to partial coverage
/// along the curve and any difference in colour shows up as a fringe at the corners.
private func toastCardFill(for appearance: NSAppearance) -> NSColor {
    let isDark = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua
    return isDark ? NSColor(white: 0.16, alpha: 1) : .white
}

/// Opaque card backing for the `solid` style: white in light mode, dark grey in dark
/// mode, with a hairline rim. Repaints automatically when the system appearance
/// changes.
private final class SolidCardView: NSView {
    /// Rim colours, matching the tokens the page used to draw before the native card
    /// took this over.
    ///
    /// The dark value is deliberately stronger than the light one: against a dark
    /// desktop the black drop shadow is effectively invisible, so this border is the
    /// only thing separating a dark grey card from what's behind it.
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

/// Sits directly behind the card and does nothing but give the drop shadow an opaque,
/// rounded shape to cast from. Filled to match the card so it never shows through the
/// card's antialiased corners.
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
