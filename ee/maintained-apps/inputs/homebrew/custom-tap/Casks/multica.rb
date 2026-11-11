cask "multica" do
  arch arm: "arm64", intel: "x64"

  version "0.8.2"
  sha256 arm:   "fd57dca1dc9299f1c36c888e9cb2271ee9deba0bf64e96984ea7cbcf9c9c7b77",
         intel: "d1c0011c6c4517bf67da16adf0f453eb4c089bdb648589440dfc1f7415a68348"

  url "https://github.com/multica-ai/multica/releases/download/v#{version}/multica-desktop-#{version}-mac-#{arch}.dmg"
  name "Multica"
  desc "Workspace for people and AI agents with a bundled local daemon"
  homepage "https://multica.ai/"

  livecheck do
    url :url
    strategy :github_latest
  end

  auto_updates true
  depends_on macos: :ventura

  app "Multica.app"

  uninstall quit: "ai.multica.desktop"
end
