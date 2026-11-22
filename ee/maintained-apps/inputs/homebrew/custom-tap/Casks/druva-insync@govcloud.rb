cask "druva-insync@govcloud" do
  version "8.1.9,111001"
  sha256 "674474f93c3910321c1b25e3cd47398597dca6ced40fecaef2c2f12a5ab2a7f7"

  url "https://downloads.druva.com/downloads/inSync/MAC/#{version.csv.first}_Gov/inSync-#{version.csv.first}-r#{version.csv.second}.dmg"
  name "Druva inSync (GovCloud)"
  desc "Endpoint data backup and recovery client for Druva GovCloud"
  homepage "https://www.druva.com/"

  livecheck do
    skip "Bumped by Fleet's custom-tap routine from the GovCloud macOS entry of https://downloads.druva.com/insync/js/data.json"
  end

  depends_on macos: :sonoma

  pkg "Install inSync.pkg"

  uninstall launchctl: [
              "com.druva.inSyncAgent",
              "com.druva.inSyncDecom",
              "com.druva.inSyncUpgrade",
              "com.druva.inSyncUpgradeDaemon",
            ],
            quit:      "com.druva.inSyncClient",
            pkgutil:   "com.druva.inSync.pkg",
            delete:    [
              "/Library/LaunchAgents/inSyncAgent.plist",
              "/Library/LaunchAgents/inSyncUpgrade.plist",
              "/Library/LaunchDaemons/inSyncDecommission.plist",
              "/Library/LaunchDaemons/inSyncUpgradeDaemon.plist",
            ]

  zap trash: [
    "~/Library/Application Support/Druva",
    "~/Library/Caches/com.druva.inSyncClient",
    "~/Library/Logs/Druva",
    "~/Library/Preferences/com.druva.inSyncClient.plist",
    "~/Library/Saved Application State/com.druva.inSyncClient.savedState",
  ]
end
