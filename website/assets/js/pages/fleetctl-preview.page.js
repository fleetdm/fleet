parasails.registerPage('fleetctl-preview', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: 'macos',
    installCommands: {
      macos: 'curl -sSL https://fleetdm.com/resources/install-fleetctl.sh | bash',
      linux: 'curl -sSL https://fleetdm.com/resources/install-fleetctl.sh | bash',
      windows: `for /f "tokens=1,* delims=:" %a in ('curl -s https://api.github.com/repos/fleetdm/fleet/releases/latest ^| findstr "browser_download_url" ^| findstr "_windows.zip"') do (curl -kOL %b) && if not exist "%USERPROFILE%\\.fleetctl" mkdir "%USERPROFILE%\\.fleetctl" && for /f "delims=" %a in ('dir /b fleetctl_*_windows.zip') do tar -xf "%a" --strip-components=1 -C "%USERPROFILE%\\.fleetctl" && del "%a"`,
      npm: 'npm install fleetctl -g',
    },
    fleetctlPreviewTerminalCommand: {
      macos: '~/.fleetctl/fleetctl preview',
      linux: '~/.fleetctl/fleetctl preview',
      windows: `%USERPROFILE%\\.fleetctl\\fleetctl preview`,
      npm: 'fleetctl preview',
    }

  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickCopyInstallCommand: async function(platform) {
      let commandToInstallFleetctl = this.installCommands[platform];
      // https://caniuse.com/mdn-api_clipboard_writetext
      $('[purpose="install-copy-button"]').addClass('copied');
      await setTimeout(()=>{
        $('[purpose="install-copy-button"]').removeClass('copied');
      }, 2000);
      navigator.clipboard.writeText(commandToInstallFleetctl);
    },

    clickCopyTerminalCommand: async function(platform) {
      let commandToRunFleetPreview = this.fleetctlPreviewTerminalCommand[platform];
      if(this.trialLicenseKey && !this.userHasExpiredTrialLicense){
        commandToRunFleetPreview += ' --license-key '+this.trialLicenseKey;
      }
      $('[purpose="command-copy-button"]').addClass('copied');
      await setTimeout(()=>{
        $('[purpose="command-copy-button"]').removeClass('copied');
      }, 2000);
      // https://caniuse.com/mdn-api_clipboard_writetext
      navigator.clipboard.writeText(commandToRunFleetPreview);
    },
  }
});
