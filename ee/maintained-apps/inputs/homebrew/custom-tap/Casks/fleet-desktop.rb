cask "fleet-desktop" do
  version "1.3.4"
  sha256 "b05a04b9df26d0d4a6333a73bfa16c0389baff1931482d0011eee1fa400b232f"

  url "https://github.com/allenhouchins/fleet-desktop/releases/download/v#{version}/fleet_desktop-v#{version}.pkg"
  name "Fleet Desktop"
  desc "End-user client for Fleet device management"
  homepage "https://github.com/allenhouchins/fleet-desktop"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: ">= :ventura"

  pkg "fleet_desktop-v#{version}.pkg"

  uninstall quit:    "com.fleetdm.fleet-desktop",
            pkgutil: "com.fleetdm.fleet-desktop"

  zap trash: [
    "~/Library/Caches/com.fleetdm.fleet-desktop",
    "~/Library/HTTPStorages/com.fleetdm.fleet-desktop",
    "~/Library/HTTPStorages/com.fleetdm.fleet-desktop.binarycookies",
    "~/Library/Preferences/com.fleetdm.fleet-desktop.plist",
    "~/Library/Saved Application State/com.fleetdm.fleet-desktop.savedState",
    "~/Library/WebKit/com.fleetdm.fleet-desktop",
  ]

  caveats <<~EOS
    Fleet Desktop requires the Mac to be enrolled in MDM with the
    com.fleetdm.fleetd.config managed preferences profile. The installer
    will fail with "Installation Failed" otherwise.
  EOS
end
