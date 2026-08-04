import AppKit
import CoreGraphics
import Foundation
import WebKit
import os

/// The `notify` subcommand. Returns as soon as the toast is on screen, but a window
/// dies with its process — so a second process owns it:
///
///     FleetDesktop notify          parent; its exit code is what the caller records
///       │  ...handshake pipe...
///       └─ FleetDesktop --detached-child     owns the window
///
/// Re-exec rather than fork(): forking with the Swift and ObjC runtimes loaded can
/// deadlock, since only the calling thread survives and other threads' locks stay held.
enum NotifyCommand {
    private static let logger = Logger(subsystem: "com.fleetdm.fleet-desktop", category: "notify")

    /// How long the parent waits for the child's line. Longer than the child's load
    /// deadline, since reporting a load failure legitimately takes that long.
    private static let handshakeTimeout: TimeInterval = ToastWindow.loadTimeout + 15

    private static let maxHandshakeLength = 4096

    static func run(_ options: NotifyOptions) -> Never {
        if options.isDetachedChild {
            runChild(options)
        } else {
            runParent(options)
        }
    }

    // MARK: - Parent

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

        // The child must not inherit our stdout or stderr. Callers read script output
        // until EOF, so a detached child holding that pipe open would make the script
        // look hung for the whole display timeout.
        posix_spawn_file_actions_addopen(&fileActions, 0, "/dev/null", O_RDONLY, 0)
        posix_spawn_file_actions_addopen(&fileActions, 1, "/dev/null", O_WRONLY, 0)
        posix_spawn_file_actions_addopen(&fileActions, 2, "/dev/null", O_WRONLY, 0)
        // Only meaningful together with CLOEXEC_DEFAULT below: without it everything is
        // inherited anyway and this line does nothing.
        posix_spawn_file_actions_addinherit_np(&fileActions, writeFD)

        // Close every other descriptor in the child. Otherwise it inherits the pipe's
        // read end plus whatever the caller had open, and holds them for the toast's
        // whole lifetime. The three addopen calls above keep stdio working.
        var attributes: posix_spawnattr_t?
        posix_spawnattr_init(&attributes)
        defer { posix_spawnattr_destroy(&attributes) }
        posix_spawnattr_setflags(&attributes, Int16(POSIX_SPAWN_CLOEXEC_DEFAULT))

        var pid: pid_t = 0
        let spawnStatus = withCStrings(arguments) { argv in
            posix_spawn(&pid, executable, &fileActions, &attributes, argv, environ)
        }
        guard spawnStatus == 0 else {
            report(.internalError, "Could not spawn the toast process (\(spawnStatus)).")
        }

        // Our copy of the write end must go, or the read below never sees EOF.
        close(writeFD)

        let line = readLine(from: readFD, deadline: Date().addingTimeInterval(handshakeTimeout))
        close(readFD)

