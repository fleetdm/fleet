cask "fleet-desktop" do
  version "1.2.0"
  sha256 "d58a07e71d350b6c7fc39319080c28ad56d298f41ef45c67e0114940dce41420"

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
