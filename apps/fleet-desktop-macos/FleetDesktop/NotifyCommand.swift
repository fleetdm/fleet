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
            if data.count > 4096 { break } // bounds a runaway child
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

    /// Set once the handshake line has been sent, so it is never sent twice.
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
    private func armWatchdog() {
        let limit = ToastWindow.watchdogLimit
        let item = DispatchWorkItem { [weak self] in
            self?.finish(.internalError, "Watchdog fired after \(Int(limit))s.")
        }
        watchdog = item
        DispatchQueue.main.asyncAfter(deadline: .now() + limit, execute: item)
    }

    /// The toast is up. Let the parent exit, but keep running.
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
        case .contentError(let detail):
            finish(.httpError, detail)
        case .noDisplay:
            finish(.noDisplay, "No display is attached.")
        }
    }

    /// Reports, if the parent is still waiting, and exits.
    private func finish(_ code: ExitCode, _ message: String) -> Never {
        if !didReport {
            didReport = true
            handshake.send(code, message)
        }
        logger.log("finished: \(message, privacy: .public) code=\(code.rawValue)")
        exit(code.rawValue)
    }

    /// Exits after the toast was already reported as displayed, so this status is only
    /// visible in the logs.
    private func exitChild(_ code: ExitCode) -> Never {
        watchdog?.cancel()
        exit(code.rawValue)
    }
}
