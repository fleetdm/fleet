cask "druva-insync@govcloud" do
  version "8.1.8,110973"
  sha256 "d28abd70ab728d2dc08865cbc68a023bceaf9abc21ef78be14f022b4f0ae58ca"

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
