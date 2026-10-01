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
      mustContain: ['WiFi/Profile/CorpNet/WlanXml', '<![CDATA['],
      mustContainElement: [['name', 'CorpNet'], ['hex', '436F72704E6574'], ['authentication', 'WPA2'], ['useOneX', 'true']],
      mustNotContainElement: [['authentication', 'WPA2PSK'], ['name', 'corpnet'], ['name', 'CORPNET']],
      mustNotContain: ['&lt;WLANProfile', '<SyncML'],
      mustNotContainOutsideCdata: ['<?xml'],
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
    // have to hold in the same profile.
    readByEye: 'Two payload dicts should be present, each with its own PayloadUUID and an identifier suffix.',
    expect: {
      mustContainElement: [['key', 'forcePIN'], ['key', 'minLength'], ['key', 'allowSimple'], ['key', 'AutomaticCheckEnabled']],
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
    readByEye: 'Two dicts, each with a distinct uppercase PayloadUUID and an identifier that is the root identifier plus a suffix.  Duplicate UUIDs install unpredictably and nothing here can detect them.',
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
