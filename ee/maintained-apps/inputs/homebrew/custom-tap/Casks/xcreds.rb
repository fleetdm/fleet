cask "xcreds" do
  version "5.9,9148"
  sha256 "ab416a7d215029cfed6f292b176951ca614ce263c2b8923beee7fcf417de199a"

  url "https://github.com/twocanoes/xcreds/releases/download/tag-#{version.csv.first}(#{version.csv.second})/XCreds_Build-#{version.csv.second}_Version-#{version.csv.first}.pkg"
  name "XCreds"
  desc "Syncs the user's IdP password with their macOS login password"
  homepage "https://twocanoes.com/products/mac/xcreds/"

  livecheck do
    skip "Twocanoes does not expose a parseable XCreds version feed; bump manually"
  end

  depends_on macos: ">= :big_sur"

  pkg "XCreds_Build-#{version.csv.second}_Version-#{version.csv.first}.pkg"

  # XCreds installs a LaunchAgent that runs the app at user login, and a
  # SecurityAgentPlugin that participates in the macOS login window. The
  # postinstall script also modifies the system authorization database
  # (system.login.console) — that part cannot be reverted declaratively;
  # admins who need a full revert should restore from the rights.bak
  # captured at install time under /Library/Application Support/xcreds/.
  uninstall launchctl: "com.twocanoes.xcreds-launchagent",
            quit:      "com.twocanoes.xcreds",
            pkgutil:   "com.twocanoes.pkg.secureremoteaccess",
            delete:    [
              "/Applications/XCreds.app",
              "/Library/Security/SecurityAgentPlugins/XCredsLoginPlugin.bundle",
              "/Library/LaunchAgents/com.twocanoes.xcreds-launchagent.plist",
            ]

  zap trash: [
    "~/Library/Application Support/xcreds",
    "~/Library/Caches/com.twocanoes.xcreds",
    "~/Library/HTTPStorages/com.twocanoes.xcreds",
    "~/Library/Preferences/com.twocanoes.xcreds.plist",
    "~/Library/Saved Application State/com.twocanoes.xcreds.savedState",
    "~/Library/WebKit/com.twocanoes.xcreds",
    "/Library/Application Support/xcreds",
    "/Library/Managed Preferences/com.twocanoes.xcreds.plist",
  ]
end
