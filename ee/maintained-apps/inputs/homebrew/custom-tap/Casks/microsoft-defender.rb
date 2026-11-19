cask "microsoft-defender" do
  version "101.26072.0017"
  sha256 "4f2670eb11eb080e864605ff41310277228991f4ebe16a651880cb3435e4f8b7"

  url "https://go.microsoft.com/fwlink/?linkid=2097502"
  name "Microsoft Defender"
  desc "Enterprise endpoint protection and antivirus client"
  homepage "https://www.microsoft.com/microsoft-365/microsoft-defender-for-endpoint"

  livecheck do
    skip "Microsoft's download link always serves the latest build with no parseable version feed; bump manually"
  end

  depends_on macos: :sonoma

  pkg "wdav.pkg",
      choices: [
        {
          "choiceIdentifier" => "com.microsoft.package.Microsoft_AutoUpdate.app",
          "choiceAttribute"  => "selected",
          "attributeSetting" => 0,
        },
      ]

  uninstall script:  {
              executable: "/Library/Application Support/Microsoft/Defender/uninstall/uninstall",
              sudo:       true,
            },
            pkgutil: [
              "com.microsoft.wdav",
              "com.microsoft.dlp.agent",
              "com.microsoft.dlp.daemon",
              "com.microsoft.dlp.ux",
            ]

  zap trash: [
    "/Library/Application Support/Microsoft/Defender",
    "~/Library/Application Support/Microsoft/Defender",
    "~/Library/Caches/com.microsoft.wdav",
    "~/Library/Preferences/com.microsoft.wdav.plist",
  ]
end
