import AppKit
import AuthenticationServices

/// Runs an IdP sign-in in the system's web authentication session instead of
/// the app's WKWebView.
///
/// WebKit only lets an embedded web view use WebAuthn for domains the app is
/// associated with, so an IdP's security key (YubiKey) and passkey prompts fail
/// there. The system session is a real browser context where they work, and it
/// shares the browser's existing IdP session.
///
/// Fleet ends the sign-in by redirecting to `fleet://sso-callback`, which the
/// session intercepts and hands back here.
final class SystemSignIn: NSObject {
    static let callbackScheme = "fleet"
    static let callbackHost = "sso-callback"

    enum Failure: Error {
        case cancelled
        case unexpectedCallback
        case notStarted
    }

    private var session: ASWebAuthenticationSession?
    private weak var anchor: NSWindow?

    /// Starts a sign-in at `url`, cancelling one already in progress.
    /// `completion` is called exactly once, on the main queue, with the query
    /// parameters of the callback URL.
    func start(
        url: URL,
        anchor: NSWindow,
        completion: @escaping (Result<[String: String], Error>) -> Void
    ) {
        cancel()
        self.anchor = anchor

        // A session that fails to start may or may not also call its handler.
        var completed = false
        let finish: (Result<[String: String], Error>) -> Void = { result in
            guard !completed else { return }
            completed = true
            completion(result)
        }

        var started: ASWebAuthenticationSession?
        let session = ASWebAuthenticationSession(url: url, callbackURLScheme: Self.callbackScheme) { [weak self] callbackURL, error in
            DispatchQueue.main.async {
                if let self = self, self.session === started {
                    self.session = nil
                }
                finish(Self.result(callbackURL: callbackURL, error: error))
            }
        }
        started = session
        session.presentationContextProvider = self
        self.session = session

        if !session.start() {
            self.session = nil
            finish(.failure(Failure.notStarted))
        }
    }

    /// Cancels the sign-in in progress, if any. Its completion still runs.
    func cancel() {
        session?.cancel()
        session = nil
    }

    private static func result(callbackURL: URL?, error: Error?) -> Result<[String: String], Error> {
        guard let callbackURL = callbackURL else {
            return .failure(error ?? Failure.cancelled)
        }
        guard callbackURL.scheme?.lowercased() == callbackScheme,
              callbackURL.host?.lowercased() == callbackHost,
              let items = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false)?.queryItems else {
            return .failure(Failure.unexpectedCallback)
        }
        var params: [String: String] = [:]
        for item in items {
            params[item.name] = item.value ?? ""
        }
        return .success(params)
    }
}

extension SystemSignIn: ASWebAuthenticationPresentationContextProviding {
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        anchor ?? ASPresentationAnchor()
    }
}
