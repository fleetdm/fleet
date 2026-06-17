cask "masv" do
  version "2.11.5"
  sha256 "a26921c38aa9ab97596948451ccd46f5bd40cac4ea279de3d886b537345c1a33"

  url "https://dl.massive.io/MASV.dmg"
  name "MASV"
  desc "Large file transfer client for media and production workflows"
  homepage "https://massive.io/"

  livecheck do
    skip "MASV ships a single unversioned latest DMG; bump manually"
  end

  depends_on macos: ">= :big_sur"

  app "MASV.app"

  uninstall quit: "io.masv.desktop"

  zap trash: [
    "~/Library/Application Support/MASV",
    "~/Library/Caches/io.masv.desktop",
    "~/Library/Caches/io.masv.desktop.ShipIt",
    "~/Library/HTTPStorages/io.masv.desktop",
    "~/Library/Logs/MASV",
    "~/Library/Preferences/io.masv.desktop.plist",
    "~/Library/Saved Application State/io.masv.desktop.savedState",
  ]
end
