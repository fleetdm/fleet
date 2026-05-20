cask "zoom-rooms" do
  version "7.0.5.12655"
  sha256 "8fb2ab355a5bdd0acdae3c6e0f10d8d41556a3011478f4faf6198dcad96dd0d1"

  url "https://cdn.zoom.us/prod/#{version}/ZoomRooms.pkg"
  name "Zoom Rooms"
  desc "Conference room software for Zoom meetings"
  homepage "https://www.zoom.com/en/products/zoom-rooms/"

  livecheck do
    skip "Zoom does not expose a parsable Zoom Rooms version feed; bump manually"
  end

  depends_on macos: ">= :catalina"

  pkg "ZoomRooms.pkg"

  # The product is branded "Zoom Rooms" but the installer drops the app at
  # /Applications/ZoomPresence.app with the legacy bundle id us.zoom.ZoomPresence.
  uninstall launchctl: [
              "us.zoom.rooms.daemon",
              "us.zoom.rooms.tool",
            ],
            quit:      "us.zoom.ZoomPresence",
            pkgutil:   "us.zoom.pkg.zp",
            delete:    [
              "/Applications/ZoomPresence.app",
              "/Library/LaunchDaemons/us.zoom.rooms.daemon.plist",
              "/Library/LaunchDaemons/us.zoom.rooms.tool.plist",
              "/Library/PrivilegedHelperTools/us.zoom.ZoomRoomsDaemon",
              "/Library/Logs/us.zoom.ZoomRoomUpdateRecord",
              "/Library/Logs/zpinstall.log",
            ]

  zap trash: [
    "~/Library/Application Support/ZoomPresence",
    "~/Library/Caches/us.zoom.ZoomPresence",
    "~/Library/HTTPStorages/us.zoom.ZoomPresence",
    "~/Library/HTTPStorages/us.zoom.ZoomPresence.binarycookies",
    "~/Library/Preferences/us.zoom.ZoomPresence.plist",
    "~/Library/Saved Application State/us.zoom.ZoomPresence.savedState",
    "~/Library/WebKit/us.zoom.ZoomPresence",
  ]
end
