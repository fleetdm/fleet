// AppDelegate.swift
// FleetPSSO host app
//
// Empty Cocoa shell whose only job is to be installable so macOS picks up
// the bundled FleetPSSOExtension. Launching the app once after install is
// enough; the user can quit immediately afterwards.

import Cocoa

@main
final class AppDelegate: NSObject, NSApplicationDelegate {
    private var window: NSWindow?

    func applicationDidFinishLaunching(_ note: Notification) {
        let rect = NSRect(x: 0, y: 0, width: 480, height: 240)
        let style: NSWindow.StyleMask = [.titled, .closable, .miniaturizable]
        window = NSWindow(contentRect: rect, styleMask: style,
                          backing: .buffered, defer: false)
        window?.title = "Fleet PSSO"
        window?.center()
        window?.makeKeyAndOrderFront(nil)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ s: NSApplication) -> Bool {
        true
    }
}
