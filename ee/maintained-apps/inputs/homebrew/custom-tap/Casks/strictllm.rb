cask "strictllm" do
  version "2.1.6"

  on_arm do
    sha256 "3f7a9938db6f64a5757c0884798021f05e29d3cb1d65ee06f0a9f8b0efe54a47"

    url "https://github.com/StrictLLM-LLC/download/releases/download/v#{version}/StrictLLM-#{version}-macos-arm64-StrictLLM.Chat-#{version}-arm64.dmg",
        verified: "github.com/StrictLLM-LLC/download/"
  end
  on_intel do
    sha256 "25182e942090e17e3a8fb78277e8ccdfd33f7ac38fed98f06f6e121408c7ff6f"

    url "https://github.com/StrictLLM-LLC/download/releases/download/v#{version}/StrictLLM-#{version}-macos-x64-StrictLLM.Chat-#{version}.dmg",
        verified: "github.com/StrictLLM-LLC/download/"
  end

  name "StrictLLM Chat"
  desc "Secure local chat interface for large language models"
  homepage "https://strictllm.com/"

  livecheck do
    url "https://github.com/StrictLLM-LLC/download"
    strategy :github_latest
  end

  depends_on macos: :monterey

  app "StrictLLM Chat.app"

  uninstall quit: "com.strictllm.chat"

  zap trash: [
    "~/Library/Application Support/StrictLLM Chat",
    "~/Library/Caches/com.strictllm.chat",
    "~/Library/Caches/com.strictllm.chat.ShipIt",
    "~/Library/HTTPStorages/com.strictllm.chat",
    "~/Library/HTTPStorages/com.strictllm.chat.binarycookies",
    "~/Library/Logs/StrictLLM Chat",
    "~/Library/Preferences/com.strictllm.chat.plist",
    "~/Library/Saved Application State/com.strictllm.chat.savedState",
  ]
end
