cask "fleet-desktop" do
  version "1.4.0"
  sha256 "c920b983524df5296c10e4b15c5789df2dacacddd5b1423562b57bb1cc6d9d71"

  url "https://download.fleetdm.com/fleet-desktop-macos/v#{version}/fleet_desktop-v#{version}.pkg"
  name "Fleet Desktop"
  desc "End-user client for Fleet device management"
  homepage "https://github.com/fleetdm/fleet/tree/main/apps/fleet-desktop-macos"

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
