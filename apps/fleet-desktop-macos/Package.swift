// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "FleetDesktopMacOS",
    platforms: [
        .macOS(.v13)  // Set this to your target version
    ],
    targets: [
        .target(
            name: "FleetDesktop",
            path: "FleetDesktop"
        ),
        .target(
            name: "FleetPSSOExtension",
            path: "FleetPSSOExtension"
        ),
    ]
)
