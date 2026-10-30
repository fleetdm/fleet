# Homebrew/homebrew-cask removed this cask on 2026-10-29 and redirected the
# token to "lingon-pro" (Lingon Pro 10), a different product with a new
# bundle identifier (com.peterborgapps.LingonPro10) that requires macOS 14+.
# Lingon X 9 is still distributed by the vendor, so Fleet keeps serving it
# from this local snapshot of the last upstream cask definition.
cask "lingon-x" do
  version "9.6.6"
  sha256 "7cafd1fc98cd23662f0670eed996698dccc98762fe1a284036ee76132553e7f3"

  url "https://www.peterborgapps.com/downloads/LingonX#{version.major}.zip"
  name "Lingon X"
  desc "Automator software to start apps, run scripts or commands and more"
  homepage "https://www.peterborgapps.com/lingon/"

  livecheck do
    skip "Lingon X 9 is superseded upstream by Lingon Pro 10; no further releases expected"
  end

  auto_updates true
  depends_on macos: ">= :monterey"

  app "Lingon X.app"

  zap trash: [
    "~/Library/Application Scripts/com.peterborgapps.LingonX*",
    "~/Library/Application Support/Lingon X",
    "~/Library/Containers/com.peterborgapps.LingonX*",
  ]
end
