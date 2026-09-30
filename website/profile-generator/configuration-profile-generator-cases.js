/**
 * Test cases for the configuration profile generator, plus the checker that scores one generated
 * profile against one case's `expect` block.
 *
 * Two runners share this file, which is why it is a module and not part of either of them:
 *
 *   - `profile-generator/tests/configuration-profile-generator.test.js`, the mocha suite: one `it()` per case, one
 *     live generation per `it()`.  This is the one to run to find out whether the prompt still works.
 *   - `sails run test-llm-generated-configuration-profile`, the script: the same cases, but repeated
 *     with --parallelTests to measure how often each one passes, optionally validated with contour,
 *     and saved as a transcript that can be diffed against an earlier run.
 *
 * `_` is the Sails lodash global rather than a local require, since both runners only ever call in
 * here from inside a loaded Sails app.
 */

//  ╔═╗╔═╗╔═╗╔═╗╔═╗
//  ║  ╠═╣╚═╗║╣ ╚═╗
//  ╚═╝╩ ╩╚═╝╚═╝╚═╝
//
// `mustContain` / `mustNotContain` are compared with all whitespace stripped from both sides,
// because the JSON formats have unpredictable indentation and an assertion like
// `"maximumTimeToLock": "300000"` would otherwise fail on formatting alone.
//
// `mustNotContainOutsideCdata` is `mustNotContain` with CDATA sections blanked first, for rules that
// are about the profile rather than about a document the profile transports.  A CSP profile must not
// carry an XML declaration; the WLANProfile inside its CDATA may.
//
// Assertions are substring-only by design.  Anything cleverer would be a validator, and the design
// decision for this generator was that there isn't one -- the CANARY cases are written as exact
// substrings so this can stay dumb.  Properties that genuinely can't be checked this way (one dict
// per payload domain, distinct PayloadUUIDs, single-line CDATA, a DDM Identifier's relationship to
// its Type) are noted on the case in `readByEye` and stay a human read.
const TEST_CASES = [

  //  ╔═╗╔═╗╔═╗
  //  ║  ╚═╗╠═╝
  //  ╚═╝╚═╝╩
  {
    id: 'csp-device-password',
    profileType: 'csp',
    canary: true,
    instructions: 'Require a password to unlock the device.',
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-removable-storage',
    profileType: 'csp',
    instructions: 'Block write access to removable storage.',
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/Storage/RemovableDiskDenyWriteAccess'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool'], ['Format', 'chr'], ['Data', '0']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-min-password-length',
    profileType: 'csp',
    instructions: 'Require device passwords to be at least 12 characters long.',
    expect: {
      mustContain: ['MinDevicePasswordLength', 'DevicePasswordEnabled'],
      mustContainElement: [['Data', '12'],['Format', 'int']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-failed-attempts',
    profileType: 'csp',
    instructions: 'Wipe a mobile device after 10 failed password attempts.',
    expect: {
      mustContain: ['DeviceLock/MaxDevicePasswordFailedAttempts'],
      mustContainElement: [['Data', '10'], ['Format', 'int']],
      // <Data>0</Data> was forbidden here and should not have been: element assertions match the whole
      // document, and a correct profile contains one.  DeviceLock/MaxDevicePasswordFailedAttempts depends
      // on DeviceLock/DevicePasswordEnabled, which the prompt requires alongside it, and Microsoft
      // documents 0 as that node's "Enabled" value.  The assertion failed every correct profile and
      // passed ones that omitted the dependency.
      mustNotContainElement: [['Format', 'chr']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-wifi-wpa2-psk-with-spaces',
    profileType: 'csp',
    canary: true,
    instructions: 'Add a wifi profile for a network with the SSID "A Network" with WPA2 authentication that uses the password "aaaaaaapassword".',
    readByEye: 'The embedded <WLANProfile> must be on ONE line inside the CDATA - no SUBSTRING assertion can express that, since whitespace is stripped from both sides before comparing.',
    expect: {
      mustContain: ['A%20Network', '<![CDATA[',],
      mustContainElement: [['name', 'A Network'], ['authentication', 'WPA2PSK'], ['keyMaterial', 'aaaaaaapassword']],
      mustNotContain: ['&lt;WLANProfile', '<SyncML', 'A%20network'],
      // "<?xml" belongs here rather than in mustNotContain.  The rule it enforces is about the profile --
      // a CSP profile is a bare sequence of OMA-DM commands and must not open with a declaration -- but
      // the WLANProfile inside the CDATA is a separate document that may legitimately carry its own, and
      // a whole-document search cannot tell the two apart.  It was failing correct profiles on this case.
      mustNotContainOutsideCdata: ['<?xml'],
    }
  },
  {
    id: 'csp-logon-banner',
    profileType: 'csp',
    instructions: 'Show "Authorized users only" as a message on the sign-in screen.',
    expect: {
      mustContain: ['LocalPoliciesSecurityOptions/InteractiveLogon_MessageTextForUsersAttemptingToLogOn'],
      mustContainElement: [['Format', 'chr'], ['Data', 'Authorized users only']]
    }
  },
  {
    id: 'csp-telemetry',
    profileType: 'csp',
    instructions: 'Send the least amount of diagnostic data allowed.',
    expect: {
      mustContain: ['Vendor/MSFT/Policy/Config/System/AllowTelemetry'],
      mustContainElement: [['Format', 'int'], ['Data', '0']]
    }
  },
  {
    id: 'csp-clipboard-history',
    profileType: 'csp',
    instructions: 'Disable clipboard history.',
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/Experience/AllowClipboardHistory'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-store-app-auto-update',
    profileType: 'csp',
    instructions: 'Enforce automatic updates for Microsoft Store apps.',
    expect: {
      mustContain: ['ApplicationManagement/AllowAppStoreAutoUpdate'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Data', '2'], ['Format', 'bool']],
      mustNotContain: ['WindowsStore/DisableAutoUpdate']
    }
  },
  {
    id: 'csp-disable-snapshots',
    profileType: 'csp',
    instructions: 'Prevent Recall from saving snapshots of the screen',
    expect: {
      mustContain: ['/Vendor/MSFT/Policy/Config/WindowsAI/DisableAIDataAnalysis'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Data', '0'], ['Format', 'bool'], ['Format', 'chr']],
    }
  },
  {
    id: 'csp-disable-guest-account',
    profileType: 'csp',
    instructions: 'Disable the built-in Guest account.',
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/Accounts_EnableGuestAccountStatus'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Data', '1'], ['Format', 'bool']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-disable-onedrive',
    profileType: 'csp',
    instructions: 'Prevent OneDrive from syncing files.',
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Data', '0'], ['Format', 'bool']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-machine-inactivity-limit',
    profileType: 'csp',
    instructions: 'Set the interactive logon machine inactivity limit to 15 minutes.',
    // The node takes seconds, so 15 minutes is 900.  A model that copies the number from the
    // instructions writes <Data>15</Data>.
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/InteractiveLogon_MachineInactivityLimit'],
      mustContainElement: [['Format', 'int'], ['Data', '900']],
      mustNotContainElement: [['Data', '15']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-defender-protections',
    profileType: 'csp',
    instructions: 'Turn on Microsoft Defender real-time protection, cloud-delivered protection, behavior monitoring, and script scanning, and send safe samples automatically.',
    // SubmitSamplesConsent is an enum, not a toggle: 1 is "send safe samples", 2 is "never send", 3 is
    // "send all".
    readByEye: 'Five Items, one per node, each with <Data>1</Data>.',
    expect: {
      mustContain: [
        'Defender/AllowRealtimeMonitoring', 'Defender/AllowCloudProtection', 'Defender/AllowBehaviorMonitoring',
        'Defender/AllowScriptScanning', 'Defender/SubmitSamplesConsent'
      ],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '0'], ['Data', '2'], ['Data', '3']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },

  //  ╔╦╗╔═╗╔╗ ╦╦  ╔═╗╔═╗╔╗╔╔═╗╦╔═╗
  //  ║║║║ ║╠╩╗║║  ║╣ ║  ║║║╠╣ ║║ ╦
  //  ╩ ╩╚═╝╚═╝╩╩═╝╚═╝╚═╝╝╚╝╚  ╩╚═╝
  {
    id: 'mobileconfig-mixed-casing',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Require a 12-character passcode with no simple passcodes, and turn on automatic checking for updates.',
    // The casing checks below all run against one document, so a model cannot pass them by being
    // self-consistently wrong: the lowercase passcode keys and the capitalized AutomaticCheckEnabled
    // have to hold in the same profile.  forcePIN is not required: Apple documents that the passcode
    // payload's presence is what makes the device ask for a passcode, so minLength and allowSimple
    // enforce on their own.  A forcePIN that does appear still has to be cased correctly.
    readByEye: 'Two payload dicts should be present, each with its own PayloadUUID and an identifier suffix.',
    expect: {
      mustContainElement: [['key', 'minLength'], ['key', 'allowSimple'], ['key', 'AutomaticCheckEnabled']],
      mustNotContainElement: [['key', 'ForcePIN'], ['key', 'MinLength'], ['key', 'AllowSimple'], ['key', 'automaticCheckEnabled']]
    }
  },
  {
    id: 'mobileconfig-showfullname',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Set the login window to show a list of users instead of name and password fields.',
    // SHOWFULLNAME false is what shows the user list, so the tempting failure is inverting to true
    // on a "show a list" reading.  The adjacent pair binds the value to the key -- a bare '<false/>'
    // would be satisfied by any other key in the profile.
    expect: {
      mustContain: ['<key>SHOWFULLNAME</key><false/>'],
      mustNotContainElement: [['string', 'false']]
    }
  },
  {
    id: 'mobileconfig-disable-camera',
    profileType: 'mobileconfig',
    instructions: 'Disable the camera.',
    expect: {
      mustContain: ['com.apple.applicationaccess', 'allowCamera', '<false/>'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-login-banner',
    profileType: 'mobileconfig',
    instructions: 'Show "Authorized use only" on the login window.',
    expect: {
      mustContain: ['<key>LoginwindowText</key><string>Authorized use only</string>'],
      mustContainElement: [['string', 'Authorized use only']]
    }
  },
  {
    id: 'mobileconfig-screensaver',
    profileType: 'mobileconfig',
    instructions: 'Show the "Flurry" screensaver after 10 minutes of inactivity and require a password immediately.',
    expect: {
      mustContain: ['<key>moduleName</key><string>Flurry</string>', '<key>idleTime</key><integer>600</integer>', '<key>askForPassword</key><true/>', '<key>askForPasswordDelay</key><integer>0</integer>'],
      mustContainElement: [['string', 'Flurry']]
    }
  },
  {
    id: 'mobileconfig-firewall',
    profileType: 'mobileconfig',
    instructions: 'Turn on the firewall in stealth mode and block all incoming connections.',
    readByEye: 'All three keys must sit in ONE dict of PayloadType com.apple.security.firewall.  Count the dicts inside PayloadContent.',
    expect: {
      mustContain: ['EnableFirewall', 'EnableStealthMode', 'BlockAllIncoming', 'com.apple.security.firewall']
    }
  },
  {
    id: 'mobileconfig-two-payloads',
    profileType: 'mobileconfig',
    instructions: 'Disable AirDrop and turn off Siri.',
    readByEye: 'Both keys are com.apple.applicationaccess, so ONE dict inside PayloadContent, not two.  Its PayloadUUID must be distinct from the root\'s, and its identifier the root identifier plus a suffix.',
    expect: {
      mustContain: ['com.apple.applicationaccess', 'allowAirDrop', 'allowAssistant'],
    }
  },
  {
    id: 'mobileconfig-diagnostics',
    profileType: 'mobileconfig',
    instructions: 'Don\'t send diagnostic reports to Apple.',
    expect: {
      mustContain: ['com.apple.applicationaccess', 'allowDiagnosticSubmission', '<false/>'],
    }
  },
  {
    id: 'mobileconfig-dock-lowercase-keys',
    profileType: 'mobileconfig',
    instructions: 'Lock the Dock to the left side of the screen and set it to auto-hide.',
    // All-lowercase keys -- fails if the model PascalCases.
    expect: {
      mustContain: ['com.apple.dock', '<key>orientation</key><string>left</string>', '<key>autohide</key><true/>', '<key>position-immutable</key><true/>'],
      mustNotContainElement: [['key', 'Autohide'], ['key', 'Orientation'], ['key', 'position']]
    }
  },
  {
    id: 'mobileconfig-app-store',
    profileType: 'mobileconfig',
    instructions: 'Prevent users from downloading books tagged as erotica from the Apple Books store',
    expect: {
      mustContain: ['com.apple.applicationaccess', 'allowBookstoreErotica'],
    }
  },
  {
    id: 'mobileconfig-allow-bookstore',
    profileType: 'mobileconfig',
    instructions: 'Enable access to Bookstore app',
    // The adjacent pair binds the value to the key -- a lone '<true/>' could be satisfied by any
    // other key -- and pins the exact key, since 'allowBookstore' is a substring-prefix of
    // 'allowBookstoreErotica'.  "Enable" means true here -- the tempting failure is inverting to
    // false because the payload is named "restrictions".
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowBookstore</key><true/>'],
      mustNotContain: ['allowBookstoreErotica', '<false/>'],
      mustNotContainElement: [['key', 'AllowBookstore'], ['string', 'true']]
    }
  },
  {
    id: 'mobileconfig-disable-guest-account',
    profileType: 'mobileconfig',
    instructions: 'Disable the guest account on the Mac.',
    // com.apple.MCX has both EnableGuestAccount and DisableGuestAccount, so the pair pins the
    // key to its value rather than accepting either spelling of "off".
    expect: {
      mustContain: ['com.apple.MCX', '<key>DisableGuestAccount</key><true/>'],
      mustNotContain: ['<key>EnableGuestAccount</key><true/>'],
      mustNotContainElement: [['string', 'true']]
    }
  },
  {
    id: 'mobileconfig-gatekeeper',
    profileType: 'mobileconfig',
    instructions: 'Turn on Gatekeeper and allow apps from the App Store and identified developers.',
    expect: {
      mustContain: ['com.apple.systempolicy.control', '<key>EnableAssessment</key><true/>', '<key>AllowIdentifiedDevelopers</key><true/>']
    }
  },
  {
    id: 'mobileconfig-limit-ad-tracking',
    profileType: 'mobileconfig',
    instructions: 'Turn off personalized ads and limit ad tracking.',
    // Two keys with opposite polarity in one payload: allow* false, force* true.  Apple documents both in
    // Restrictions; the it-and-security profile uses com.apple.AdLib, a preference domain with no manifest.
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowApplePersonalizedAdvertising</key><false/>', '<key>forceLimitAdTracking</key><true/>']
    }
  },
  {
    id: 'mobileconfig-disable-content-caching',
    profileType: 'mobileconfig',
    instructions: 'Disable content caching.',
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowContentCaching</key><false/>'],
      mustNotContainElement: [['key', 'AllowContentCaching']]
    }
  },
  {
    id: 'mobileconfig-automatic-date-time',
    profileType: 'mobileconfig',
    instructions: 'Force the date and time to be set automatically.',
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>forceAutomaticDateAndTime</key><true/>']
    }
  },
  {
    id: 'mobileconfig-app-store-auto-updates',
    profileType: 'mobileconfig',
    instructions: 'Automatically install App Store app updates.',
    expect: {
      mustContain: ['com.apple.SoftwareUpdate', '<key>AutomaticallyInstallAppUpdates</key><true/>'],
      mustNotContain: ['AutomaticallyInstallMacOSUpdates']
    }
  },
  {
    id: 'mobileconfig-screen-lock-grace-period',
    profileType: 'mobileconfig',
    instructions: 'Start the screen saver after 15 minutes of inactivity and require a password within one minute after it starts.',
    // Both values are seconds; copying the numbers from the instructions gives 15 and 1.
    expect: {
      mustContain: ['com.apple.screensaver', '<key>idleTime</key><integer>900</integer>', '<key>askForPassword</key><true/>', '<key>askForPasswordDelay</key><integer>60</integer>']
    }
  },
  {
    id: 'mobileconfig-lock-screen-message',
    profileType: 'mobileconfig',
    instructions: 'Show "This device is property of Fleet Device Management Inc." on the lock screen of iPhones and iPads.',
    expect: {
      mustContain: ['com.apple.shareddeviceconfiguration', '<key>LockScreenFootnote</key><string>This device is property of Fleet Device Management Inc.</string>'],
      mustContainElement: [['string', 'This device is property of Fleet Device Management Inc.']],
      mustNotContainElement: [['key', 'IfLostReturnToMessage']]
    }
  },
  // The mobileconfig-cis-* cases come from the "Profile Method" blocks in
  // ee/cis/macos-26/cis-policy-queries.yml, which give the payload type, key and value.  Where CIS
  // relies on a key Apple does not document, the case follows Apple's schema instead, since the
  // prompt forbids undocumented keys.
  {
    id: 'mobileconfig-cis-auto-download-updates',
    profileType: 'mobileconfig',
    instructions: 'Download macOS updates automatically as soon as they are available.',
    expect: {
      mustContain: ['com.apple.SoftwareUpdate', '<key>AutomaticDownload</key><true/>'],
      mustNotContain: ['AutomaticallyInstallMacOSUpdates'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-install-macos-updates',
    profileType: 'mobileconfig',
    instructions: 'Install macOS updates automatically.',
    expect: {
      mustContain: ['com.apple.SoftwareUpdate', '<key>AutomaticallyInstallMacOSUpdates</key><true/>'],
      mustNotContain: ['AutomaticallyInstallMacOSupdates', 'AutomaticallyInstallAppUpdates'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-security-responses',
    profileType: 'mobileconfig',
    instructions: 'Install security responses and system data files automatically.',
    readByEye: 'Both keys must sit in ONE com.apple.SoftwareUpdate dict.',
    expect: {
      mustContain: ['com.apple.SoftwareUpdate', '<key>ConfigDataInstall</key><true/>', '<key>CriticalUpdateInstall</key><true/>'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-update-deferral',
    profileType: 'mobileconfig',
    instructions: 'Hold back macOS updates from users for 30 days after release.',
    // forceDelayedSoftwareUpdates is what turns the delay on; the delay keys only set its length and
    // default to 30, so omitting them is correct.  These live in com.apple.applicationaccess, not
    // com.apple.SoftwareUpdate.  Apple removes them in macOS 27 in favour of DDM.
    readByEye: 'If a delay key (enforcedSoftwareUpdateDelay or enforcedSoftwareUpdateMinorOSDeferredInstallDelay) is present, its value must be 30.',
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>forceDelayedSoftwareUpdates</key><true/>'],
      mustNotContain: ['EnforcedSoftwareUpdateDelay', 'ForceDelayedSoftwareUpdates', 'com.apple.SoftwareUpdate', 'com.apple.configuration.softwareupdate'],
      mustNotContainElement: [['string', 'true'], ['string', '30']],
    }
  },
  {
    id: 'mobileconfig-cis-media-sharing-modification',
    profileType: 'mobileconfig',
    instructions: 'Stop users from changing the Media Sharing settings.',
    // CIS also sets allowMediaSharing, which is not in Apple's schema, so it is neither required nor
    // forbidden here.
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowMediaSharingModification</key><false/>'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-cis-airplay-receiver',
    profileType: 'mobileconfig',
    instructions: 'Stop this Mac from accepting incoming AirPlay requests.',
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowAirPlayIncomingRequests</key><false/>'],
      mustNotContain: ['AllowAirPlayIncomingRequests', 'allowAirplayIncomingRequests'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-cis-writing-tools',
    profileType: 'mobileconfig',
    instructions: 'Turn off Apple Intelligence Writing Tools.',
    // The DDM spelling is AllowWritingTools in com.apple.configuration.intelligence.settings.
    expect: {
      mustContain: ['com.apple.applicationaccess', '<key>allowWritingTools</key><false/>'],
      mustNotContain: ['AllowWritingTools', 'com.apple.configuration.intelligence'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-cis-prevent-filevault-disable',
    profileType: 'mobileconfig',
    instructions: 'Stop users from turning FileVault off.',
    // dontAllowFDEEnable is the lookalike that does the opposite.
    readByEye: 'The prompt says disk encryption is often managed natively by the MDM, so "deliveryNotes" should warn about a possible conflict -- one of the few cases where an empty deliveryNotes is wrong.',
    expect: {
      mustContain: ['com.apple.MCX', '<key>dontAllowFDEDisable</key><true/>'],
      mustNotContain: ['DontAllowFDEDisable', 'dontAllowFDEEnable'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-login-window-name-and-password',
    profileType: 'mobileconfig',
    instructions: 'Make the login window ask for a username and password instead of showing a list of accounts.',
    // The inverse of mobileconfig-showfullname: true gives name-and-password fields.  Together the two
    // catch a model that always emits the same value for SHOWFULLNAME.
    expect: {
      mustContain: ['com.apple.loginwindow', '<key>SHOWFULLNAME</key><true/>'],
      mustNotContain: ['ShowFullName', 'showFullName'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-password-hints',
    profileType: 'mobileconfig',
    instructions: 'Never show password hints on the login window.',
    // Zero retries is what disables the hint; "disable means false" puts a boolean in an integer key.
    expect: {
      mustContain: ['com.apple.loginwindow', '<key>RetriesUntilHint</key><integer>0</integer>'],
      mustNotContain: ['<key>RetriesUntilHint</key><false/>'],
    }
  },
  // Apple preference domains with no payload manifest, so not in the provided schema and answered
  // from recall.  The domain casing is the trap: com.apple.Safari, not com.apple.safari.
  {
    id: 'mobileconfig-cis-bonjour-advertising',
    profileType: 'mobileconfig',
    instructions: 'Stop this Mac advertising services over Bonjour.',
    expect: {
      mustContain: ['com.apple.mDNSResponder', '<key>NoMulticastAdvertisements</key><true/>'],
      mustNotContain: ['com.apple.mdnsresponder', 'com.apple.MDNSResponder'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-safari-safe-downloads',
    profileType: 'mobileconfig',
    instructions: 'Stop Safari from automatically opening files it considers safe after downloading them.',
    expect: {
      mustContain: ['com.apple.Safari', '<key>AutoOpenSafeDownloads</key><false/>'],
      mustNotContain: ['com.apple.safari'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-cis-safari-full-url',
    profileType: 'mobileconfig',
    instructions: 'Show the full website address in the Safari address bar.',
    expect: {
      mustContain: ['com.apple.Safari', '<key>ShowFullURLInSmartSearchField</key><true/>'],
      mustNotContain: ['ShowFullUrlInSmartSearchField', 'com.apple.safari'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-terminal-secure-keyboard',
    profileType: 'mobileconfig',
    instructions: 'Turn on secure keyboard entry in Terminal.',
    expect: {
      mustContain: ['com.apple.Terminal', '<key>SecureKeyboardEntry</key><true/>'],
      mustNotContain: ['com.apple.terminal'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-cis-password-policy-multi',
    profileType: 'mobileconfig',
    instructions: 'Require a password of at least 15 characters, lock the account after 5 failed attempts, expire passwords after 365 days, and stop the last 24 passwords being reused.',
    // maxPINAgeInDays keeps a capital PIN mid-key; a model normalizing casing writes maxPinAgeInDays.
    readByEye: 'All four keys must sit in ONE com.apple.mobiledevice.passwordpolicy dict.',
    expect: {
      mustContain: [
        'com.apple.mobiledevice.passwordpolicy',
        '<key>minLength</key><integer>15</integer>',
        '<key>maxFailedAttempts</key><integer>5</integer>',
        '<key>maxPINAgeInDays</key><integer>365</integer>',
        '<key>pinHistory</key><integer>24</integer>'
      ],
      mustNotContain: ['MaxFailedAttempts', 'MinLength', 'maxPinAgeInDays', 'PinHistory'],
    }
  },
  // Grouping: N settings across M payload domains must be M dicts inside PayloadContent -- not N,
  // and not 1.  Dict counts are beyond a substring check, so each of these carries a readByEye.
  {
    id: 'mobileconfig-grouping-three-settings-two-domains',
    profileType: 'mobileconfig',
    instructions: 'Turn off AirDrop, turn off content caching, and start the screen saver after 10 minutes.',
    readByEye: 'Exactly TWO dicts in PayloadContent: com.apple.applicationaccess with both allow* keys, and com.apple.screensaver.  Each with its own PayloadUUID and a distinct identifier.',
    expect: {
      mustContain: [
        'com.apple.applicationaccess', '<key>allowAirDrop</key><false/>', '<key>allowContentCaching</key><false/>',
        'com.apple.screensaver', '<key>idleTime</key><integer>600</integer>'
      ],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-grouping-four-settings-three-domains',
    profileType: 'mobileconfig',
    instructions: 'Show "Property of Acme Corp" on the login window, disable the guest account, turn off Siri, and stop diagnostics being sent to Apple.',
    // Siri and diagnostics both belong to com.apple.applicationaccess (see mobileconfig-diagnostics),
    // so four settings make three dicts.
    readByEye: 'Exactly THREE dicts: com.apple.loginwindow, com.apple.MCX, and one com.apple.applicationaccess holding both allowAssistant and allowDiagnosticSubmission.  Distinct PayloadUUIDs and identifiers.',
    expect: {
      mustContain: [
        'com.apple.loginwindow', '<key>LoginwindowText</key><string>Property of Acme Corp</string>',
        'com.apple.MCX', '<key>DisableGuestAccount</key><true/>',
        'com.apple.applicationaccess', '<key>allowAssistant</key><false/>', '<key>allowDiagnosticSubmission</key><false/>'
      ],
      mustNotContain: ['com.apple.SubmitDiagInfo', '<key>EnableGuestAccount</key><true/>'],
    }
  },
  {
    id: 'mobileconfig-grouping-same-domain-not-split',
    profileType: 'mobileconfig',
    instructions: 'Turn off AirDrop, turn off iCloud Desktop and Documents, turn off content caching, and block personalized ads.',
    readByEye: 'All four keys are com.apple.applicationaccess, so ONE dict with four keys.  Four dicts sharing a PayloadType is what the prompt forbids and the likeliest shape for four separate asks.',
    expect: {
      mustContain: [
        'com.apple.applicationaccess',
        '<key>allowAirDrop</key><false/>',
        '<key>allowCloudDesktopAndDocuments</key><false/>',
        '<key>allowContentCaching</key><false/>',
        '<key>allowApplePersonalizedAdvertising</key><false/>'
      ],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-grouping-mixed-value-types',
    profileType: 'mobileconfig',
    instructions: 'Show "Authorized use only" on the login window, never show password hints, and require the password immediately after the screen saver starts.',
    // Three plist types in one profile: <string>, <integer>, and <true/>.
    expect: {
      mustContain: [
        'com.apple.loginwindow', '<key>LoginwindowText</key><string>Authorized use only</string>', '<key>RetriesUntilHint</key><integer>0</integer>',
        'com.apple.screensaver', '<key>askForPassword</key><true/>', '<key>askForPasswordDelay</key><integer>0</integer>'
      ],
      mustNotContainElement: [['string', 'true'], ['string', '0']],
    }
  },
  {
    id: 'mobileconfig-login-text-fidelity',
    profileType: 'mobileconfig',
    instructions: 'Show this on the login screen, exactly as written: Property of ACME-Corp (IT) — contact it@acme.example',
    // The whole string as one needle, so re-capitalizing, turning the em dash into a hyphen, or
    // entity-escaping a character that needs none all fail it.
    expect: {
      mustContain: ['com.apple.loginwindow', '<key>LoginwindowText</key><string>Property of ACME-Corp (IT) — contact it@acme.example</string>'],
    }
  },
  {
    id: 'mobileconfig-xml-escaping-in-value',
    profileType: 'mobileconfig',
    instructions: 'Set the login window message to: Research & Development — "restricted" <internal use only>',
    // The one case where escaping is required: an unescaped & or < does not parse.  Whitespace is
    // stripped before comparing, so the raw "Research&Development" is not a substring of the escaped
    // "Research&amp;Development".  Quotes may legally be left as-is or written &quot;.
    expect: {
      mustContain: ['com.apple.loginwindow', 'LoginwindowText', 'Research &amp; Development', '&lt;internal use only'],
      mustNotContain: ['Research & Development', '<internal use only>', '&amp;amp;'],
    }
  },

  //  ╔╦╗╔╦╗╔╦╗
  //   ║║ ║║║║║
  //  ═╩╝═╩╝╩ ╩
  {
    id: 'ddm-passcode-alphanumeric',
    profileType: 'ddm',
    canary: true,
    instructions: 'Require a 10-character alphanumeric passcode and lock the device after 10 failed attempts.',
    // MaximumFailedAttempts accepts 2-11, so 10 is in range: the assertion below is what catches it
    // being clamped or rewritten.
    readByEye: 'Identifier must not be a copy of Type, and must be 64 bytes or fewer.',
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', 'RequireAlphanumericPasscode', '"MinimumLength":10', '"MaximumFailedAttempts":10'],
      mustNotContain: ['requirePasscode', 'forcePIN', 'minLength']
    }
  },
  {
    id: 'ddm-beta-enroll',
    profileType: 'ddm',
    instructions: 'Enroll devices in our beta program using the enrollment token BETA-TOKEN-123, shown to users as "Acme macOS Beta".',
    expect: {
      mustContain: ['com.apple.configuration.softwareupdate.settings', 'RequireProgram', 'BETA-TOKEN-123']
    }
  },
  {
    id: 'ddm-beta-block',
    profileType: 'ddm',
    instructions: 'Block enrollment in any beta program.',
    expect: {
      mustContain: ['com.apple.configuration.softwareupdate.settings', 'ProgramEnrollment', 'AlwaysOff'],
      mustNotContain: ['AlwaysOn']
    }
  },
  {
    id: 'ddm-defer-minor-updates',
    profileType: 'ddm',
    instructions: 'Defer minor updates by 30 days.',
    expect: {
      mustContain: ['softwareupdate.settings', 'Deferrals', '"MinorPeriodInDays":30']
    }
  },
  {
    id: 'ddm-auto-install-security-updates',
    profileType: 'ddm',
    instructions: 'Automatically download and install security updates.',
    expect: {
      mustContain: ['softwareupdate.settings', '"AutomaticActions"', '"InstallSecurityUpdate":"AlwaysOn"', '"Download":"AlwaysOn"'],
    }
  },
  {
    id: 'ddm-enforce-os-version',
    profileType: 'ddm',
    instructions: 'Enforce macOS 26.1 by December 15, 2026 at 6:00 PM local time.',
    expect: {
      mustContain: ['softwareupdate.enforcement.specific', '"TargetOSVersion":"26.1"', '"TargetLocalDateTime":"2026-12-15T18:00:00"']
    }
  },
  {
    id: 'ddm-intelligence-off',
    profileType: 'ddm',
    instructions: 'Turn off every Apple Intelligence feature.',
    expect: {
      mustContain: [
        'intelligence.settings', '"AllowWritingTools":false', '"AllowGenmoji":false', '"AllowImagePlayground":false',
        '"AllowImageWand":false', '"AllowAppleIntelligenceReport":false', '"AllowPersonalizedHandwritingResults":false',
        '"AllowVisualIntelligenceSummary":false'
      ]
    }
  },
  {
    id: 'ddm-intelligence-partial',
    profileType: 'ddm',
    instructions: 'Block Writing Tools and Image Playground but leave the rest of Apple Intelligence available.',
    expect: {
      mustContain: ['intelligence.settings', '"AllowWritingTools":false', '"AllowImagePlayground":false'],
      mustNotContain: ['"AllowGenmoji":false', '"AllowImageWand":false', '"AllowAppleIntelligenceReport":false', '"AllowVisualIntelligenceSummary":false']
    }
  },
  {
    id: 'ddm-migration-assistant',
    profileType: 'ddm',
    instructions: 'Turn on managed migration and keep the Downloads/ folder out of anything that gets migrated.',
    // ExcludedPaths entries are relative to the home directory and directory paths need a trailing
    // slash, which is why the assertion is the exact array and why an absolute path is forbidden.
    expect: {
      mustContain: ['migration-assistant.settings', 'ShouldDoManagedMigration', '"ExcludedPaths":["Downloads/"]'],
      mustNotContain: ['/Users/']
    }
  },
  {
    id: 'ddm-mobileconfig-install',
    profileType: 'ddm',
    instructions: 'Install a mobileconfig profile hosted at https://www.example.com/profiles/passcode.mobileconfig',
    expect: {
      mustContain: ['configuration.legacy', '"ProfileURL":"https://www.example.com/profiles/passcode.mobileconfig"'],
    }
  },
  {
    id: 'ddm-external-storage-read-only',
    profileType: 'ddm',
    instructions: 'Make external storage read-only.',
    expect: {
      mustContain: ['com.apple.configuration.diskmanagement.settings', '"Restrictions"', '"ExternalStorage":"ReadOnly"'],
      mustNotContain: ['"Disallowed"', 'NetworkStorage']
    }
  },
  {
    id: 'ddm-passcode-complex-inactivity',
    profileType: 'ddm',
    instructions: 'Require a complex passcode of at least 6 characters and lock the device after 5 minutes of inactivity.',
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', '"RequireComplexPasscode":true', '"MinimumLength":6', '"MaximumInactivityInMinutes":5'],
      mustNotContain: ['forcePIN', 'minLength', 'maxInactivity']
    }
  },
  {
    id: 'ddm-passcode-grace-period',
    profileType: 'ddm',
    instructions: 'Require an alphanumeric passcode of at least 10 characters with at least 1 special character, allow a 1-minute grace period before the passcode is required, and lock the screen after 15 minutes of inactivity.',
    // MaximumInactivityInMinutes tops out at 15 on macOS, so 15 is the edge of the range and must
    // not be clamped.
    expect: {
      mustContain: [
        'com.apple.configuration.passcode.settings', '"RequireAlphanumericPasscode":true', '"MinimumLength":10',
        '"MinimumComplexCharacters":1', '"MaximumGracePeriodInMinutes":1', '"MaximumInactivityInMinutes":15'
      ]
    }
  },
  {
    id: 'ddm-rapid-security-response',
    profileType: 'ddm',
    instructions: 'Automatically download updates, always install security updates, let users choose when to install OS updates, and turn on Rapid Security Responses.',
    // The sub-key is "Enable", not "Enabled" -- the latter is an easy slip and is silently ignored.
    expect: {
      mustContain: [
        'softwareupdate.settings', '"Download":"AlwaysOn"', '"InstallSecurityUpdate":"AlwaysOn"',
        '"InstallOSUpdates":"Allowed"', '"RapidSecurityResponse"', '"Enable":true'
      ],
      mustNotContain: ['"Enabled"', '"InstallOSUpdates":"AlwaysOn"']
    }
  },

  {
    id: 'ddm-safari-cookies-and-popups',
    profileType: 'ddm',
    instructions: 'In Safari, only accept cookies from sites people have actually visited, and block pop-ups.',
    readByEye: 'AcceptCookies is an enum with exactly four values in the supplied schema (Never|CurrentWebsite|VisitedWebsites|Always).  "Sites people have visited" is VisitedWebsites -- confirm the model picked from the enum rather than inventing a boolean or a string like "visited".',
    expect: {
      mustContain: ['com.apple.configuration.safari.settings', 'AcceptCookies', 'VisitedWebsites', 'AllowPopups'],
      mustNotContain: ['acceptCookies', 'allowPopups', 'allowPopUps'],
    }
  },
  {
    id: 'ddm-disk-management-external-readonly',
    profileType: 'ddm',
    instructions: 'Let people read from USB drives but not write to them.',
    readByEye: 'Restrictions.ExternalStorage is a nested dictionary with the enum Allowed|ReadOnly|Disallowed.  ReadOnly is the answer; a boolean here is an unknown value in a known key.',
    expect: {
      mustContain: ['com.apple.configuration.diskmanagement.settings', 'Restrictions', 'ExternalStorage', 'ReadOnly'],
      mustNotContain: ['externalStorage', 'NetworkStorage'],
    }
  },
  {
    id: 'ddm-keyboard-restrictions',
    profileType: 'ddm',
    instructions: 'Turn off dictation and predictive text on the keyboard.',
    expect: {
      mustContain: ['com.apple.configuration.keyboard.settings', 'AllowDictation', 'AllowPredictiveText'],
      mustNotContain: ['allowDictation', 'allowPredictiveText', 'AllowSpellCheck', 'AllowAutoCorrection'],
    }
  },
  {
    id: 'ddm-intelligence-nested-apps',
    profileType: 'ddm',
    canary: true,
    instructions: 'Leave Apple Intelligence on generally, but turn off the Mail summary and the Safari summary features.',
    readByEye: 'These two live in the nested Apps dictionary -- Apps.Mail.AllowSummary and Apps.Safari.AllowSummary -- not at the top level of the payload.  A flat AllowMailSummary key is an unknown key that enforces nothing.  Also confirm the top-level Allow* keys were NOT set, since the request explicitly leaves the rest on.',
    expect: {
      mustContain: ['com.apple.configuration.intelligence.settings', 'Apps', 'Mail', 'Safari', 'AllowSummary'],
      mustNotContain: ['AllowMailSummary', 'AllowSafariSummary', 'AllowWritingTools', 'AllowGenmoji'],
    }
  },
  {
    id: 'ddm-screensharing-host-limits',
    profileType: 'ddm',
    instructions: 'On our screen sharing hosts, do not allow files to be copied in either direction, and cap virtual displays at 1.',
    readByEye: 'MaximumVirtualDisplays has a documented range of 0-2, so 1 is in range -- confirm it was not clamped or turned into a boolean.',
    expect: {
      mustContain: ['com.apple.configuration.screensharing.host.settings', 'PreventCopyFilesFromHost', 'PreventCopyFilesToHost', 'MaximumVirtualDisplays'],
      mustNotContain: ['preventCopyFilesFromHost', 'maximumVirtualDisplays'],
    }
  },
  {
    id: 'ddm-passcode-boundary-values',
    profileType: 'ddm',
    canary: true,
    instructions: 'Require a 16-character passcode, lock after 11 failed attempts, and make people change it every 730 days.',
    readByEye: 'Every one of these is the exact top of its documented range: MinimumLength 0-16, MaximumFailedAttempts 2-11, MaximumPasscodeAgeInDays 0-730.  All three are legal.  A model that "helpfully" reduces any of them to a safer-looking number has silently changed what the admin asked for.',
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', 'MinimumLength', 'MaximumFailedAttempts', 'MaximumPasscodeAgeInDays', '16', '11', '730'],
      mustNotContain: ['minLength', 'forcePIN', 'maxFailedAttempts', 'RequirePasscode": false'],
    }
  },
  {
    id: 'ddm-softwareupdate-automatic-actions',
    profileType: 'ddm',
    instructions: 'Let macOS download updates on its own, but never install OS updates without us saying so.',
    readByEye: 'AutomaticActions.Download and AutomaticActions.InstallOSUpdates each take the enum Allowed|AlwaysOn|AlwaysOff.  "Never install without approval" is AlwaysOff on InstallOSUpdates; a boolean false is an unknown value.',
    expect: {
      mustContain: ['com.apple.configuration.softwareupdate.settings', 'AutomaticActions', 'Download', 'InstallOSUpdates', 'AlwaysOff'],
      mustNotContain: ['AutomaticCheckEnabled', 'AutomaticDownload', 'automaticActions'],
    }
  },
  {
    id: 'ddm-safari-extension-any-dict',
    profileType: 'ddm',
    canary: true,
    instructions: 'Allow the Safari extension with identifier com.acme.toolbar to run, but never in private browsing.',
    readByEye: 'ManagedExtensions is an ANY dictionary keyed by the extension identifier, so com.acme.toolbar must appear as a KEY with a {State, PrivateBrowsing} object under it -- not as the value of an "Identifier" field.  PrivateBrowsing takes Allowed|AlwaysOn|AlwaysOff, so "never" is AlwaysOff.',
    expect: {
      mustContain: ['com.apple.configuration.safari.extensions.settings', 'ManagedExtensions', 'com.acme.toolbar', 'PrivateBrowsing', 'AlwaysOff'],
      mustNotContain: ['managedExtensions', 'privateBrowsing'],
    }
  },
  {
    id: 'ddm-multi-declaration-single-file',
    profileType: 'ddm',
    canary: true,
    instructions: 'Require a 12-character passcode and defer minor macOS updates by 30 days.',
    readByEye:
      'IMPORTANT -- this case exists to surface a design question, not to be graded pass/fail.\n' +
      'These are two declaration TYPES (passcode.settings and softwareupdate.settings), and Fleet ingests one\n' +
      'declaration per file: server/fleet/apple_mdm.go GetRawDeclarationValues unmarshals the upload into a single\n' +
      '{Type, Identifier, Payload} object, so a JSON array or two concatenated objects is rejected on upload.\n' +
      'There is no single-file answer to this request.  Acceptable outcomes: generate ONE declaration and describe\n' +
      'the second in deliveryNotes, or return couldNotGenerateProfile naming the conflict.  A JSON array is a bug.\n' +
      'The same question applies to the existing ddm-identifier-collision case, which asks for two declarations\n' +
      'and asserts both types appear in one configurationProfile string.',
    expect: {
      // "}{" only occurs where two top-level objects were concatenated -- whitespace is stripped
      // before comparing, so "}\n{" collapses to it.  An array shape stays a readByEye.
      mustNotContain: ['}{'],
      deliveryNotesIsNotEmpty: true,
    }
  },

  //  ╔╗╔╔═╗╔═╗╔═╗╔╦╗╦╦  ╦╔═╗  ┌─┐┌─┐┌─┐┌─┐┌─┐
  //  ║║║║╣ ║ ╦╠═╣ ║ ║╚╗╔╝║╣   │  ├─┤└─┐├┤ └─┐
  //  ╝╚╝╚═╝╚═╝╩ ╩ ╩ ╩ ╚╝ ╚═╝  └─┘┴ ┴└─┘└─┘└─┘
  //
  // Every one of these is a real CIS macOS 26 benchmark item with no profile method, chosen because
  // the control is enforced somewhere a configuration profile cannot reach.  The prompt has five
  // separate "return couldNotGenerateProfile rather than guess" rules and the current suite tests
  // none of them, so this is the whole of that coverage.
  //
  // What a failure means here: the tool invented a payload key that does not exist, wrote a profile
  // that installs cleanly and enforces nothing, and told the admin they were compliant.
  {
    id: 'negative-screen-sharing-disabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 2.3.3.1',
    instructions: 'Make sure Screen Sharing is turned off on every Mac.',
    readByEye: 'Screen Sharing is a launchd service (com.apple.screensharing), toggled with launchctl, not a managed preference.  The right answer names that.  Do NOT accept a com.apple.screensharing payload -- the domain exists for connection settings, not for turning the service off.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-remote-login-disabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 2.3.3.4',
    instructions: 'Turn off Remote Login so nobody can SSH into these Macs.',
    readByEye: 'Remote Login is the ssh launchd job.  A profile cannot disable it.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-file-sharing-disabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 2.3.3.2',
    instructions: 'Turn off File Sharing.',
    readByEye: 'SMB file sharing is a launchd service. There is no managed preference for it.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-sip-enabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 5.1.2',
    instructions: 'Make sure System Integrity Protection is enabled.',
    readByEye: 'SIP is toggled with csrutil from recoveryOS and cannot be set by MDM at all.  This is the clearest negative in the set -- if anything passes here, the abstention rules are not working.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-root-account-disabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 5.6',
    instructions: 'Make sure the root account is disabled.',
    readByEye: 'Local account state, changed with dsenableroot. Not a profile setting.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-sudo-timeout-zero',
    profileType: 'mobileconfig',
    cisId: 'macos-26 5.4',
    instructions: 'Set the sudo timeout period to zero.',
    readByEye: 'This lives in /etc/sudoers.d and is a file, not a preference domain.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-security-auditing-enabled',
    profileType: 'mobileconfig',
    cisId: 'macos-26 3.1',
    instructions: 'Make sure security auditing is turned on.',
    readByEye: 'auditd is configured through /etc/security/audit_control.  No payload domain covers it.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-no-world-writable-library-folders',
    profileType: 'mobileconfig',
    cisId: 'macos-26 5.1.7',
    instructions: 'Make sure there are no world-writable folders in the Library folder.',
    readByEye: 'Filesystem permissions are an observation about the disk, not a setting that can be enforced.  A profile cannot express this at all.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-updates-are-installed',
    profileType: 'mobileconfig',
    cisId: 'macos-26 1.1',
    instructions: 'Make sure all Apple-provided software updates are installed.',
    readByEye: 'This is a state to be audited, not a setting to enforce.  The adjacent CIS items (1.2-1.6) ARE profile-settable, so a model that confuses "install updates automatically" with "updates are installed" will generate a com.apple.SoftwareUpdate payload and look right.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-xprotect-running',
    profileType: 'mobileconfig',
    cisId: 'macos-26 5.10',
    instructions: 'Make sure XProtect is running and up to date.',
    readByEye: 'Another state-not-setting.  ConfigDataInstall (CIS 1.5) keeps XProtect definitions current and is a legitimate ADJACENT answer -- if that came back with a caveat explaining the difference, read it as a good answer rather than a failure, and tell the dev the case needs splitting.',
    expect: { expectFailure: true }
  },


];

/**
 * Compare a generated profile against a case's `expect` block.
 *
 * @param  {Dictionary} expectations
 * @param  {Dictionary} generatedProfile
 * @returns {Array} human-readable descriptions of what failed
 */
function checkExpectations(expectations, generatedProfile) {
  let failures = [];

  if(expectations.expectFailure) {
    return ['expected this request to be refused, but a profile was generated'];
  }

  // Whitespace is stripped from both sides so an assertion doesn't fail on JSON indentation.
  let stripWhitespace = (str)=>{ return String(str).replace(/\s+/g, ''); };
  let profileWithoutWhitespace = stripWhitespace(generatedProfile.profile);

  // Element matchers tolerate attributes.  A plain substring cannot be used for XML elements the
  // prompt requires attributes on: '<Format>int</Format>' never matches the real
  // '<Format xmlns="syncml:metinf">int</Format>'.  That cuts both ways -- as a mustNotContain it
  // would silently miss a wrong format rather than merely reporting a false failure.
  let escapeForRegExp = (str)=>{ return String(str).replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); };
  let elementRegExp = (tagName, textContent)=>{
    return new RegExp('<' + escapeForRegExp(tagName) + '(?:\\s[^>]*)?>\\s*' + escapeForRegExp(textContent) + '\\s*</' + escapeForRegExp(tagName) + '>');
  };

  for (let [tagName, textContent] of expectations.mustContainElement || []) {
    if(!elementRegExp(tagName, textContent).test(generatedProfile.profile)) {
      // Name what was actually there.  Reporting only what was wanted makes a wrong value and an
      // unmatched pattern read identically, and the obvious conclusion is that the matcher is broken
      // rather than the profile -- <Format>chr</Format> where int was expected looks like a regex that
      // cannot cope with the xmlns attribute, right up until you see the value.
      let elementsThatWereThere = generatedProfile.profile.match(
        new RegExp('<' + escapeForRegExp(tagName) + '(?:\\s[^>]*)?>[\\s\\S]*?</' + escapeForRegExp(tagName) + '>', 'g')
      ) || [];
      failures.push(
        `missing expected element: <${tagName}>${textContent}</${tagName}>` +
        (elementsThatWereThere.length > 0
          ? ` -- found ${_.uniq(elementsThatWereThere).slice(0, 3).map((element)=>{ return element.replace(/\s+/g, ' '); }).join(', ')}`
          : ` -- no <${tagName}> element in the profile at all`)
      );
    }
  }

  for (let [tagName, textContent] of expectations.mustNotContainElement || []) {
    if(elementRegExp(tagName, textContent).test(generatedProfile.profile)) {
      failures.push(`contains forbidden element: <${tagName}>${textContent}</${tagName}>`);
    }
  }

  for (let needle of expectations.mustContain || []) {
    if(!_.contains(profileWithoutWhitespace, stripWhitespace(needle))) {
      failures.push(`missing expected substring: ${JSON.stringify(needle)}`);
    }
  }

  for (let needle of expectations.mustNotContain || []) {
    if(_.contains(profileWithoutWhitespace, stripWhitespace(needle))) {
      failures.push(`contains forbidden substring: ${JSON.stringify(needle)}`);
    }
  }

  // Same check, but blind to anything inside a CDATA section.  A CSP profile can carry a whole other
  // document as a node's value -- a WLANProfile, an ADMX fragment -- and a rule about the profile is not
  // a rule about what it transports.  Replaced with an empty CDATA rather than deleted so a needle
  // cannot be formed by splicing together the text on either side.
  if((expectations.mustNotContainOutsideCdata || []).length > 0) {
    let outsideCdata = stripWhitespace(String(generatedProfile.profile).replace(/<!\[CDATA\[[\s\S]*?\]\]>/g, '<![CDATA[]]>'));
    for (let needle of expectations.mustNotContainOutsideCdata) {
      if(_.contains(outsideCdata, stripWhitespace(needle))) {
        failures.push(`contains forbidden substring outside CDATA: ${JSON.stringify(needle)}`);
      }
    }
  }

  return failures;
}


module.exports = {
  TEST_CASES,
  checkExpectations,
};
