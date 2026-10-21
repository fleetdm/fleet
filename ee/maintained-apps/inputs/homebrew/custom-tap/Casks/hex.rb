cask "hex" do
  version "2.3.0"
  sha256 "3a2872b29bde17f3a5c3cc2643e74eaa11ae7d6911b53a803f619a62fd55e5ce"

  url "https://downloads.hex.kitlangton.dev/releases/HEX-#{version}-arm64.dmg"
  name "Hex"
  desc "Local-first voice dictation"
  homepage "https://hex.kitlangton.com/"

  livecheck do
    url "https://downloads.hex.kitlangton.dev/appcast.xml"
    strategy :sparkle, &:short_version
  end

  auto_updates true
  depends_on arch: :arm64
  depends_on macos: :sequoia

  app "Hex.app"

  uninstall quit: "com.kitlangton.hex2"

  zap trash: [
    "~/.config/hex",
    "~/Library/Application Support/voice-control",
    "~/Library/Caches/com.kitlangton.hex2",
    "~/Library/Caches/moonshine_voice/download.moonshine.ai/model/small-streaming-en",
    "~/Library/Cookies/com.kitlangton.hex2.binarycookies",
    "~/Library/HTTPStorages/com.kitlangton.hex2",
    "~/Library/HTTPStorages/com.kitlangton.hex2.binarycookies",
    "~/Library/Preferences/com.kitlangton.hex2.plist",
    "~/Library/Saved Application State/com.kitlangton.hex2.savedState",
  ]
end
