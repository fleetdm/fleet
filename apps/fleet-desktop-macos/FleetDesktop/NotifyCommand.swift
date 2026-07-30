import AppKit
import CoreGraphics
import Foundation
import WebKit
import os

/// The `notify` subcommand: show a patch notification toast and report whether it
/// reached the screen.
///
/// The command returns as soon as the toast is on screen, but the toast has to
/// outlive it — a window dies with its process. So there are two processes:
///
///     FleetDesktop notify          parent, short-lived; its exit code is what the
///       │  ...handshake pipe...    caller records
///       └─ FleetDesktop --detached-child
///                                 owns the window, lives until dismissed or the
///                                 display timeout expires
///
/// The child does all the real work and reports its outcome over the pipe; the parent
/// exists only to relay that as an exit code. Re-exec rather than `fork()`: forking a
/// process with the Swift and ObjC runtimes loaded is a known hazard, since only the
/// calling thread survives and any lock another thread held stays locked.
enum NotifyCommand {
    private static let logger = Logger(subsystem: "com.fleetdm.fleet-desktop", category: "notify")

    static func run(_ options: NotifyOptions) -> Never {
        if options.isDetachedChild {
            runChild(options)
        } else {
            runParent(options)
        }
    }

    // MARK: - Parent

    /// Spawns the child, waits for its one-line report, relays it and exits.
    private static func runParent(_ options: NotifyOptions) -> Never {
        var fds: [Int32] = [-1, -1]
        guard pipe(&fds) == 0 else {
            report(.internalError, "Could not create the handshake pipe.")
        }
        let readFD = fds[0]
        let writeFD = fds[1]

        guard let executable = Bundle.main.executablePath else {
            report(.internalError, "Could not determine our own executable path.")
        }

        let arguments = [
            executable,
            "notify",
            "--url", options.url.absoluteString,
            "--detached-child",
            "--handshake-fd", String(writeFD),
        ]

        var fileActions: posix_spawn_file_actions_t?
        posix_spawn_file_actions_init(&fileActions)
        defer { posix_spawn_file_actions_destroy(&fileActions) }

        // The child must not inherit our stdout or stderr. Callers capture script
        // output by reading until EOF, and a detached child holding the write end of
        // that pipe open would make the script look like it hung for the whole display
        // timeout even though it returned immediately.
        posix_spawn_file_actions_addopen(&fileActions, 0, "/dev/null", O_RDONLY, 0)
        posix_spawn_file_actions_addopen(&fileActions, 1, "/dev/null", O_WRONLY, 0)
        posix_spawn_file_actions_addopen(&fileActions, 2, "/dev/null", O_WRONLY, 0)
        // Keep the pipe's write end open across the exec; everything else closes.
        posix_spawn_file_actions_addinherit_np(&fileActions, writeFD)

        var pid: pid_t = 0
        let spawnStatus = withCStrings(arguments) { argv in
            posix_spawn(&pid, executable, &fileActions, nil, argv, environ)
        }
        guard spawnStatus == 0 else {
            report(.internalError, "Could not spawn the toast process (\(spawnStatus)).")
        }

        // Our copy of the write end must go, or the read below never sees EOF.
        close(writeFD)

        let line = readLine(from: readFD)
        close(readFD)

        guard let line = line, let outcome = Handshake.decode(line) else {
            report(.internalError, "The toast process exited without reporting.")
        }
        report(outcome.code, outcome.message)
    }

    /// Reads up to the first newline, or to EOF.
    private static func readLine(from fd: Int32) -> String? {
        var data = Data()
        var byte: UInt8 = 0
        while true {
            let count = Foundation.read(fd, &byte, 1)
            if count <= 0 { break }
            if byte == UInt8(ascii: "\n") { break }
            data.append(byte)
            // A well-behaved child sends a short line; this only bounds a runaway.
            if data.count > 4096 { break }
        }
        return data.isEmpty ? nil : String(data: data, encoding: .utf8)
    }

    /// Writes the outcome for the caller and exits. The only exit path in the parent.
    private static func report(_ code: ExitCode, _ message: String) -> Never {
        CLI.emit(message, toStderr: code != .displayed)
        exit(code.rawValue)
    }

    // MARK: - Child

    private static func runChild(_ options: NotifyOptions) -> Never {
        // Detach from the caller's process group first, so tearing down the script
        // that started us doesn't take the toast with it.
        setsid()
        // The parent closes the pipe once it has our line; writing after that would
        // otherwise kill us.
        signal(SIGPIPE, SIG_IGN)

        guard let handshakeFD = options.handshakeFD else {
            // cli.swift rejects this combination, so reaching here is a bug.
            exit(ExitCode.internalError.rawValue)
        }
        let handshake = Handshake(fd: handshakeFD)

        // Never log the URL itself: the server embeds the device token in its path,
        // and that is a bearer credential which os_log persists to disk.
        logger.log("target host \(options.url.host ?? "?", privacy: .public)")

        // Someone is logged in — the caller established that — but behind a locked
        // screen the toast would expire unseen while reporting success. Report instead
        // and let the caller retry.
        if isScreenLocked() {
            handshake.send(.screenLocked, "The screen is locked.")
            exit(ExitCode.screenLocked.rawValue)
        }

        let app = NSApplication.shared
        // Before anything else: without this the process is .regular (there is no
        // LSUIElement in Info.plist, and there cannot be — the GUI app needs its Dock
        // tile), which would put a second Fleet Desktop icon in the Dock for as long
        // as the toast is up. Setting it here also keeps the window during which the
        // GUI app's single-instance guard could mistake us for the primary as short as
        // possible.
        app.setActivationPolicy(.accessory)

        let delegate = ChildDelegate(url: options.url, handshake: handshake, logger: logger)
        app.delegate = delegate
        app.run()

        // NSApplication.run() doesn't return; every path leaves through the delegate.
        exit(ExitCode.internalError.rawValue)
    }


