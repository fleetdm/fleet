import Foundation

/// Detects Fleet's device-page error screen from the rendered DOM.
///
/// Fleet answers with HTTP 200 and error copy when a device token has expired, so there
/// is no status code to key off. Requiring more than one phrase to match is what keeps
/// legitimate page content from tripping it.
///
/// Shared by `BrowserWindow` and `ToastWindow`: with a copy in each, a reword of Fleet's
/// error copy would silently break detection in one and not the other.
enum FleetErrorPage {
    private static let phrases = [
        "Something went wrong",
        "Error loading software",
        "Please contact your IT admin",
    ]

    private static let minimumMatches = 2

    /// A JavaScript expression evaluating to true when the document looks like Fleet's
    /// error page. Expects `text` to be in scope, holding the body's innerText.
    static var matchesExpression: String {
        let counts = phrases
            .map { "(text.indexOf(\"\($0)\") !== -1 ? 1 : 0)" }
            .joined(separator: " + ")
        return "(\(counts)) >= \(minimumMatches)"
    }
}
