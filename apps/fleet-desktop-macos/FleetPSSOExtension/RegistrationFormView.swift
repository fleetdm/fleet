// RegistrationFormView.swift
// FleetPSSOExtension
//
// Native username/password form shown inside the Platform SSO registration
// window when macOS asks the extension to register a user it hasn't
// authenticated yet. Fleet proxies password grants to the IdP, so there is no
// browser leg — a plain AppKit form is the whole sign-in surface.

import Cocoa

final class RegistrationFormView: NSView {
    var onSubmit: ((_ username: String, _ password: String) -> Void)?
    var onCancel: (() -> Void)?

    let usernameField = NSTextField()
    private let passwordField = NSSecureTextField()
    private let errorLabel = NSTextField(wrappingLabelWithString: "")
    private let spinner = NSProgressIndicator()
    private let signInButton = NSButton(title: "Sign in", target: nil, action: nil)
    private let cancelButton = NSButton(title: "Cancel", target: nil, action: nil)
    private var isBusy = false

    init(initialUserName: String?) {
        super.init(frame: .zero)
        usernameField.stringValue = initialUserName ?? ""
        buildLayout()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("RegistrationFormView is built programmatically")
    }

    func setBusy(_ busy: Bool) {
        isBusy = busy
        usernameField.isEnabled = !busy
        passwordField.isEnabled = !busy
        signInButton.isEnabled = !busy
        cancelButton.isEnabled = !busy
        if busy {
            errorLabel.isHidden = true
            spinner.startAnimation(nil)
        } else {
            spinner.stopAnimation(nil)
        }
        spinner.isHidden = !busy
    }

    func showError(_ message: String) {
        errorLabel.stringValue = message
        errorLabel.isHidden = false
        passwordField.stringValue = ""
        window?.makeFirstResponder(passwordField)
    }

    private func buildLayout() {
        let title = NSTextField(labelWithString: "Sign in to your organization")
        title.font = .systemFont(ofSize: 17, weight: .semibold)

        let subtitle = NSTextField(wrappingLabelWithString:
            "Enter the username and password you use with your identity provider "
            + "to finish setting up Platform SSO for this Mac account.")
        subtitle.font = .systemFont(ofSize: 12)
        subtitle.textColor = .secondaryLabelColor

        usernameField.placeholderString = "Username or email"
        passwordField.placeholderString = "Password"

        errorLabel.font = .systemFont(ofSize: 12)
        errorLabel.textColor = .systemRed
        errorLabel.isHidden = true

        spinner.style = .spinning
        spinner.controlSize = .small
        spinner.isHidden = true

        signInButton.target = self
        signInButton.action = #selector(submit)
        signInButton.keyEquivalent = "\r"
        cancelButton.target = self
        cancelButton.action = #selector(cancel)
        cancelButton.keyEquivalent = "\u{1b}"

        let buttons = NSStackView(views: [spinner, NSView(), cancelButton, signInButton])
        buttons.orientation = .horizontal
        buttons.spacing = 8

        let stack = NSStackView(views: [title, subtitle, usernameField, passwordField, errorLabel, buttons])
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 12
        stack.setCustomSpacing(20, after: subtitle)
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)

        for view in [subtitle, usernameField, passwordField, errorLabel, buttons] {
            view.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true
        }
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 28),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -28),
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 28),
            stack.bottomAnchor.constraint(lessThanOrEqualTo: bottomAnchor, constant: -28),
        ])
    }

    // Return in either field hits the default button, so this doubles as the
    // "advance to the next empty field" handler.
    @objc private func submit() {
        guard !isBusy else { return }
        let username = usernameField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !username.isEmpty else {
            window?.makeFirstResponder(usernameField)
            return
        }
        let password = passwordField.stringValue
        guard !password.isEmpty else {
            window?.makeFirstResponder(passwordField)
            return
        }
        onSubmit?(username, password)
    }

    @objc private func cancel() {
        onCancel?()
    }
}
