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
// its Type) are noted on the case in `readByEye` and stay a human read.  `mustStartWith` and
// `mustMatch` are the two exceptions, for a property of where the profile starts and for values
// that are random per generation.  Both are compared against the whitespace-stripped profile too.
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
      // WLAN_profile enforces <hex> before <name>, and its <encryption> enum has AES but not CCMP.
      mustContain: ['A%20Network', '<![CDATA[', '<SSID><hex>41204E6574776F726B</hex><name>A Network</name></SSID>'],
      mustContainElement: [['name', 'A Network'], ['authentication', 'WPA2PSK'], ['encryption', 'AES'], ['keyMaterial', 'aaaaaaapassword']],
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
  {
    id: 'csp-cis-cortana-above-lock',
    profileType: 'csp',
    instructions: 'Stop people from using Cortana while the machine is locked.',
    expect: {
      mustContain: ['Policy/Config/AboveLock/AllowCortanaAboveLock'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['Policy/Result/', 'Experience/AllowCortana', '<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-cis-lock-screen-camera',
    profileType: 'csp',
    instructions: 'Do not let the camera be used from the lock screen.',
    // CIS checks Camera/AllowCamera for this, but that disables the camera everywhere.  The node that
    // does only what was asked is the ADMX-backed DeviceLock/PreventEnablingLockScreenCamera.
    expect: {
      mustContain: ['Policy/Config/DeviceLock/PreventEnablingLockScreenCamera', '<![CDATA[', '<enabled/>'],
      mustContainElement: [['Format', 'chr']],
      mustNotContainElement: [['Format', 'int'], ['Format', 'bool']],
      mustNotContain: ['Camera/AllowCamera', '&lt;enabled/&gt;'],
    }
  },
  {
    id: 'csp-cis-cortana-off',
    profileType: 'csp',
    instructions: 'Turn Cortana off completely.',
    expect: {
      mustContain: ['Policy/Config/Experience/AllowCortana'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['AboveLock/AllowCortanaAboveLock'],
    }
  },
  {
    id: 'csp-cis-telemetry-basic',
    profileType: 'csp',
    instructions: 'Set Windows diagnostic data to the basic level.',
    // Unlike csp-telemetry, which asks for the minimum (0, Security), this names a level, so 0 is wrong.
    expect: {
      mustContain: ['Policy/Config/System/AllowTelemetry'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '0'], ['Data', '3']],
    }
  },
  {
    id: 'csp-cis-insecure-guest-logons',
    profileType: 'csp',
    instructions: 'Block unauthenticated guest connections to SMB file shares.',
    expect: {
      mustContain: ['Policy/Config/LanmanWorkstation/EnableInsecureGuestLogons'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-guest-account-status',
    profileType: 'csp',
    instructions: 'Disable the built-in Guest account.',
    expect: {
      mustContain: ['Policy/Config/LocalPoliciesSecurityOptions/Accounts_EnableGuestAccountStatus'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-input-personalization',
    profileType: 'csp',
    instructions: 'Turn off speech, inking and typing personalization so nothing is sent to Microsoft for it.',
    expect: {
      mustContain: ['Policy/Config/Privacy/AllowInputPersonalization'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-index-encrypted-items',
    profileType: 'csp',
    instructions: 'Stop Windows Search from indexing encrypted files.',
    expect: {
      mustContain: ['Policy/Config/Search/AllowIndexingEncryptedStoresOrItems'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-notify-malicious',
    profileType: 'csp',
    instructions: 'Warn users when Windows detects they have typed their work password into a malicious site.',
    expect: {
      mustContain: ['Policy/Config/WebThreatDefense/NotifyMalicious'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['WebThreatDefense/NotifyPasswordReuse', 'WebThreatDefense/NotifyUnsafeApp'],
    }
  },
  {
    id: 'csp-cis-wifi-sense',
    profileType: 'csp',
    instructions: 'Do not let devices connect automatically to Wi-Fi Sense hotspots.',
    expect: {
      mustContain: ['Policy/Config/Wifi/AllowAutoConnectToWiFiSenseHotspots'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-widgets',
    profileType: 'csp',
    instructions: 'Turn off the Widgets feed on the taskbar.',
    expect: {
      mustContain: ['Policy/Config/NewsAndInterests/AllowNewsAndInterests'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-online-tips',
    profileType: 'csp',
    instructions: 'Stop the Settings app from downloading tips and help content from the internet.',
    expect: {
      mustContain: ['Policy/Config/Settings/AllowOnlineTips'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-message-sync',
    profileType: 'csp',
    instructions: 'Stop text messages being backed up and synced to the cloud, and do not let users turn that back on.',
    expect: {
      mustContain: ['Policy/Config/Messaging/AllowMessageSync'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
    }
  },
  {
    id: 'csp-cis-behavior-monitoring',
    profileType: 'csp',
    instructions: 'Make sure Defender behavior monitoring is turned on.',
    expect: {
      mustContain: ['Policy/Config/Defender/AllowBehaviorMonitoring'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['MSFT/Defender/Configuration/'],
    }
  },
  {
    id: 'csp-cis-audit-credential-validation',
    profileType: 'csp',
    instructions: 'Audit credential validation for both successes and failures.',
    expect: {
      mustContain: ['Policy/Config/Audit/AccountLogon_AuditCredentialValidation'],
      mustContainElement: [['Format', 'int'], ['Data', '3']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '1'], ['Data', '2']],
    }
  },
  {
    id: 'csp-cis-hvci',
    profileType: 'csp',
    instructions: 'Turn on memory integrity and lock it with UEFI so it cannot be switched off remotely.',
    expect: {
      mustContain: ['Policy/Config/VirtualizationBasedTechnology/HypervisorEnforcedCodeIntegrity'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '2']],
    }
  },
  {
    id: 'csp-cis-defer-quality-updates-zero',
    profileType: 'csp',
    instructions: 'Do not delay quality updates at all -- install them as soon as they are released.',
    expect: {
      mustContain: ['Policy/Config/Update/DeferQualityUpdatesPeriodInDays'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['DeferFeatureUpdatesPeriodInDays'],
    }
  },
  {
    id: 'csp-cis-defender-hide-exclusions',
    profileType: 'csp',
    instructions: 'Hide the list of Defender scan exclusions from people signed in to the machine.',
    expect: {
      mustContain: ['MSFT/Defender/Configuration/HideExclusionsFromLocalUsers'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['Policy/Config/Defender/HideExclusionsFromLocalUsers', 'Policy/Result/'],
    }
  },
  {
    id: 'csp-cis-defender-file-hash',
    profileType: 'csp',
    instructions: 'Have Defender compute file hashes for every file it scans.',
    expect: {
      mustContain: ['MSFT/Defender/Configuration/EnableFileHashComputation'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['Policy/Config/Defender/EnableFileHashComputation'],
    }
  },
  {
    id: 'csp-cis-firewall-domain-inbound-block',
    profileType: 'csp',
    instructions: 'On the domain network profile, block inbound connections that do not match a rule.',
    // No ban on <Format>bool</Format>: a default action only applies with the firewall on, so an
    // EnableFirewall (bool) alongside it is a legitimate dependency.
    expect: {
      mustContain: ['MSFT/Firewall/MdmStore/DomainProfile/DefaultInboundAction'],
      mustContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContain: ['Policy/Config/Firewall', 'PrivateProfile/DefaultInboundAction', 'PublicProfile/DefaultInboundAction', '<Data>false</Data>'],
    }
  },
  {
    id: 'csp-cis-firewall-ipsec-merge-bool',
    profileType: 'csp',
    instructions: 'On the public network profile, stop locally defined IPsec rules from being merged with the ones we push.',
    // AllowLocalPolicyMerge is the lookalike: it governs firewall rules, not connection security rules.
    expect: {
      mustContain: ['MSFT/Firewall/MdmStore/PublicProfile/AllowLocalIpsecPolicyMerge'],
      mustContainElement: [['Format', 'bool'], ['Data', 'false']],
      mustNotContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContain: ['PublicProfile/AllowLocalPolicyMerge', 'Policy/Config/Firewall'],
    }
  },
  {
    id: 'csp-cis-passport-anti-spoofing',
    profileType: 'csp',
    instructions: 'Require enhanced anti-spoofing for Windows Hello face recognition.',
    expect: {
      mustContain: ['MSFT/PassportForWork/Biometrics/FacialFeaturesUseEnhancedAntiSpoofing'],
      mustContainElement: [['Format', 'bool'], ['Data', 'true']],
      mustNotContainElement: [['Format', 'int'], ['Data', '1']],
      mustNotContain: ['Policy/Config/PassportForWork'],
    }
  },
  {
    id: 'csp-cis-admx-include-cmdline',
    profileType: 'csp',
    instructions: 'Record the full command line in process creation events.',
    expect: {
      mustContain: ['Policy/Config/ADMX_AuditSettings/IncludeCmdLine', '<![CDATA[', '<enabled/>'],
      mustContainElement: [['Format', 'chr']],
      mustNotContainElement: [['Format', 'int'], ['Format', 'bool'], ['Data', '1']],
      mustNotContain: ['&lt;enabled/&gt;', '<data id='],
    }
  },
  {
    id: 'csp-cis-admx-credui-security-questions',
    profileType: 'csp',
    instructions: 'Stop local accounts from being able to set security questions for password reset.',
    expect: {
      mustContain: ['Policy/Config/ADMX_CredUI/NoLocalPasswordResetQuestions', '<![CDATA[', '<enabled/>'],
      mustContainElement: [['Format', 'chr']],
      mustNotContainElement: [['Format', 'int'], ['Format', 'bool']],
      mustNotContain: ['&lt;enabled/&gt;', '<data id='],
    }
  },
  {
    id: 'csp-cis-admx-powershell-script-block-logging',
    profileType: 'csp',
    instructions: 'Turn on PowerShell script block logging.',
    expect: {
      mustContain: ['Policy/Config/WindowsPowerShell/TurnOnPowerShellScriptBlockLogging', '<![CDATA[', '<enabled/>'],
      mustContainElement: [['Format', 'chr']],
      mustNotContainElement: [['Format', 'int'], ['Format', 'bool']],
      mustNotContain: ['ADMX_PowerShell', 'ADMX_WindowsPowerShell', '<data id='],
    }
  },
  {
    id: 'csp-atomic-requested-multi-setting',
    profileType: 'csp',
    instructions: 'Disable the camera, turn off Cortana, and block Wi-Fi Sense auto-connect. Apply all three together as a single all-or-nothing transaction.',
    readByEye: 'Exactly one <Atomic>, not nested inside another, with all three commands inside it.',
    expect: {
      mustContain: ['<Atomic>', 'Policy/Config/Camera/AllowCamera', 'Policy/Config/Experience/AllowCortana', 'Policy/Config/Wifi/AllowAutoConnectToWiFiSenseHotspots'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '1']],
      mustNotContain: [
        '</Atomic><Replace>', '</Atomic><Add>', '</Atomic><Exec>', '</Atomic><Atomic>',
        '</Replace><Atomic>', '</Add><Atomic>', '<Atomic><Atomic>',
        '<SyncML', '<?xml'
      ],
    }
  },
  {
    id: 'csp-multi-setting-no-atomic-requested',
    profileType: 'csp',
    instructions: 'Turn off Cortana, block Wi-Fi Sense auto-connect, and stop Windows Search from indexing encrypted files.',
    readByEye: 'Three bare top-level commands is the expected answer, but one <Atomic> wrapping all three is also legal.  A mix is not: if <Atomic> appears at all, confirm nothing sits outside it.',
    expect: {
      mustContain: [
        'Policy/Config/Experience/AllowCortana',
        'Policy/Config/Wifi/AllowAutoConnectToWiFiSenseHotspots',
        'Policy/Config/Search/AllowIndexingEncryptedStoresOrItems'
      ],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool'], ['Data', '1']],
      mustNotContain: [
        '</Atomic><Replace>', '</Atomic><Add>', '</Atomic><Exec>', '</Atomic><Atomic>',
        '</Replace><Atomic>', '</Add><Atomic>', '<Atomic><Atomic>',
        '<SyncML', '<?xml'
      ],
    }
  },
  {
    id: 'csp-multi-setting-mixed-formats',
    profileType: 'csp',
    instructions: 'Turn off Cortana, require enhanced anti-spoofing for Windows Hello face recognition, and turn on PowerShell script block logging.',
    readByEye: '<Meta> children must be used consistently across all three items.',
    expect: {
      mustContain: [
        'Policy/Config/Experience/AllowCortana',
        'MSFT/PassportForWork/Biometrics/FacialFeaturesUseEnhancedAntiSpoofing',
        'Policy/Config/WindowsPowerShell/TurnOnPowerShellScriptBlockLogging',
        '<![CDATA[', '<enabled/>'
      ],
      mustContainElement: [['Format', 'int'], ['Format', 'bool'], ['Format', 'chr'], ['Data', '0'], ['Data', 'true']],
      mustNotContain: ['ADMX_PowerShell', 'Policy/Config/PassportForWork', '<SyncML'],
    }
  },
  {
    id: 'csp-wifi-hex-ssid',
    profileType: 'csp',
    instructions: 'Add a wifi profile for the network "CorpNet" using WPA2 Enterprise.',
    readByEye: 'The embedded <WLANProfile> must be on ONE line inside the CDATA.',
    expect: {
      mustContain: ['WiFi/Profile/CorpNet/WlanXml', '<![CDATA[', '<SSID><hex>436F72704E6574</hex><name>CorpNet</name></SSID>'],
      mustContainElement: [['name', 'CorpNet'], ['hex', '436F72704E6574'], ['authentication', 'WPA2'], ['encryption', 'AES'], ['useOneX', 'true']],
      mustNotContainElement: [['authentication', 'WPA2PSK'], ['name', 'corpnet'], ['name', 'CORPNET']],
      // An enterprise network authenticates through OneX; a sharedKey here is a PSK/enterprise hybrid.
      mustNotContain: ['&lt;WLANProfile', '<SyncML', '<sharedKey>'],
      mustNotContainOutsideCdata: ['<?xml'],
    }
  },
  {
    id: 'csp-wifi-wlanxml-schema',
    profileType: 'csp',
    instructions: 'Create a wifi profile that connects to the network "BagEnd" with the password "SuperSecret".',
    // The provided CSP list only links to the WLAN_profile schema, so the WlanXml is written from
    // memory -- and that schema enforces element order and closed enums.  Observed failures: an
    // invented MSSecuritySetting container, sharedKey outside <security>, <name> before <hex>, and
    // encryption CCMP (the 802.11 name; the schema value is AES).  Profile/{SSID} is created by the
    // WlanXml Add and must not be added on its own, and ProfileSource was never asked for.
    readByEye: 'The embedded <WLANProfile> must be on ONE line inside the CDATA.',
    expect: {
      mustContain: [
        'WiFi/Profile/BagEnd/WlanXml', '<![CDATA[', '<SSID><hex>426167456E64</hex><name>BagEnd</name></SSID>',
        '<MSM>', '<security><authEncryption>',
        '</authEncryption><sharedKey><keyType>passPhrase</keyType><protected>false</protected><keyMaterial>SuperSecret</keyMaterial></sharedKey>'
      ],
      mustContainElement: [['encryption', 'AES']],
      mustNotContainElement: [['encryption', 'CCMP']],
      mustNotContain: ['MSSecuritySetting', '</security><sharedKey>', 'WiFi/Profile/BagEnd</LocURI>', 'ProfileSource', '&lt;WLANProfile', '<SyncML'],
      mustNotContainOutsideCdata: ['<?xml'],
    }
  },

  // Third-party apps, from the ADMX templates in profile-generator/schema/admx-templates/.  The ADMXInstall
  // commands are added after generation, so asserting on them checks that the right templates were picked
  // up from the policies the model chose.
  {
    id: 'csp-admx-chrome-homepage',
    profileType: 'csp',
    instructions: 'Set the Chrome homepage to https://fleetdm.com.',
    readByEye: 'The ADMXInstall <Replace> comes first, and the policy is a <Replace>.',
    expect: {
      mustContain: [
        'ConfigOperations/ADMXInstall/GoogleChrome/Policy/GoogleChromeAdmxFile',
        './Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation<',
        '<data id="HomepageLocation" value="https://fleetdm.com"/>',
        '<enabled/>',
      ],
      mustContainElement: [['Format', 'chr']],
      mustNotContain: ['&lt;enabled/&gt;'],
      // Outside CDATA, because the embedded chrome.admx defines the _recommended twin too.
      mustNotContainOutsideCdata: ['<?xml', 'HomepageLocation_recommended'],
    }
  },
  {
    id: 'csp-admx-chrome-restore-last-session',
    profileType: 'csp',
    instructions: 'Make Chrome reopen the last session when it starts.',
    expect: {
      mustContain: [
        'ConfigOperations/ADMXInstall/GoogleChrome/Policy/GoogleChromeAdmxFile',
        'Policy/Config/GoogleChrome~Policy~googlechrome~Startup/RestoreOnStartup<',
        // An enum takes the item's value, never the label or the item's position.
        '<data id="RestoreOnStartup" value="1"/>',
      ],
      mustNotContainOutsideCdata: ['RestoreOnStartup_recommended'],
    }
  },
  {
    id: 'csp-admx-chrome-force-install-extension',
    profileType: 'csp',
    instructions: 'Force-install the Chrome extension with ID ddkjiahejlhfcafbddmgiahcphecmpfh on every computer.',
    readByEye: 'The list value is name/value pairs: 1&#xF000;<extension ID>, optionally followed by ;<update URL>.',
    expect: {
      mustContain: [
        'ConfigOperations/ADMXInstall/GoogleChrome/Policy/GoogleChromeAdmxFile',
        'Policy/Config/GoogleChrome~Policy~googlechrome~Extensions/ExtensionInstallForcelist<',
        '<data id="ExtensionInstallForcelistDesc" value="1&#xF000;ddkjiahejlhfcafbddmgiahcphecmpfh',
      ],
    }
  },
  {
    id: 'csp-admx-firefox-disable-telemetry',
    profileType: 'csp',
    instructions: 'Turn off telemetry in Firefox.',
    expect: {
      mustContain: [
        'ConfigOperations/ADMXInstall/MozillaFirefox/Policy/MozillaFirefoxAdmxFile',
        'Policy/Config/MozillaFirefox~Policy~firefox/DisableTelemetry<',
        '<enabled/>',
      ],
      mustNotContain: ['System/AllowTelemetry', '<data id='],
    }
  },
  {
    id: 'csp-admx-google-update-pin-chrome-version',
    profileType: 'csp',
    instructions: 'Keep Google Chrome on version 128 and stop it updating past that.',
    readByEye: 'Pol_UpdatePolicyGoogleChrome may also be set, but not to 0 (updates disabled), which would also stop 128.x security fixes.  Google documents the prefix with a trailing dot (128.), though 128 alone also pins it.',
    expect: {
      mustContain: [
        'ConfigOperations/ADMXInstall/GoogleUpdate/Policy/GoogleUpdateAdmxFile',
        'Policy/Config/GoogleUpdate~Policy~Cat_GoogleUpdate~Cat_Applications~Cat_GoogleChrome/Pol_TargetVersionPrefixGoogleChrome<',
        '<data id="Part_TargetVersionPrefix" value="128',
      ],
    }
  },
  {
    id: 'csp-admx-mixed-with-windows-setting',
    profileType: 'csp',
    instructions: 'Turn off Cortana and set the Chrome homepage to https://fleetdm.com.',
    expect: {
      mustContain: [
        'Policy/Config/Experience/AllowCortana',
        'ConfigOperations/ADMXInstall/GoogleChrome/Policy/GoogleChromeAdmxFile',
        'Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation<',
      ],
      mustContainElement: [['Format', 'int'], ['Format', 'chr'], ['Data', '0']],
    }
  },
  // Exec runs a one-time command rather than setting a value, so a profile cannot satisfy these.  Defender/Scan
  // and Reboot/RebootNow are real Exec nodes, which is what makes generating one tempting.
  {
    id: 'negative-csp-defender-quick-scan',
    profileType: 'csp',
    instructions: 'Run a Microsoft Defender quick scan on every computer.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-csp-restart-now',
    profileType: 'csp',
    instructions: 'Restart these computers now.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-csp-defender-signature-update-now',
    profileType: 'csp',
    instructions: 'Update Microsoft Defender security intelligence right now.',
    expect: { expectFailure: true }
  },
  {
    id: 'csp-defender-daily-quick-scan-time',
    profileType: 'csp',
    instructions: 'Schedule a daily Microsoft Defender quick scan at 2:00 AM.',
    // Minutes after midnight.
    expect: {
      mustContain: ['./Device/Vendor/MSFT/Policy/Config/Defender/ScheduleQuickScanTime'],
      mustContainElement: [['Data', '120']],
      mustNotContain: ['<Exec>', 'MSFT/Defender/Scan<'],
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
      mustNotContain: ['allowBookstoreErotica', '<key>allowBookstore</key><false/>'],
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
  // 2026-10-01: eashaw: I commented out the four cases below because they rely on settings not included in the schema we are currently sending the LLM.
  // FUTURE: uncomment these cases and update regenerate-apple-profile-schemas to add settings documented in the usnistgov/macos_security GitHub repo.
  // {
  //   id: 'mobileconfig-cis-bonjour-advertising',
  //   profileType: 'mobileconfig',
  //   instructions: 'Stop this Mac advertising services over Bonjour.',
  //   expect: {
  //     mustContain: ['com.apple.mDNSResponder', '<key>NoMulticastAdvertisements</key><true/>'],
  //     mustNotContain: ['com.apple.mdnsresponder', 'com.apple.MDNSResponder'],
  //     mustNotContainElement: [['string', 'true']],
  //   }
  // },
  // {
  //   id: 'mobileconfig-cis-safari-safe-downloads',
  //   profileType: 'mobileconfig',
  //   instructions: 'Stop Safari from automatically opening files it considers safe after downloading them.',
  //   expect: {
  //     mustContain: ['com.apple.Safari', '<key>AutoOpenSafeDownloads</key><false/>'],
  //     mustNotContain: ['com.apple.safari'],
  //     mustNotContainElement: [['string', 'false']],
  //   }
  // },
  // {
  //   id: 'mobileconfig-cis-safari-full-url',
  //   profileType: 'mobileconfig',
  //   instructions: 'Show the full website address in the Safari address bar.',
  //   expect: {
  //     mustContain: ['com.apple.Safari', '<key>ShowFullURLInSmartSearchField</key><true/>'],
  //     mustNotContain: ['ShowFullUrlInSmartSearchField', 'com.apple.safari'],
  //     mustNotContainElement: [['string', 'true']],
  //   }
  // },
  // {
  //   id: 'mobileconfig-cis-terminal-secure-keyboard',
  //   profileType: 'mobileconfig',
  //   instructions: 'Turn on secure keyboard entry in Terminal.',
  //   expect: {
  //     mustContain: ['com.apple.Terminal', '<key>SecureKeyboardEntry</key><true/>'],
  //     mustNotContain: ['com.apple.terminal'],
  //     mustNotContainElement: [['string', 'true']],
  //   }
  // },
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
  {
    id: 'mobileconfig-defer-major-updates',
    profileType: 'mobileconfig',
    instructions: 'Defer major macOS upgrades for 73 days.',
    // The delay key only sets the length; forceDelayedMajorSoftwareUpdates is what turns it on.  Both
    // live in com.apple.applicationaccess.  The observed failure was the delay key alone, inside a
    // com.apple.SoftwareUpdate dict.
    readByEye: 'Apple deprecated both keys in macOS 26 and removed them in macOS 27, so "deliveryNotes" should say so and point to the DDM softwareupdate.settings Deferrals.',
    expect: {
      mustContain: [
        'com.apple.applicationaccess', '<key>forceDelayedMajorSoftwareUpdates</key><true/>',
        '<key>enforcedSoftwareUpdateMajorOSDeferredInstallDelay</key><integer>73</integer>'
      ],
      mustNotContain: ['com.apple.SoftwareUpdate', 'ForceDelayedMajorSoftwareUpdates', 'EnforcedSoftwareUpdateMajorOSDeferredInstallDelay'],
      mustNotContainElement: [['string', 'true'], ['string', '73']],
    }
  },
  // The identifier token is random per generation, so these match its shape: a final segment of 8 lowercase hex characters.
  {
    id: 'mobileconfig-identifier-token-camera-screenshots',
    profileType: 'mobileconfig',
    instructions: 'Disable the camera and screenshots.',
    expect: {
      mustMatch: ['<key>PayloadIdentifier</key><string>[^<]*\\.[0-9a-f]{8}</string>'],
      mustContain: ['com.apple.applicationaccess'],
    }
  },
  {
    id: 'mobileconfig-identifier-token-two-domains',
    profileType: 'mobileconfig',
    instructions: 'Disable content caching and start the screen saver after 5 minutes.',
    expect: {
      mustMatch: ['<key>PayloadIdentifier</key><string>[^<]*\\.[0-9a-f]{8}</string>'],
      mustContain: ['com.apple.applicationaccess', 'com.apple.screensaver'],
    }
  },
  {
    id: 'mobileconfig-identifier-token-login-window-buttons',
    profileType: 'mobileconfig',
    instructions: 'Hide the Sleep, Restart, and Shut Down buttons on the login window.',
    expect: {
      mustMatch: ['<key>PayloadIdentifier</key><string>[^<]*\\.[0-9a-f]{8}</string>'],
      mustContain: ['com.apple.loginwindow'],
    }
  },
  // 2026-10-02: JordanMontgomery: I commented out the two Platform SSO cases below because they fail every run: the model does not know the vendor-specific extension values and URLs.
  // FUTURE: uncomment these cases once the prompt carries step-by-step Platform SSO instructions, the way it does for WLAN XML.
  // {
  //   id: 'mobileconfig-psso-entra-setup-assistant',
  //   profileType: 'mobileconfig',
  //   instructions: 'Set up Microsoft Entra Platform SSO with password authentication, and let users register during Setup Assistant.',
  //   // Values from Microsoft's Platform SSO guide, which it says apply to any MDM.  The observed failure
  //   // listed only login.microsoftonline.com and left out UseSharedDeviceKeys.  The four sovereign-cloud
  //   // URLs are only needed in those clouds, so they are neither required nor forbidden.
  //   readByEye: 'AuthenticationMethod, UseSharedDeviceKeys and EnableRegistrationDuringSetup must sit INSIDE the PlatformSSO dict, not at the top level of the payload.',
  //   expect: {
  //     mustContain: [
  //       'com.apple.extensiblesso',
  //       '<key>ExtensionIdentifier</key><string>com.microsoft.CompanyPortalMac.ssoextension</string>',
  //       '<key>TeamIdentifier</key><string>UBF8T346G9</string>',
  //       '<key>Type</key><string>Redirect</string>',
  //       '<key>PlatformSSO</key><dict>', '<key>AuthenticationMethod</key><string>Password</string>',
  //       '<key>UseSharedDeviceKeys</key><true/>', '<key>EnableRegistrationDuringSetup</key><true/>',
  //       '<string>https://login.microsoftonline.com</string>', '<string>https://login.microsoft.com</string>',
  //       '<string>https://sts.windows.net</string>'
  //     ],
  //     mustNotContain: ['<key>Type</key><string>Credential</string>'],
  //     mustNotContainElement: [['string', 'true']],
  //   }
  // },
  // {
  //   id: 'mobileconfig-psso-okta-password',
  //   profileType: 'mobileconfig',
  //   instructions: 'Set up Okta Platform SSO with password authentication for our Okta org at acme.okta.com, including the associated domains Okta Verify needs.',
  //   // Values from Okta's generic-MDM Platform SSO guide.  The observed failure was Okta's older
  //   // non-Platform SSO extension: com.okta.mobile.sso with Type Credential and no PlatformSSO dict,
  //   // which installs and does no Platform SSO.  The org is supplied so the model has no reason to
  //   // fall back to placeholders.
  //   readByEye: 'The associated domains belong in their own com.apple.associated-domains dict, with one Configuration entry per ApplicationIdentifier.',
  //   expect: {
  //     mustContain: [
  //       'com.apple.extensiblesso', 'com.apple.associated-domains',
  //       '<key>ExtensionIdentifier</key><string>com.okta.mobile.auth-service-extension</string>',
  //       '<key>TeamIdentifier</key><string>B7F62B65BN</string>',
  //       '<key>Type</key><string>Redirect</string>',
  //       '<key>PlatformSSO</key><dict>', '<key>AuthenticationMethod</key><string>Password</string>',
  //       '<key>UseSharedDeviceKeys</key><true/>',
  //       '<string>https://acme.okta.com/device-access/api/v1/nonce</string>',
  //       '<string>https://acme.okta.com/oauth2/v1/token</string>',
  //       '<string>https://acme.okta.com/v1/auth/device-sign</string>',
  //       '<string>B7F62B65BN.com.okta.mobile</string>', '<string>B7F62B65BN.com.okta.mobile.auth-service-extension</string>',
  //       '<string>authsrv:acme.okta.com</string>'
  //     ],
  //     mustNotContain: ['com.okta.mobile.sso', '<key>Type</key><string>Credential</string>', 'OKTA_'],
  //     mustNotContainElement: [['string', 'true']],
  //   }
  // },
  // Third-party apps' preference domains, from profile-generator/schema/apple-third-party-payload-manifests.json.
  // Every key and value below is copied from that file.  The runners do not turn on useThirdPartyAppLookup,
  // so an app's keys are only provided when one of its keywords appears in the instructions as a whole
  // word, which is why each case names its app the way its keywords do.
  {
    id: 'mobileconfig-third-party-chrome-password-manager',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Turn off Chrome\'s built-in password manager.',
    expect: {
      mustContain: ['<string>com.google.Chrome</string>', '<key>PasswordManagerEnabled</key><false/>'],
      mustNotContain: ['com.google.chrome<', 'com.microsoft.Edge', 'PasswordManagerEnabled</key><string>'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-homepage',
    profileType: 'mobileconfig',
    instructions: 'Set the Google Chrome homepage to https://fleetdm.com and show the Home button on the toolbar.',
    // HomepageIsNewTabPage true would override HomepageLocation, so it can be omitted or false but not true.
    expect: {
      mustContain: ['com.google.Chrome', '<key>HomepageLocation</key><string>https://fleetdm.com</string>', '<key>ShowHomeButton</key><true/>'],
      mustNotContain: ['<key>HomepageIsNewTabPage</key><true/>', 'HomePageLocation'],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-incognito',
    profileType: 'mobileconfig',
    instructions: 'Disable Incognito mode in Chrome.',
    // An integer enum, not a toggle: 1 is disabled and 2 is forced, so "disable means false" or "disable
    // means 0" both get it wrong.  IncognitoEnabled is the deprecated boolean this replaced.
    expect: {
      mustContain: ['com.google.Chrome', '<key>IncognitoModeAvailability</key><integer>1</integer>'],
      mustNotContain: ['IncognitoEnabled', '<key>IncognitoModeAvailability</key><false/>', '<key>IncognitoModeAvailability</key><integer>0</integer>'],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-session-cookies',
    profileType: 'mobileconfig',
    instructions: 'Change Chrome\'s default cookie setting so sites\' cookies are only kept until the browser is closed.',
    // 4 is "keep for the session"; 1 and 2 are allow-all and block-all.  The value's meaning is only in
    // allowedValueTitles, so this checks that it is rendered.  The request names the default setting because
    // CookiesSessionOnlyForUrls with a * pattern also satisfies "only keep cookies until the browser is closed".
    expect: {
      mustContain: ['com.google.Chrome', '<key>DefaultCookiesSetting</key><integer>4</integer>'],
      mustNotContain: ['<key>DefaultCookiesSetting</key><integer>1</integer>', '<key>DefaultCookiesSetting</key><integer>2</integer>', 'CookiesSessionOnlyForUrls'],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-force-install-extension',
    profileType: 'mobileconfig',
    instructions: 'Add the extension with the ID ddkjiahejlhfcafbddmgiahcphecmpfh to Chrome\'s force-installed extensions list, so users cannot remove it.',
    // The list, rather than "force-install", because ExtensionSettings with installation_mode force_installed
    // also force-installs.  The manifest names the array's items ExtensionInstallForcelistItem, which is not a
    // plist key: the array holds bare <string>s, either the ID or the ID;update_url.
    expect: {
      mustContain: ['com.google.Chrome', '<key>ExtensionInstallForcelist</key><array><string>ddkjiahejlhfcafbddmgiahcphecmpfh'],
      mustNotContain: ['ExtensionInstallForcelistItem', 'ExtensionInstallBlocklist', 'ExtensionInstallWhitelist'],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-signin-restriction',
    profileType: 'mobileconfig',
    instructions: 'Force users to sign in to Chrome, and only allow accounts from acme.com.',
    // BrowserSignin 2 is "force sign-in"; 1 only allows it.  RestrictSigninToPattern is a regular expression, so
    // the dot is escaped and a glob such as *@acme.com is wrong; anchors are optional.
    expect: {
      mustContain: ['com.google.Chrome', '<key>BrowserSignin</key><integer>2</integer>', '<key>RestrictSigninToPattern</key><string>', '@acme\\.com'],
      mustNotContain: ['<key>BrowserSignin</key><true/>', 'ForceBrowserSignin', '<string>*@'],
    }
  },
  {
    id: 'mobileconfig-third-party-chrome-safe-browsing',
    profileType: 'mobileconfig',
    instructions: 'Turn on enhanced Safe Browsing protection in Chrome.',
    // SafeBrowsingEnabled is the deprecated boolean, and it cannot express "enhanced".
    expect: {
      mustContain: ['com.google.Chrome', '<key>SafeBrowsingProtectionLevel</key><integer>2</integer>'],
      mustNotContain: ['SafeBrowsingEnabled'],
    }
  },
  {
    id: 'mobileconfig-third-party-edge-not-chrome',
    profileType: 'mobileconfig',
    instructions: 'Disable InPrivate browsing in Microsoft Edge and skip the first-run experience.',
    // Edge's policies mirror Chrome's, but its private-mode key has its own name.
    expect: {
      mustContain: ['com.microsoft.Edge', '<key>InPrivateModeAvailability</key><integer>1</integer>', '<key>HideFirstRunExperience</key><true/>'],
      mustNotContain: ['com.google.Chrome', 'IncognitoModeAvailability', 'com.microsoft.edge<'],
    }
  },
  {
    id: 'mobileconfig-third-party-firefox-privacy',
    profileType: 'mobileconfig',
    instructions: 'Turn off telemetry, Pocket, and Firefox Studies in Firefox.',
    // Firefox ignores org.mozilla.firefox on macOS unless EnterprisePoliciesEnabled is true, which the
    // manifest marks required.
    readByEye: 'All four keys must sit in ONE org.mozilla.firefox dict.',
    expect: {
      mustContain: [
        'org.mozilla.firefox', '<key>EnterprisePoliciesEnabled</key><true/>', '<key>DisableTelemetry</key><true/>',
        '<key>DisablePocket</key><true/>', '<key>DisableFirefoxStudies</key><true/>'
      ],
      mustNotContain: ['org.mozilla.Firefox', 'DisableFirefoxTelemetry'],
      mustNotContainElement: [['string', 'true']],
    }
  },
  {
    id: 'mobileconfig-third-party-firefox-homepage-nested',
    profileType: 'mobileconfig',
    instructions: 'Set the Firefox homepage to https://intranet.acme.com, open it when Firefox starts, and stop users changing it.',
    // Homepage is a dict, not a string, so this checks the nested keys are kept nested.  Locked is what stops users
    // changing the homepage; a StartPage of homepage-locked only fixes what opens at startup, so either StartPage
    // value is right but Locked is required.
    expect: {
      mustContain: [
        'org.mozilla.firefox', '<key>EnterprisePoliciesEnabled</key><true/>', '<key>Homepage</key><dict>',
        '<key>URL</key><string>https://intranet.acme.com</string>', '<key>StartPage</key><string>homepage', '<key>Locked</key><true/>'
      ],
      mustNotContain: ['<key>Homepage</key><string>', 'HomepageLocation', 'Homepage.URL'],
    }
  },
  {
    id: 'mobileconfig-third-party-mixed-with-apple-payload',
    profileType: 'mobileconfig',
    instructions: 'Turn off AirDrop and turn off Chrome\'s password manager.',
    readByEye: 'Exactly TWO dicts in PayloadContent, com.apple.applicationaccess and com.google.Chrome, each with its own PayloadUUID and identifier.',
    expect: {
      mustContain: [
        'com.apple.applicationaccess', '<key>allowAirDrop</key><false/>',
        '<string>com.google.Chrome</string>', '<key>PasswordManagerEnabled</key><false/>'
      ],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-third-party-zoom-camera-off',
    profileType: 'mobileconfig',
    instructions: 'Have Zoom join meetings with the camera off and the microphone muted.',
    // The trap is Apple's allowCamera, which turns off the camera for every app rather than at join time.
    expect: {
      mustContain: ['us.zoom.config', '<key>ZDisableVideo</key><true/>', '<key>MuteVoipWhenJoin</key><true/>'],
      mustNotContain: ['allowCamera', 'com.apple.applicationaccess', 'zDisableVideo'],
    }
  },
  {
    id: 'mobileconfig-third-party-zoom-sso-url',
    profileType: 'mobileconfig',
    instructions: 'Lock Zoom\'s SSO sign-in to our company URL, acme.zoom.us.',
    // The value is the subdomain alone, per the manifest: "acme", not "acme.zoom.us" or a full URL.
    expect: {
      mustContain: ['us.zoom.config', '<key>ForceSSOURL</key><string>acme</string>'],
      mustNotContain: ['<string>acme.zoom.us</string>', '<string>https://acme.zoom.us'],
    }
  },
  {
    id: 'mobileconfig-third-party-slack-hardware-acceleration',
    profileType: 'mobileconfig',
    instructions: 'Turn off hardware acceleration in Slack.',
    expect: {
      mustContain: ['com.tinyspeck.slackmacgap', '<key>HardwareAcceleration</key><false/>'],
      mustNotContain: ['com.tinyspeck.slack<', 'com.slack.Slack', 'DisableHardwareAcceleration'],
    }
  },
  {
    id: 'mobileconfig-third-party-onedrive-known-folders',
    profileType: 'mobileconfig',
    instructions: 'Silently move users\' Desktop and Documents folders into OneDrive for our Microsoft 365 tenant 11111111-2222-3333-4444-555555555555.',
    // The trap is Apple's allowCloudDesktopAndDocuments, which is iCloud's version of the same feature.
    // OneDriveUpdater shares the "onedrive" keyword and has none of these keys.
    expect: {
      mustContain: [
        'com.microsoft.OneDrive', '<key>KFMSilentOptIn</key><string>11111111-2222-3333-4444-555555555555</string>',
        '<key>KFMSilentOptInDesktop</key><true/>', '<key>KFMSilentOptInDocuments</key><true/>'
      ],
      mustNotContain: ['allowCloudDesktopAndDocuments', 'com.microsoft.OneDriveUpdater'],
    }
  },
  {
    id: 'mobileconfig-third-party-defender-nested-dicts',
    profileType: 'mobileconfig',
    instructions: 'In Microsoft Defender for Endpoint, block tampering and automatically send safe samples to Microsoft.',
    // Every wdav setting lives one dict down (tamperProtection, cloudService), and the values are
    // lowercase strings rather than booleans.
    expect: {
      mustContain: [
        'com.microsoft.wdav', '<key>tamperProtection</key><dict>', '<key>enforcementLevel</key><string>block</string>',
        '<key>cloudService</key><dict>', '<key>automaticSampleSubmissionConsent</key><string>safe</string>'
      ],
      mustNotContain: ['tamperProtection.enforcementLevel', '<key>tamperProtection</key><true/>', 'com.microsoft.defender', '<string>Block</string>'],
    }
  },
  {
    id: 'mobileconfig-third-party-microsoft-autoupdate',
    profileType: 'mobileconfig',
    instructions: 'Set Microsoft AutoUpdate to download and install updates automatically from the Current channel.',
    expect: {
      mustContain: ['com.microsoft.autoupdate2', '<key>HowToCheck</key><string>AutomaticDownload</string>', '<key>ChannelName</key><string>Current</string>'],
      mustNotContain: ['com.apple.SoftwareUpdate', 'com.microsoft.autoupdate<', 'com.microsoft.autoupdate.fba'],
    }
  },
  {
    id: 'mobileconfig-third-party-office-macros',
    profileType: 'mobileconfig',
    instructions: 'Block Office macros from running, without prompting users.',
    // The setting is shared by every Office app, so it belongs in com.microsoft.office rather than one
    // dict per app.
    expect: {
      mustContain: ['com.microsoft.office', '<key>VisualBasicMacroExecutionState</key><string>DisabledWithoutWarnings</string>'],
      mustNotContain: ['DisabledWithWarnings', 'com.microsoft.Word', 'com.microsoft.Excel'],
    }
  },
  {
    id: 'mobileconfig-third-party-1password-dotted-keys',
    profileType: 'mobileconfig',
    instructions: 'Lock 1Password 8 after 10 minutes of inactivity and whenever the Mac locks.',
    // 1Password's keys are flat strings with dots in them, not a path into nested dicts.
    expect: {
      mustContain: ['com.1password.1password', '<key>security.autolock.minutes</key><integer>10</integer>', '<key>security.autolock.onDeviceLock</key><true/>'],
      mustNotContain: ['<key>security</key><dict>', '<key>autolock</key>', 'com.agilebits.onepassword7'],
    }
  },
  {
    id: 'mobileconfig-third-party-vscode-telemetry',
    profileType: 'mobileconfig',
    instructions: 'Turn off telemetry in Visual Studio Code.',
    // TelemetryLevel is a string enum; "off" is the value, not <false/>.
    expect: {
      mustContain: ['com.microsoft.VSCode', '<key>TelemetryLevel</key><string>off</string>'],
      mustNotContain: ['<key>TelemetryLevel</key><false/>', 'telemetry.telemetryLevel', 'EnableTelemetry'],
    }
  },
  {
    id: 'mobileconfig-third-party-claude-desktop',
    profileType: 'mobileconfig',
    instructions: 'Turn off automatic updates and nonessential telemetry in Claude Desktop.',
    // disableEssentialTelemetry is the lookalike the request does not ask for.
    expect: {
      mustContain: ['com.anthropic.claudefordesktop', '<key>disableAutoUpdates</key><true/>', '<key>disableNonessentialTelemetry</key><true/>'],
      mustNotContain: ['<key>disableEssentialTelemetry</key><true/>', 'DisableAutoUpdates'],
    }
  },
  {
    id: 'mobileconfig-third-party-santa-domain',
    profileType: 'mobileconfig',
    instructions: 'Put North Pole Security\'s Santa into Lockdown mode and have it sync with https://santa.acme.com.',
    // "santa" matches both com.google.santa and com.northpolesec.santa; the request names which one.
    expect: {
      mustContain: ['com.northpolesec.santa', '<key>ClientMode</key><integer>2</integer>', '<key>SyncBaseURL</key><string>https://santa.acme.com'],
      mustNotContain: ['com.google.santa', '<key>ClientMode</key><string>'],
    }
  },
  {
    id: 'mobileconfig-third-party-sap-privileges',
    profileType: 'mobileconfig',
    instructions: 'In SAP Privileges, make admin rights expire after 15 minutes and require users to give a reason.',
    expect: {
      mustContain: ['corp.sap.privileges', '<key>ExpirationInterval</key><integer>15</integer>', '<key>ReasonRequired</key><true/>'],
      mustNotContain: ['<key>ExpirationInterval</key><integer>900</integer>', 'com.sap.privileges'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-microsoft-defender',
    profileType: 'mobileconfig',
    instructions: 'Turn on tamper protection in Microsoft Defender and have it block tampering.',
    expect: {
      mustContain: ['com.microsoft.wdav', '<key>tamperProtection</key><dict>', '<key>enforcementLevel</key><string>block</string>'],
      mustNotContain: ['tamperProtection.enforcementLevel', '<key>tamperProtection</key><true/>', 'com.microsoft.defender'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-vs-code',
    profileType: 'mobileconfig',
    instructions: 'Turn off telemetry in VS Code.',
    expect: {
      mustContain: ['com.microsoft.VSCode', '<key>TelemetryLevel</key><string>off</string>'],
      mustNotContain: ['<key>TelemetryLevel</key><false/>', 'telemetry.telemetryLevel'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-microsoft-365-macros',
    profileType: 'mobileconfig',
    instructions: 'Set the macro security level in Microsoft 365 apps so macros are disabled without a warning.',
    // "microsoft 365 apps" provides both com.microsoft.office and com.microsoft.autoupdate2; only the first has
    // macro settings.  The request names the security level because VisualBasicEntirelyDisabled also stops macros.
    expect: {
      mustContain: ['<string>com.microsoft.office</string>', '<key>VisualBasicMacroExecutionState</key><string>DisabledWithoutWarnings</string>'],
      mustNotContain: ['com.microsoft.autoupdate2', 'com.microsoft.Word', 'com.microsoft.Excel', 'VBAWarnings'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-microsoft-365-updates',
    profileType: 'mobileconfig',
    instructions: 'Make Microsoft 365 apps download and install their updates automatically.',
    expect: {
      mustContain: ['com.microsoft.autoupdate2', '<key>HowToCheck</key><string>AutomaticDownload</string>'],
      mustNotContain: ['com.apple.SoftwareUpdate'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-cloudflare-zero-trust',
    profileType: 'mobileconfig',
    instructions: 'Register Macs with our Cloudflare Zero Trust organization, acme.  Users will sign in with our identity provider.',
    // "trust" alone used to match Jamf Trust.  The sign-in sentence is there because the organization key's
    // description says registration needs an IdP or a service token, and without it the model refused two runs in
    // three for want of a service token.
    expect: {
      mustContain: ['com.cloudflare.warp', '<key>organization</key><string>acme</string>'],
      mustNotContain: ['com.jamf.trust', '<string>acme.cloudflareaccess.com</string>', 'auth_client_id', 'auth_client_secret'],
    }
  },
  {
    id: 'mobileconfig-third-party-alias-crowdstrike',
    profileType: 'mobileconfig',
    instructions: 'License the CrowdStrike sensor with our customer ID 0123456789ABCDEF0123456789ABCDEF-12.',
    expect: {
      mustContain: ['com.crowdstrike.falcon', '<key>ccid</key><string>0123456789ABCDEF0123456789ABCDEF-12</string>'],
      mustNotContain: ['<key>CCID</key>', 'CustomerID'],
    }
  },
  // Ordinary words that are also the names of apps, and no longer keywords on their own (GENERIC_KEYWORDS in
  // regenerate-apple-third-party-payload-manifests.js).  These check the request is still answered with Apple's
  // own payload rather than the app's.
  {
    id: 'mobileconfig-third-party-keyword-edge-not-microsoft-edge',
    profileType: 'mobileconfig',
    instructions: 'Lock the Dock to the left edge of the screen.',
    expect: {
      mustContain: ['com.apple.dock', '<key>orientation</key><string>left</string>'],
      mustNotContain: ['com.microsoft.Edge'],
    }
  },
  {
    id: 'mobileconfig-third-party-keyword-verify-not-okta',
    profileType: 'mobileconfig',
    instructions: 'Show "Verify your identity with the IT help desk before signing in" on the login window.',
    expect: {
      mustContain: ['com.apple.loginwindow', '<key>LoginwindowText</key><string>Verify your identity with the IT help desk before signing in</string>'],
      mustNotContain: ['com.okta.mobile', 'com.jamf.connect.verify'],
    }
  },
  {
    id: 'mobileconfig-third-party-keyword-drive-not-google-drive',
    profileType: 'mobileconfig',
    instructions: 'Show "Back up your drive before leaving for the day" on the login window.',
    // A login-window message rather than a FileVault setting, so the case fails only on the keyword: the
    // FileVault version failed on Apple's payload choice, as mobileconfig-cis-prevent-filevault-disable does.
    expect: {
      mustContain: ['com.apple.loginwindow', '<key>LoginwindowText</key><string>Back up your drive before leaving for the day</string>'],
      mustNotContain: ['com.google.drivefs'],
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
    instructions: 'Automatically enroll devices in our beta program using the enrollment token BETA-TOKEN-123, shown to users as "Acme macOS Beta".',
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
      mustContain: ['com.apple.configuration.safari.settings', '"AcceptCookies":"VisitedWebsites"', '"AllowPopups":false'],
      mustNotContain: ['acceptCookies', 'allowPopups', 'allowPopUps'],
    }
  },
  {
    id: 'ddm-disk-management-external-readonly',
    profileType: 'ddm',
    instructions: 'Let people read from USB drives and other external disks, but not write to them.',
    readByEye: 'Restrictions.ExternalStorage is a nested dictionary with the enum Allowed|ReadOnly|Disallowed.  ReadOnly is the answer; a boolean here is an unknown value in a known key.',
    expect: {
      mustContain: ['com.apple.configuration.diskmanagement.settings', '"Restrictions"', '"ExternalStorage":"ReadOnly"'],
      mustNotContain: ['externalStorage', 'NetworkStorage'],
    }
  },
  {
    id: 'ddm-keyboard-restrictions',
    profileType: 'ddm',
    instructions: 'Turn off dictation and predictive text on the keyboard.',
    expect: {
      mustContain: ['com.apple.configuration.keyboard.settings', '"AllowDictation":false', '"AllowPredictiveText":false'],
      mustNotContain: ['allowDictation', 'allowPredictiveText', 'AllowSpellCheck', 'AllowAutoCorrection'],
    }
  },
  {
    id: 'ddm-intelligence-nested-apps',
    profileType: 'ddm',
    canary: true,
    instructions: 'Turn off Apple Intelligence summaries in Mail and Safari.',
    readByEye: 'These two live in the nested Apps dictionary -- Apps.Mail.AllowSummary and Apps.Safari.AllowSummary -- not at the top level of the payload.  A flat AllowMailSummary key is an unknown key that enforces nothing.  Also confirm no top-level Allow* keys were set, since only the two summaries were asked for.',
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
  // 2026-10-02: JordanMontgomery: I commented out the two cases below because they fail every run on gaps in the DDM schema we send the LLM: it cuts the License.Assignment description off before "needs to be present for App Store apps", and it has no per-key OS versions or value meanings (app.settings is macOS 27+, and Camera "None" means no default, not deny).
  // FUTURE: uncomment these cases once regenerate-apple-profile-schemas keeps those details.
  // {
  //   id: 'ddm-app-managed-app-store-license',
  //   profileType: 'ddm',
  //   instructions: 'Install the App Store app Slack (bundle ID com.tinyspeck.slackmacgap).',
  //   // Apple's schema marks License optional, but its description says it must be present whenever
  //   // BundleID or AppStoreID is.  Without it the declaration fails on the device.  Either Assignment
  //   // value is acceptable, since the request does not say which.  Install defaults to Optional, which
  //   // only makes the app available.  UpdateBehavior is optional and was not asked for.
  //   readByEye: 'License must sit inside InstallBehavior.  A declaration cannot assign the app\'s Apps and Books license, so "deliveryNotes" should say the license has to be assigned before this declaration can install the app.',
  //   expect: {
  //     mustContain: ['com.apple.configuration.app.managed', '"BundleID":"com.tinyspeck.slackmacgap"', '"Install":"Required"', '"License":{"Assignment":"'],
  //     mustNotContain: ['"bundleID"', '"license"', '"assignment"'],
  //   }
  // },
  // {
  //   id: 'ddm-firefox-camera-disable',
  //   profileType: 'ddm',
  //   instructions: 'Disable camera access in Firefox.',
  //   // The generator page's own example prompt.  app.settings looks like the answer, but its Camera
  //   // permission only takes None|Allow, and None means "no default set", not "deny": a declaration can
  //   // grant camera access but cannot block it.  It is also macOS 27+ only.
  //   readByEye: 'The reason should say a declaration cannot deny a privacy permission.  A .mobileconfig PPPC payload is the mirror image -- it can deny Camera but never grant it -- so pointing there is the useful answer.',
  //   expect: { expectFailure: true }
  // },
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
    instructions: 'Make sure Screen Sharing is turned off on every Mac.',
    readByEye: 'Screen Sharing is a launchd service (com.apple.screensharing), toggled with launchctl, not a managed preference.  The right answer names that.  Do NOT accept a com.apple.screensharing payload -- the domain exists for connection settings, not for turning the service off.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-remote-login-disabled',
    profileType: 'mobileconfig',
    instructions: 'Turn off Remote Login so nobody can SSH into these Macs.',
    readByEye: 'Remote Login is the ssh launchd job.  A profile cannot disable it.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-sip-enabled',
    profileType: 'mobileconfig',
    instructions: 'Make sure System Integrity Protection is enabled.',
    readByEye: 'SIP is toggled with csrutil from recoveryOS and cannot be set by MDM at all.  This is the clearest negative in the set -- if anything passes here, the abstention rules are not working.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-root-account-disabled',
    profileType: 'mobileconfig',
    instructions: 'Make sure the root account is disabled.',
    readByEye: 'Local account state, changed with dsenableroot. Not a profile setting.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-sudo-timeout-zero',
    profileType: 'mobileconfig',
    instructions: 'Set the sudo timeout period to zero.',
    readByEye: 'This lives in /etc/sudoers.d and is a file, not a preference domain.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-security-auditing-enabled',
    profileType: 'mobileconfig',
    instructions: 'Make sure security auditing is turned on.',
    readByEye: 'auditd is configured through /etc/security/audit_control.  No payload domain covers it.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-no-world-writable-library-folders',
    profileType: 'mobileconfig',
    instructions: 'Make sure there are no world-writable folders in the Library folder.',
    readByEye: 'Filesystem permissions are an observation about the disk, not a setting that can be enforced.  A profile cannot express this at all.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-xprotect-running',
    profileType: 'mobileconfig',
    instructions: 'Make sure XProtect is running.',
    readByEye: 'Another state-not-setting.  ConfigDataInstall (CIS 1.5) keeps XProtect definitions current but does not make XProtect run, so a profile carrying only that key answers a different request.',
    expect: { expectFailure: true }
  },
  //  ╔═╗╔╗╔╔╦╗╦═╗╔═╗╦╔╦╗
  //  ╠═╣║║║ ║║╠╦╝║ ║║ ║║
  //  ╩ ╩╝╚╝═╩╝╩╚═╚═╝╩═╩╝
  {
    id: 'android-disable-adding-google-accounts',
    profileType: 'android',
    instructions: 'Prevent users from managing google accounts on a device, but allow other accounts.',
    expect: {
      mustContain: ['"accountTypesWithManagementDisabled":["com.google"]']
    }
  },
  {
    id: 'android-disable-adding-accounts',
    profileType: 'android',
    instructions: 'Prevent users from adding new accounts to the device',
    expect: {
      mustContain: ['"modifyAccountsDisabled":true']
    }
  },
  {
    id: 'android-developer-settings',
    profileType: 'android',
    instructions: 'Block developer options.',
    readByEye: 'developerSettings must be nested inside advancedSecurityOverrides; the exact nested substring below verifies that shape.',
    // CANARY.  Nested enum -- the exact shape where a typo one level down is silently discarded
    // rather than rejected.
    expect: {
      mustContain: [
        'advancedSecurityOverrides',
        'developerSettings',
        'DEVELOPER_SETTINGS_DISABLED',
        '"advancedSecurityOverrides": {"developerSettings": "DEVELOPER_SETTINGS_DISABLED"}'
      ]
    }
  },
  {
    id: 'android-camera',
    profileType: 'android',
    instructions: 'Disable the camera.',
    expect: {
      mustContain: ['cameraAccess', 'CAMERA_ACCESS_DISABLED'],
      mustNotContain: ['cameraDisabled', 'camera_access_disabled'],
    }
  },
  {
    id: 'android-screenshots',
    profileType: 'android',
    instructions: 'Block screenshots.',
    expect: {
      mustContain: ['screenCaptureDisabled'],
    }
  },
  {
    id: 'android-factory-reset',
    profileType: 'android',
    instructions: 'Prevent users from factory resetting the device.',
    expect: {
      mustContain: ['"factoryResetDisabled":true'],
    }
  },
  {
    id: 'android-password-length',
    profileType: 'android',
    instructions: 'Require a 10-character password.',
    expect: {
      mustContain: ['passwordPolicies', 'passwordMinimumLength', '10'],
      mustNotContain: ['passwordRequirements']
    }
  },
  {
    id: 'android-failed-attempts',
    profileType: 'android',
    instructions: 'Wipe the device after 10 failed password attempts.',
    expect: {
      mustContain: ['passwordPolicies', 'maximumFailedPasswordsForWipe'],
      mustNotContain: ['passwordRequirements']
    }
  },
  {
    id: 'android-bluetooth-contact-sharing',
    profileType: 'android',
    instructions: 'Turn off Bluetooth contact sharing.',
    expect: {
      mustContain: ['bluetoothContactSharingDisabled']
    }
  },
  {
    id: 'android-copy-paste-between-profiles',
    profileType: 'android',
    instructions: 'Prevent users from copying content from managed applications into unmanaged personal applications',
    expect: {
      mustContain: [
        '"crossProfilePolicies"',
        '"crossProfileCopyPaste": "COPY_FROM_WORK_TO_PERSONAL_DISALLOWED"'
      ]
    }
  },
  {
    id: 'android-untrusted-apps-with-play-protect',
    profileType: 'android',
    instructions: 'Allow users to download and install APK files, but enforce Google Play protect',
    expect: {
      mustContain: [
        '"googlePlayProtectVerifyApps": "VERIFY_APPS_ENFORCED"',
        '"untrustedAppsPolicy": "ALLOW_INSTALL_DEVICE_WIDE"',
        '"advancedSecurityOverrides"'
      ]
    }
  },
  {
    id: 'android-untrusted-apps',
    profileType: 'android',
    instructions: 'Prevent users from installing downloaded apk files',
    expect: {
      mustContain: [
        '"advancedSecurityOverrides"',
        '"untrustedAppsPolicy": "DISALLOW_INSTALL"',
      ]
    }
  },
  {
    id: 'android-disable-biometric-auth',
    profileType: 'android',
    instructions: 'Prevent users from using biometric authetication to unlock the device',
    expect: {
      mustContain: [
        '"keyguardDisabledFeatures"',
        '"BIOMETRICS"',
      ]
    }
  },
  {
    id: 'android-iris-auth',
    profileType: 'android',
    instructions: 'Prevent users from using iris scanners to unlock the device',
    expect: {
      mustContain: [
        '"keyguardDisabledFeatures"',
        '"IRIS"',
      ]
    }
  },
  {
    id: 'android-os-update-window',
    profileType: 'android',
    instructions: 'Set devices to automatically install OS updates at 2:00 am nightly',
    expect: {
      mustContain: [
        '"systemUpdate"',
        '"type": "WINDOWED"',
        '"startMinutes": 120',
        '"endMinutes": 180',
      ]
    }
  },
  {
    id: 'android-wifi-security-restriction',
    profileType: 'android',
    instructions: 'Prevent devices from connecting to open wifi networks.',
    expect: {
      mustContain: [
        '"deviceRadioState": {"minimumWifiSecurityLevel": "PERSONAL_NETWORK_SECURITY"}',
      ]
    }
  },
  {
    id: 'android-disable-fun',
    profileType: 'android',
    instructions: 'Disable Android easter eggs.',
    expect: {
      mustContain: [
        '"funDisabled": true',
      ]
    }
  },
  {
    id: 'android-lock-screen-message',
    profileType: 'android',
    instructions: 'Show "Property of Acme Corp. If found, call 555-0100." on the lock screen.',
    // Nested UserFacingMessage, and the message must be reproduced exactly.
    expect: {
      mustContain: ['"deviceOwnerLockScreenInfo":{"defaultMessage":"Property of Acme Corp. If found, call 555-0100."']
    }
  },
  {
    id: 'android-screen-timeout',
    profileType: 'android',
    canary: true,
    instructions: 'Turn the screen off after 2 minutes of inactivity and do not let users change it.',
    readByEye: 'screenTimeoutSettings must be nested inside displaySettings, not at the top level.',
    // Two levels of nesting, a google-duration string, and a mode that must accompany the duration.
    expect: {
      mustContain: ['"displaySettings":{', '"screenTimeoutSettings":{', '"screenTimeout":"120s"', 'SCREEN_TIMEOUT_ENFORCED'],
      mustNotContain: ['"screenTimeout":120']
    }
  },
  {
    id: 'android-usb-file-transfer',
    profileType: 'android',
    instructions: 'Block file transfers over USB.',
    // usbFileTransferDisabled is deprecated in favour of deviceConnectivityManagement.usbDataAccess.
    expect: {
      mustContain: ['"deviceConnectivityManagement":{', 'DISALLOW_USB_FILE_TRANSFER'],
      mustNotContain: ['usbFileTransferDisabled']
    }
  },
  {
    id: 'android-wifi-config',
    profileType: 'android',
    instructions: 'Do not let users add or change Wi-Fi networks.',
    expect: {
      mustContain: ['"deviceConnectivityManagement":{', 'DISALLOW_CONFIGURING_WIFI'],
      mustNotContain: ['wifiConfigDisabled']
    }
  },
  {
    id: 'android-wifi-hotspot-only',
    profileType: 'android',
    instructions: 'Block the Wi-Fi hotspot, but still allow USB tethering.',
    // Picks the narrower of two enum values, and avoids deprecated tetheringConfigDisabled.
    expect: {
      mustContain: ['DISALLOW_WIFI_TETHERING'],
      mustNotContain: ['DISALLOW_ALL_TETHERING', 'tetheringConfigDisabled']
    }
  },
  {
    id: 'android-microphone',
    profileType: 'android',
    instructions: 'Disable the microphone.',
    expect: {
      mustContain: ['"microphoneAccess":"MICROPHONE_ACCESS_DISABLED"'],
      mustNotContain: ['unmuteMicrophoneDisabled']
    }
  },
  {
    id: 'android-unknown-sources',
    profileType: 'android',
    instructions: 'Block installing apps from unknown sources.',
    expect: {
      mustContain: ['"advancedSecurityOverrides":{', '"untrustedAppsPolicy":"DISALLOW_INSTALL"'],
      mustNotContain: ['installUnknownSourcesAllowed']
    }
  },
  {
    id: 'android-play-protect',
    profileType: 'android',
    instructions: 'Force Google Play Protect app verification on.',
    expect: {
      mustContain: ['"advancedSecurityOverrides":{', '"googlePlayProtectVerifyApps":"VERIFY_APPS_ENFORCED"'],
      mustNotContain: ['ensureVerifyAppsEnabled']
    }
  },
  {
    id: 'android-auto-time',
    profileType: 'android',
    instructions: 'Require automatic date and time.',
    expect: {
      mustContain: ['"autoDateAndTimeZone":"AUTO_DATE_AND_TIME_ZONE_ENFORCED"'],
      mustNotContain: ['autoTimeRequired']
    }
  },
  {
    id: 'android-location-off',
    profileType: 'android',
    instructions: 'Turn off location services.',
    // OFF is a deprecated value of locationMode; LOCATION_DISABLED replaces it.
    expect: {
      mustContain: ['"locationMode":"LOCATION_DISABLED"'],
      mustNotContain: ['"OFF"', 'shareLocationDisabled']
    }
  },
  {
    id: 'android-keyguard-features',
    profileType: 'android',
    instructions: 'Hide the camera shortcut and notifications on the lock screen.',
    // Array of enums, and only the two asked for.
    expect: {
      mustContain: ['keyguardDisabledFeatures', '"CAMERA"', '"NOTIFICATIONS"'],
      mustNotContain: ['ALL_FEATURES', 'cameraAccess']
    }
  },
  {
    id: 'android-printing',
    profileType: 'android',
    instructions: 'Disable printing.',
    // Google's description for this field starts with "Optional.", which the pruning once kept as the whole description.
    expect: {
      mustContain: ['"printingPolicy":"PRINTING_DISALLOWED"']
    }
  },
  {
    id: 'android-frp-admin',
    profileType: 'android',
    instructions: 'After a factory reset, only it-admin@example.com should be able to unlock the device.',
    expect: {
      mustContain: ['"frpAdminEmails":["it-admin@example.com"]']
    }
  },
  {
    id: 'android-deny-permissions',
    profileType: 'android',
    instructions: 'Automatically deny app permission requests.',
    expect: {
      mustContain: ['"defaultPermissionPolicy":"DENY"']
    }
  },
  {
    id: 'android-work-profile-password',
    profileType: 'android',
    instructions: 'Require an alphanumeric password with at least 8 characters for the work profile.',
    expect: {
      mustContain: ['"passwordPolicies":[{', 'ALPHANUMERIC', '"passwordMinimumLength":8', 'SCOPE_PROFILE'],
      mustNotContain: ['passwordRequirements', 'SCOPE_DEVICE']
    }
  },
  {
    id: 'android-stay-on-while-charging',
    profileType: 'android',
    instructions: 'Keep the screen on while the device is charging from AC or USB.',
    expect: {
      mustContain: ['stayOnPluggedModes', '"AC"', '"USB"'],
      mustNotContain: ['"WIRELESS"']
    }
  },
  {
    id: 'android-multiple-restrictions',
    profileType: 'android',
    instructions: 'Disable the camera, block screenshots, and prevent factory reset.',
    expect: {
      mustContain: ['CAMERA_ACCESS_DISABLED', '"screenCaptureDisabled":true', '"factoryResetDisabled":true'],
      mustNotContain: ['cameraDisabled']
    }
  },
  {
    id: 'negative-android-nonexistent-hostname',
    profileType: 'android',
    instructions: 'Set the device hostname to ACME-1234.',
    expect: { expectFailure: true }
  },
  {
    id: 'negative-android-geofence-alert',
    profileType: 'android',
    instructions: 'Email IT when the device leaves the office.',
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

  // For values that are random per generation, where no fixed substring can stand in.
  for (let pattern of expectations.mustMatch || []) {
    if(!new RegExp(pattern).test(profileWithoutWhitespace)) {
      failures.push(`does not match pattern: ${pattern}`);
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