    /// Whether the session's screen is locked.
    ///
    /// `CGSessionCopyCurrentDictionary` is public, but the lock key is not formally
    /// documented — so treat an absent key as unlocked rather than failing closed. A
    /// missing dictionary means we aren't in a GUI session at all, which the caller
    /// should already have ruled out.
    private static func isScreenLocked() -> Bool {
        guard let session = CGSessionCopyCurrentDictionary() as NSDictionary? else {
            return false
        }
        return session["CGSSessionScreenIsLocked"] as? Bool ?? false
    }

    /// Runs `body` with a null-terminated argv built from `arguments`.
    private static func withCStrings<T>(
        _ arguments: [String],
        _ body: (UnsafeMutablePointer<UnsafeMutablePointer<CChar>?>) -> T
    ) -> T {
        var pointers: [UnsafeMutablePointer<CChar>?] = arguments.map { strdup($0) }
        pointers.append(nil)
        defer { pointers.forEach { if let p = $0 { free(p) } } }
        return pointers.withUnsafeMutableBufferPointer { body($0.baseAddress!) }
    }
}

// MARK: - Handshake

/// The parent/child channel: one line, one time.
///
/// Deliberately trivial to parse — the two ends are the same binary, so there's no
/// version skew to accommodate, and keeping it dumb means a partially written line
/// can't be misread as a different outcome.
struct Handshake {
    let fd: Int32

    private static let separator: Character = " "

    func send(_ code: ExitCode, _ message: String) {
        // Strip newlines so the message can't forge extra lines.
        let sanitized = message.replacingOccurrences(of: "\n", with: " ")
        let line = "\(code.rawValue)\(Handshake.separator)\(sanitized)\n"
        guard let data = line.data(using: .utf8) else { return }
        data.withUnsafeBytes { buffer in
            _ = write(fd, buffer.baseAddress, buffer.count)
        }
    }

    static func decode(_ line: String) -> (code: ExitCode, message: String)? {
        let parts = line.split(separator: separator, maxSplits: 1, omittingEmptySubsequences: false)
        guard let first = parts.first,
            let raw = Int32(first),
            let code = ExitCode(rawValue: raw)
        else {
            return nil
        }
        let message = parts.count > 1 ? String(parts[1]) : ""
        return (code, message)
    }
}

// MARK: - Child app delegate

/// Owns the toast for the lifetime of the child process.
private final class ChildDelegate: NSObject, NSApplicationDelegate {
    private let url: URL
    private let handshake: Handshake
    private let logger: Logger

    private var toast: ToastWindow?
    private var watchdog: DispatchWorkItem?

    /// True once the handshake line has been sent, so it is never sent twice.
    private var didReport = false

    init(url: URL, handshake: Handshake, logger: Logger) {
        self.url = url
        self.handshake = handshake
        self.logger = logger
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        guard !NSScreen.screens.isEmpty else {
            finish(.internalError, "No display is available.")
        }

        let toast = ToastWindow(url: url, logger: logger)
        self.toast = toast

        toast.onDisplayed = { [weak self] in
            self?.reportDisplayed()
        }
        toast.onFinish = { [weak self] outcome in
            self?.handleFinish(outcome)
        }

        armWatchdog()
        toast.present()
    }

    /// Last resort. The toast has no title bar, no close button beyond what the page
    /// draws, and no Esc handling, so a wedged WebKit would leave an undismissable
    /// window floating over everything. Bounded by both timeouts plus slack.
    private func armWatchdog() {
        let limit = ToastWindow.watchdogLimit
        let item = DispatchWorkItem { [weak self] in
            self?.finish(.internalError, "Watchdog fired after \(Int(limit))s.")
        }
        watchdog = item
        DispatchQueue.main.asyncAfter(deadline: .now() + limit, execute: item)
    }

    /// The toast is up. Tell the parent so it can exit, but keep running.
    private func reportDisplayed() {
        guard !didReport else { return }
        didReport = true
        handshake.send(.displayed, "Notification displayed.")
    }

    private func handleFinish(_ outcome: ToastOutcome) {
        switch outcome {
        case .primaryAction(let id):
            logger.log("primary action\(id.map { " id=\($0)" } ?? "", privacy: .public)")
            exitChild(.displayed)
        case .dismissed(let reason):
            logger.log("dismissed\(reason.map { " reason=\($0)" } ?? "", privacy: .public)")
            exitChild(.displayed)
        case .timedOut:
            logger.log("display timeout expired")
            exitChild(.displayed)
        case .loadFailed(let detail):
            finish(.loadFailed, "Page did not load: \(detail)")
        case .httpError(let status):
            finish(.httpError, "Page returned HTTP \(status).")
        case .noDisplay:
            finish(.internalError, "No display is available.")
        }
    }

    /// Reports (if the parent is still waiting) and exits.
    private func finish(_ code: ExitCode, _ message: String) -> Never {
        if !didReport {
            didReport = true
            handshake.send(code, message)
        }
        logger.log("finished: \(message, privacy: .public) code=\(code.rawValue)")
        exit(code.rawValue)
    }

    /// Exits after the toast has already been reported as displayed. The parent has
    /// long since exited with 0, so this status is only visible in the logs.
    private func exitChild(_ code: ExitCode) -> Never {
        watchdog?.cancel()
        exit(code.rawValue)
    }
}
