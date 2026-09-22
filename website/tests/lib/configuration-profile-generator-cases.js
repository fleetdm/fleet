/**
 * Test cases for the configuration profile generator, plus the checker that scores one generated
 * profile against one case's `expect` block.
 *
 * Two runners share this file, which is why it is a module and not part of either of them:
 *
 *   - `test/configuration-profile-generator.test.js`, the mocha suite: one `it()` per case, one
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
      mustContain: ['LocalPoliciesSecurityOptions/InteractiveLogon_MessageTextForUsersAttemptingToLogOn', 'Authorized users only'],
      mustContainElement: [['Format', 'chr']]
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
      mustContain: ['LoginwindowText', 'Authorized use only']
    }
  },
  {
    id: 'mobileconfig-screensaver',
    profileType: 'mobileconfig',
    instructions: 'Show the "Flurry" screensaver after 10 minutes of inactivity and require a password immediately.',
    expect: {
      mustContain: ['idleTime', 'askForPassword', 'moduleName', '<key>idleTime</key><integer>600</integer>', '<key>askForPassword</key><true/>', '<key>askForPasswordDelay</key><integer>0</integer>'],
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
    instructions: 'Set the Dock to auto-hide and pin it to the left side of the screen.',
    // All-lowercase keys -- fails if the model PascalCases.
    expect: {
      mustContain: ['com.apple.dock'],
      mustContainElement: [['key', 'autohide'], ['key', 'orientation']],
      mustNotContainElement: [['key', 'Autohide'], ['key', 'Orientation']]
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
