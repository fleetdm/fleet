cask "druva-insync" do
  version "7.6.1,110931"
  sha256 "a67784b4d6789e9a671e2d77789c408116b10c89c6c8893c3f08ed6212684bf2"

  url "https://downloads.druva.com/downloads/inSync/MAC/#{version.csv.first}/inSync-#{version.csv.first}-r#{version.csv.second}.dmg"
  name "Druva inSync"
  desc "Endpoint data backup and recovery client"
  homepage "https://www.druva.com/"

  livecheck do
    skip "Druva does not expose a parseable version feed; bump manually"
  end

  depends_on macos: ">= :big_sur"

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
