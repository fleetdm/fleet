cask "druva-insync" do
  version "8.1.9,110999"
  sha256 "d03fb6c6ea4824afd70d42188970dbc1367255881cb641551244a4625a8086d2"

  url "https://downloads.druva.com/downloads/inSync/MAC/#{version.csv.first}/inSync-#{version.csv.first}-r#{version.csv.second}.dmg"
  name "Druva inSync"
  desc "Endpoint data backup and recovery client"
  homepage "https://www.druva.com/"

  livecheck do
    skip "Druva does not expose a parseable version feed; bump manually"
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