        guard let line = line, let outcome = Handshake.decode(line) else {
            report(.internalError, "The toast process exited without reporting.")
        }
        report(outcome.code, outcome.message)
    }

    /// Reads one line, or gives up at `deadline`.
    ///
    /// The deadline is what keeps a wedged child from hanging the caller: every other
    /// bound lives on the child's main queue, which is exactly what freezes if WebKit
    /// stalls. Returns nil on timeout, EOF before a newline, or an over-long line —
    /// all of which the caller reports as an internal error.
    private static func readLine(from fd: Int32, deadline: Date) -> String? {
        var data = Data()
        var byte: UInt8 = 0

        while true {
            let remaining = deadline.timeIntervalSinceNow
            if remaining <= 0 { return nil }

            var descriptor = pollfd(fd: fd, events: Int16(POLLIN), revents: 0)
            let milliseconds = Int32(min(remaining * 1000, Double(Int32.max)))
            let ready = poll(&descriptor, 1, milliseconds)
            if ready < 0 {
                if errno == EINTR { continue }
                return nil
            }
            if ready == 0 { return nil } // deadline passed

            let count = Foundation.read(fd, &byte, 1)
            if count < 0 {
                if errno == EINTR { continue }
                return nil
            }
            if count == 0 { break } // EOF
            if byte == UInt8(ascii: "\n") { break }
            // Checked before appending, so an over-long line is rejected rather than
            // silently truncated into something that still decodes.
            if data.count >= maxHandshakeLength { return nil }
            data.append(byte)
        }

        return data.isEmpty ? nil : String(data: data, encoding: .utf8)
    }

    /// The only exit path in the parent.
    private static func report(_ code: ExitCode, _ message: String) -> Never {
        CLI.emit(message, toStderr: code != .displayed)
        exit(code.rawValue)
    }

    // MARK: - Child

    private static func runChild(_ options: NotifyOptions) -> Never {
        // Detach, so tearing down the calling script doesn't take the toast with it.
        setsid()
        // The parent closes the pipe once it has our line.
        signal(SIGPIPE, SIG_IGN)

        guard let handshakeFD = options.handshakeFD else {
            exit(ExitCode.internalError.rawValue) // cli.swift rejects this; a bug if hit
        }
        let handshake = Handshake(fd: handshakeFD)

        // Never log the URL: the server embeds the device token in its path.
        logger.log("target host \(options.url.host ?? "?", privacy: .public)")

        // Behind a locked screen the toast would expire unseen while reporting success.
        if isScreenLocked() {
            handshake.send(.screenLocked, "The screen is locked.")
            exit(ExitCode.screenLocked.rawValue)
        }

        let app = NSApplication.shared
        // Without this the process is .regular and puts a second Fleet Desktop icon in
        // the Dock. Set first, so the GUI's single-instance guard never sees us as one.
        app.setActivationPolicy(.accessory)

        let delegate = ChildDelegate(url: options.url, handshake: handshake, logger: logger)
        app.delegate = delegate
        app.run()

        exit(ExitCode.internalError.rawValue) // run() doesn't return
    }

    /// The lock key is undocumented, so an absent key means unlocked rather than a
    /// failure.
    private static func isScreenLocked() -> Bool {
        guard let session = CGSessionCopyCurrentDictionary() as NSDictionary? else {
            return false
        }
        return session["CGSSessionScreenIsLocked"] as? Bool ?? false
    }

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

/// The parent/child channel: one line, once. Kept trivial to parse so a partial line
/// cannot be misread as a different outcome.
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

    /// Guards `didReport`. The watchdog fires on a background queue, so the flag is
    /// reachable from two threads.
    private let reportLock = NSLock()
    private var didReport = false

    init(url: URL, handshake: Handshake, logger: Logger) {
        self.url = url
        self.handshake = handshake
        self.logger = logger
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        guard !NSScreen.screens.isEmpty else {
            finish(.noDisplay, "No display is attached.")
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

    /// Last resort: the toast has no title bar, no close button and no Esc handling, so
    /// a wedged WebKit would leave an undismissable window floating over everything.
    ///
    /// Deliberately NOT on the main queue. A frozen main thread is the case this exists
    /// to catch, and a watchdog scheduled there would freeze with it.
    private func armWatchdog() {
        let limit = ToastWindow.watchdogLimit
        let item = DispatchWorkItem { [weak self] in
            self?.finish(.internalError, "Watchdog fired after \(Int(limit))s.")
        }
        watchdog = item
        DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + limit, execute: item)
    }

    /// Claims the right to send the handshake line. Returns false if it is already sent.
    private func claimReport() -> Bool {
        reportLock.lock()
        defer { reportLock.unlock() }
        if didReport { return false }
        didReport = true
        return true
    }

    /// The toast is up. Let the parent exit, but keep running.
    private func reportDisplayed() {
        guard claimReport() else { return }
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
        case .contentError(let detail):
            finish(.httpError, detail)
        case .noDisplay:
            finish(.noDisplay, "No display is attached.")
        }
    }

    /// Reports, if the parent is still waiting, and exits.
    private func finish(_ code: ExitCode, _ message: String) -> Never {
        if claimReport() {
            handshake.send(code, message)
        }
        logger.log("finished: \(message, privacy: .public) code=\(code.rawValue)")
        exit(code.rawValue)
    }

    /// Exits after a user action or the display timeout.
    ///
    /// Normally the toast was already reported, so this status only reaches the logs.
    /// But a page can post `primary` or `dismiss` before it posts `ready`, in which case
    /// nothing has been reported yet and exiting silently would leave the parent reading
    /// EOF and calling it an internal error.
    private func exitChild(_ code: ExitCode) -> Never {
        watchdog?.cancel()
        if claimReport() {
            handshake.send(code, "Notification closed before it was displayed.")
        }
        exit(code.rawValue)
    }
}
