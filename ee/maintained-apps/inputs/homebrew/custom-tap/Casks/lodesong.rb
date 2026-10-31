cask "lodesong" do
  version "1.9.1"
  sha256 "d0f9fd0916e22ee4a34df81d90fa686ec6b522e174fae5f798a495cf67cde193"

  url "https://github.com/FindMalek/lodesong/releases/download/v#{version}/lodesong-macos.zip"
  name "lodesong"
  desc "Self-hosted music player for Navidrome"
  homepage "https://github.com/FindMalek/lodesong"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: :sonoma

  app "lodesong.app"

  zap trash: [
    "~/Library/Caches/engineer.findmalek.lodesong",
    "~/Library/HTTPStorages/engineer.findmalek.lodesong",
    "~/Library/Preferences/engineer.findmalek.lodesong.plist",
    "~/Library/Saved Application State/engineer.findmalek.lodesong.savedState",
  ]
end
