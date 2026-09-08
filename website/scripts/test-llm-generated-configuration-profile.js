//  ╔═╗╦═╗╔═╗╔╦╗╔═╗╔╦╗
//  ╠═╝╠╦╝║ ║║║║╠═╝ ║
//  ╩  ╩╚═╚═╝╩ ╩╩   ╩
//
// Copied from api/controllers/get-llm-generated-configuration-profile.js so this script can sweep
// models and effort levels without editing a public endpoint (an endpoint that accepted a model
// override as an input would be an abuse vector).
//
// This is a copy, so it will drift.  Two things make that visible rather than silent:
//   - Every rule below is copied VERBATIM, and --all checks that each one still appears in the
//     action's source, warning on any that no longer do.  Keep them byte-identical or that check
//     stops working.
//   - A rule ADDED to the action cannot be detected that way.  Re-copy when the action changes.
//
// Known faithful copy of a defect: the failure shape's "reasonWhyAProfileCouldNotBeGenerated"
// value is unquoted TODO in the action, so the model is shown one valid and one invalid example
// of the same response.  Copied as-is on purpose -- fixing it here would test a prompt the action
// does not send.

const SHARED_RULES = [
  'Generate a profile that any MDM can deliver.  Use only syntax defined by Apple, Microsoft, or Google -- never a vendor-specific variable, placeholder, or extension, and never Fleet-specific syntax such as $FLEET_SECRET_ or FLEET_VAR_.  A vendor placeholder the delivering MDM does not recognize is shipped to the device as a literal value.',
  'Reproduce user-supplied identifiers character for character, including case: SSIDs, profile names, certificate subjects, domain names.  Never re-capitalize, trim, or reword them.  An SSID differing by one letter\'s case deploys cleanly and matches nothing.',
  'Use only settings you can attribute to a specific published source (an Apple payload key, a Windows CSP node, or an Apple declaration type).  If the instructions cannot be satisfied that way, do not approximate -- return the "couldNotGenerateProfile" shape instead.',
  'You have no network access and cannot open any URL.  Never state or imply that you validated this profile against a reference, a schema, or a linter.  "documentationUrl" is where a human can check your work, not evidence that you checked it.',
  'Enforce only what the instructions ask for.  The only settings you may add beyond the request are ones the requested setting depends on, and each of those must be called out in "caveats".',
  'Write credentials the admin supplied as literals, since the profile is unusable without them.  Do not invent a placeholder.  Note in "deliveryNotes" that the file contains a cleartext credential.',
  'When a platform requires a companion artifact the profile cannot contain -- a DDM activation declaration, a referenced asset declaration -- generate the configuration itself and describe the companion in "deliveryNotes".',
  'If a setting is commonly managed by an MDM directly rather than by a custom profile, such as disk encryption, still generate the profile as asked and note in "deliveryNotes" that some MDMs manage this natively and may reject or conflict with a custom profile.',
  'Escape newlines inside "configurationProfile" as \\n so the surrounding JSON stays valid.  Do not emit raw line breaks inside the string, and do not collapse the profile onto one line.',
];

const DELIVERY_NOTES_RULES = [
  '"deliveryNotes" is for exceptions only: something the admin has to do or decide that is not visible in the profile itself.  Use an empty string when nothing applies.  An empty string is the right answer for most profiles -- prefer it whenever you are unsure whether a note earns its place.',
  'Never write a sentence stating that a condition does not apply.  "No credentials or secrets are embedded" and "no companion declaration is required" are not notes -- leaving them out already says that.',
  'Never restate what the profile is, which platform it targets, or how that platform is normally delivered.  The admin chose the format and already knows.',
  'One sentence per action, addressed to the admin, and no more than two sentences in total.  For example: "Replace the passphrase with a secret variable before committing this to a repository."',
];

const PROMPT_CONFIG_BY_PROFILE_TYPE = {

  'csp': {
    description: 'CSP XML profile that enforces OS settings on Windows devices',
    references: [
      'Windows CSP nodes, formats, and allowed values: https://learn.microsoft.com/en-us/windows/client-management/mdm/',
    ],
    rules: [
      'A Windows profile is a sequence of OMA-DM command elements, not a SyncML document.  The top level must be one or more <Add>, <Replace>, <Exec>, or <Atomic> elements and nothing else.  Never wrap them in <SyncML>, <SyncBody>, <Final/>, or an invented container such as <WindowsCSP>, and never emit an XML declaration -- the SyncML envelope carries session state (SyncHdr, SessionID, MsgID) that only the MDM server can populate at delivery time.',
      'If you use <Atomic>, it must wrap every command in the profile.  Never mix an <Atomic> element with sibling top-level commands, and never nest one <Atomic> inside another.',
      'Emit only LocURI paths that exist in a published CSP.  Never construct, guess, or extrapolate a path, and name the CSP you used in "schemaReference".',
      'Percent-encode any character in a LocURI path segment that is not URI-legal.  An SSID named "Cool Network" is Cool%20Network in the LocURI, while the value inside the payload keeps its literal spaces.',
      'In the Policy CSP, settings that read as boolean are almost always DFFormat int with allowed values 0 and 1, not bool.  Default to <Format>int</Format> with numeric <Data>.  Use bool only when the node genuinely declares it, and say so in "caveats".',
      'Before returning, re-read every Item and confirm <Data> is legal for the <Format> you declared: int accepts digits only, bool accepts literally true or false, chr accepts text.  If you have written 0 or 1 as the data, the format is int, not bool.',
      'Never infer a boolean value from the node name.  DeviceLock/DevicePasswordEnabled takes 0 for enabled and 1 for disabled.  Always state what the value you chose actually does in "valueMeaning".',
      'State the node\'s declared DFFormat verbatim in "allowedValues", next to the format you emitted, so a reviewer can compare the two side by side.',
      'If you cannot recall a node\'s documented DFFormat with confidence, do not guess -- return the "couldNotGenerateProfile" shape and name the node you were unsure about.',
      'Use Replace for Policy CSP leaf nodes that hold a value.  Reserve Add for nodes that do not exist until you create them, such as ADMXInstall, certificate installs, and WiFi or VPN profile instances.  When a node\'s AccessType permits both, choose Replace, because the profile may reach a host where the value is already set and Add can fail there.',
      'Only use an Add or Replace verb on a node whose AccessType permits it.  Roughly 500 leaf nodes accept only Get, and a Replace against one is accepted, deploys, and then fails on the device.',
      'Include the nodes the requested setting depends on.  DeviceLock/MinDevicePasswordLength has no effect unless DeviceLock/DevicePasswordEnabled is also set.',
      'Record the minimum OS build and the Windows editions each node applies to in "caveats".  A valid setting aimed at the wrong SKU is a silent no-op, not an error.',
      'When a node\'s value is embedded XML -- WiFi/Profile/*/WlanXml, ADMXInstall, and similar -- wrap it in <![CDATA[ ... ]]> rather than escaping it as entities, and emit that value as a single line with no line breaks or indentation between its elements.  This applies only to the embedded value; the surrounding SyncML keeps its normal indentation.',
      'Be consistent within a profile about optional <Meta> children.  If you emit <Type> for one item, emit it for all of them, and namespace every <Meta> child as xmlns="syncml:metinf".',
      'For WiFi profiles, emit both <name> and <hex> inside <SSID>, where <hex> is the uppercase hex encoding of the SSID bytes.  Windows and some MDMs treat the hex form as authoritative when both are present.',
    ],
  },

  'mobileconfig': {
    description: 'XML .mobileconfig profile that enforces OS settings on macOS devices',
    references: [
      'First-party Apple payloads: https://github.com/apple/device-management/tree/release/mdm/profiles',
      'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
    ],
    rules: [
      'If this is an attempt to change a third-party application\'s settings, use that application\'s preference domain -- com.google.Chrome, us.zoom.config and its keys must come from the ProfileManifests reference.',
      'Emit valid property list XML: the plist DOCTYPE, plist version="1.0", and correctly typed values.',
      'Include PayloadIdentifier, PayloadType, PayloadUUID, PayloadVersion, and PayloadDisplayName on the root dict and on every dict inside PayloadContent.  The root PayloadType is "Configuration" and PayloadVersion is 1.',
      'Keep the plist indented and readable across multiple lines.  Do not collapse it onto one line.',
      'Apple payload keys are not consistently cased, and the inconsistency is inside a single dict.  The passcode payload uses forcePIN, minLength, and allowSimple -- lowercase first letter -- beside PascalCase PayloadIdentifier and PayloadType.  Reproduce every key exactly as documented for its payload type.  Never normalize casing in either direction.',
      'Use only keys documented for the payload type you chose.  An invented key is written into the profile and nothing downstream rejects it, so the profile looks right and does nothing.',
      'If you cannot recall a payload type\'s exact key names, do not guess a casing and do not adapt a key from a DDM declaration -- return the "couldNotGenerateProfile" shape and name the payload type you were unsure about.',
      'Type every value as plist: <true/> or <false/> for booleans, never <string>true</string>; <integer> for whole numbers; <real> for decimals; <data> with base64 for binary; <date> with an ISO 8601 timestamp.',
      'PayloadUUID is a distinct uppercase UUID in 8-4-4-4-12 form on every dict, including the root.  Two dicts sharing a UUID is a profile that installs unpredictably.',
      'PayloadIdentifier is reverse-DNS.  Each payload dict\'s identifier is the root identifier plus a distinguishing suffix, and no two identifiers in the profile are the same.',
      'PayloadDisplayName on the root is what an end user sees in System Settings, and some MDMs use it as the profile name.  Make it human-readable and specific to what the profile does.',
      'Put every key for one payload domain in a single dict inside PayloadContent.  Do not emit several dicts with the same PayloadType.',
    ],
  },

  'ddm': {
    description: 'Apple DDM declaration in JSON format that enforces OS settings on macOS devices',
    references: [
      'Apple DDM declaration types, keys, and values: https://github.com/apple/device-management/tree/release/declarative/declarations',
    ],
    rules: [
      'The declaration is JSON, not XML.  Include Type, Identifier, and Payload.',
      'Type must be a real declaration type, such as com.apple.configuration.passcode.settings.',
      'Every key inside Payload is PascalCase, with the first letter capitalized: RequirePasscode, MinimumLength, RequireAlphanumericPasscode.  A key with a lowercase first letter is an unknown key -- it is accepted as a typo and enforces nothing.',
      'Declarations do not reuse .mobileconfig payload key names, and the DDM name is not the .mobileconfig name recapitalized.  In the passcode payload, forcePIN becomes RequirePasscode and minLength becomes MinimumLength.  Never carry a .mobileconfig key into a declaration and never transform one into a declaration key.',
      'If you cannot recall a declaration\'s exact Payload key names, do not guess a casing and do not convert a .mobileconfig key -- return the "couldNotGenerateProfile" shape and name the declaration type you were unsure about.',
      'Identifier is your own reverse-DNS identifier for this declaration instance, not a copy of Type.  Copying Type conflates Apple\'s namespace with yours and collides the moment a second declaration of the same type exists.',
      'Derive Identifier from the full declaration type rather than its last component, or passcode.settings and softwareupdate.settings collapse into one identifier and silently overwrite each other.',
      'Keep Identifier to 64 bytes or fewer.  Apple\'s DeclarationBase caps it, and a longer identifier is accepted by an MDM and then rejected by the device at delivery.',
    ],
  },

  // Not yet in the action.  Present here so the Android prompt can be tested before it is wired
  // into a public endpoint; the drift check skips it for that reason.
  'android': {
    notYetInTheAction: true,
    usesPolicySchema: true,
    description: 'Android Management API Policy in JSON format that enforces OS settings on Android devices',
    references: [
      'Android Management API Policy fields, types, and enum values: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies',
    ],
    rules: [
      'The profile is a single JSON object whose top-level keys are Policy fields.  Do not wrap it in an envelope such as {"policy": ...}, and do not add keys that are not Policy fields.',
      'Copy every field name verbatim from the provided schema.  A wrong key at the top level is rejected by name, but the same typo one level down inside an object is silently discarded -- the profile deploys and the setting is simply absent.',
      'Build nested objects along the exact path the schema declares.  When a property\'s value is a "$ref", expand it as a nested object under that key using the referenced schema\'s properties -- never flatten a nested field to the top level or invent an intermediate key.',
      'Copy enum values character for character from the schema\'s "enum" array, including case.  They are SCREAMING_SNAKE_CASE.  Enum values are not checked before the policy reaches Google, so a wrong-case or invented value passes every local check and then fails at delivery.',
      'Match each value to the type the schema declares.  Booleans are JSON booleans, not the strings "true" and "false".  Integers are JSON numbers.  Fields the schema declares as type "string" with format "int64" -- durations and timeouts such as maximumTimeToLock -- must be quoted strings even though they hold a number.',
      'If the schema marks a field deprecated in favour of another, or its description says the field has no effect when another is set, use the replacement.',
      'If the requested setting is not a field in the provided schema, do not approximate with a similarly named field -- return the "couldNotGenerateProfile" shape and name the setting you could not find.',
      'Software management, status reporting, disk encryption, kiosk mode, and setup-experience fields -- applications, statusReportingSettings, encryptionPolicy, kioskCustomization, setupActions, systemUpdate, and similar -- are commonly owned by the MDM rather than by a custom profile.  Still emit them when asked, and note in "deliveryNotes" that some MDMs manage these natively and will reject a profile that sets them.',
    ],
  },

};

// The tail of the action's systemPrompt, verbatim.
const RESPONSE_SHAPE = `Respond in JSON with this data shape:
{
  "configurationProfile": "TODO",
  "profileFilename": "TODO",
  // Things the admin must do or decide that are not visible in the profile itself.
  // Empty string when there is nothing exceptional, which is the common case.
  "deliveryNotes": "",
  "settingsEnforced": [// For each setting enforced by the configuration profile.
    {
      // The name (key) of the setting that is enforced. e.g., LoginwindowText
      name: "TODO",
      // The value of the setting that is enforced
      value: "TODO",
      // Where this setting comes from: the CSP node path, the Apple payload domain and key, or the declaration type.
      schemaReference: "TODO",
      // The documented range, enum, or type this setting accepts, including the declared format.
      allowedValues: "TODO",
      // What the value above actually does, in words. e.g., "0 = a password is required"
      valueMeaning: "TODO",
      // The Apple or Microsoft reference page for this setting.
      documentationUrl: "TODO",
      // Applicability, dependencies, and any condition under which this setting deploys but does nothing. Empty string if there are none.
      caveats: "TODO"
    },
    {...}
  ]
}

If a configuration profile cannot be generated from the provided instructions, respond with this shape instead:
{
  "couldNotGenerateProfile": true,
  // Explain why a profile could not be generated, naming the specific setting, node, or key that could not be confirmed. The tone should be informational and brief.
  "reasonWhyAProfileCouldNotBeGenerated": TODO
}
`;


//  ╔═╗╔═╗╔═╗╔═╗╔═╗
//  ║  ╠═╣╚═╗║╣ ╚═╗
//  ╚═╝╩ ╩╚═╝╚═╝╚═╝
//
// `mustContain` / `mustNotContain` are compared with all whitespace stripped from both sides,
// because the JSON formats have unpredictable indentation and an assertion like
// `"maximumTimeToLock": "300000"` would otherwise fail on formatting alone.
//
// Assertions are substring-only by design.  Anything cleverer would be a validator, and the design
// decision for this generator was that there isn't one -- the CANARY cases are written as exact
// substrings so this can stay dumb.  Properties that genuinely can't be checked this way (one dict
// per payload domain, distinct PayloadUUIDs, single-line CDATA) are noted on the case and stay a
// human read.
const TEST_CASES = [

  //  ╔═╗╔═╗╔═╗
  //  ║  ╚═╗╠═╝
  //  ╚═╝╚═╝╩
  {
    id: 'csp-device-password',
    profileType: 'csp',
    canary: true,
    instructions: 'Require a password to unlock the device.',
    readByEye: 'valueMeaning should say 0 means a password IS required.  If it describes 0 as disabling the password, the value is right by luck and the reasoning is wrong -- which will not hold on the next node.',
    // CANARY.  Three traps in one three-word request: the inverted value (0 means enabled), the
    // format (int, not bool), and the document shape (no SyncML envelope, no XML declaration).
    expect: {
      mustContain: ['DeviceLock/DevicePasswordEnabled'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['<SyncML', '<?xml'],
      deliveryNotes: ''
    }
  },
  {
    id: 'csp-removable-storage',
    profileType: 'csp',
    instructions: 'Block write access to removable storage.',
    expect: {
      mustContain: ['RemovableDiskDenyWriteAccess'],
      mustContainElement: [['Format', 'int']],
      mustNotContainElement: [['Format', 'bool']],
      deliveryNotes: ''
    }
  },
  {
    id: 'csp-min-password-length',
    profileType: 'csp',
    instructions: 'Require device passwords to be at least 12 characters long.',
    // Length alone is a silent no-op, so the dependency has to come along.
    expect: { mustContain: ['MinDevicePasswordLength', 'DevicePasswordEnabled'], mustContainElement: [['Data', '12']] }
  },
  {
    id: 'csp-failed-attempts',
    profileType: 'csp',
    instructions: 'Wipe the device after 10 failed password attempts.',
    expect: { mustContain: ['MaxDevicePasswordFailedAttempts', 'DevicePasswordEnabled'] }
  },
  {
    id: 'csp-wifi-wpa2-psk-with-spaces',
    profileType: 'csp',
    canary: true,
    instructions: 'Add a wifi profile for a network with the SSID "Cool Network" with WPA2 authentication that uses the password "aaaaaaapassword".',
    readByEye: 'The embedded <WLANProfile> must be on ONE line inside the CDATA -- no assertion can express that.  Also confirm <hex> decodes to the SSID, and that deliveryNotes names the cleartext passphrase rather than describing the profile.',
    // CANARY, and the highest-value case in the set -- it caught four distinct defects across four
    // iterations during design.  Not checkable here: that the embedded WLANProfile is on a single
    // line.  Read that one by eye.
    expect: {
      mustContain: ['Cool%20Network', '<![CDATA[', '436F6F6C204E6574776F726B'],
      mustContainElement: [['name', 'Cool Network']],
      mustNotContain: ['&lt;WLANProfile', '<SyncML', 'Cool%20network'],
      deliveryNotesIsNotEmpty: true
    }
  },
  {
    id: 'csp-script-block-logging',
    profileType: 'csp',
    instructions: 'Turn on PowerShell script block logging.',
    // ADMX-backed, and the worked example in articles/creating-windows-csps.md:130.  The area is
    // WindowsPowerShell even though the backing file is PowerShellExecutionPolicy.admx: most
    // ADMX-backed areas are named ADMX_<AdmxFileName>, this one is not, and deriving the area from
    // the filename yields a real-looking LocURI whose node does not exist under it.
    readByEye: 'Payload should be <![CDATA[<enabled/>]]> with no <data> elements, since no sub-option was requested.  Confirm documentationUrl points at policy-csp-windowspowershell -- a URL that resolves is not the same as a URL that contains the node.',
    expect: {
      mustContain: ['Config/WindowsPowerShell/TurnOnPowerShellScriptBlockLogging', '<![CDATA[', '<enabled/>'],
      mustContainElement: [['Format', 'chr']],
      mustNotContain: ['ADMX_PowerShellExecutionPolicy']
    }
  },
  {
    id: 'csp-logon-banner',
    profileType: 'csp',
    instructions: 'Show "Authorized users only" as a message on the sign-in screen.',
    // Non-int format -- catches a model that defaults everything to int.
    expect: { mustContain: ['InteractiveLogon'], mustContainElement: [['Format', 'chr']] }
  },
  {
    id: 'csp-telemetry',
    profileType: 'csp',
    instructions: 'Set diagnostic data to the lowest level allowed.',
    expect: { mustContain: ['AllowTelemetry'], mustContainElement: [['Format', 'int']] }
  },
  {
    id: 'csp-disk-encryption-natively-managed',
    profileType: 'csp',
    instructions: 'Turn on BitLocker with XTS-AES 256 encryption on the system drive.',
    // Must SUCCEED with a warning, not refuse: legal in Intune, managed natively by some MDMs.
    expect: { deliveryNotesIsNotEmpty: true }
  },
  {
    id: 'csp-nonexistent-email-alert',
    profileType: 'csp',
    instructions: 'Email the security team whenever a user fails to unlock their device three times.',
    expect: { expectFailure: true }
  },

  //  ╔╦╗╔═╗╔╗ ╦╦  ╔═╗╔═╗╔╗╔╔═╗╦╔═╗
  //  ║║║║ ║╠╩╗║║  ║╣ ║  ║║║╠╣ ║║ ╦
  //  ╩ ╩╚═╝╚═╝╩╩═╝╚═╝╚═╝╝╚╝╚  ╩╚═╝
  {
    id: 'mobileconfig-mixed-casing',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Require a 12-character passcode with no simple passcodes, and turn on automatic checking for updates.',
    readByEye: 'Two payload dicts should be present, each with its own PayloadUUID and an identifier suffix.  Confirm the passcode keys kept their lowercase first letter in the SAME profile where AutomaticCheckEnabled kept its capital -- a model can be self-consistently wrong and still pass one of these two checks.',
    // CANARY.  Forces both casing conventions into one profile: a model that normalizes either
    // direction breaks exactly one of the two dicts and leaves the other looking fine.
    expect: {
      mustContain: ['forcePIN', 'minLength', 'allowSimple', 'AutomaticCheckEnabled'],
      mustNotContain: ['ForcePIN', 'MinLength', 'AllowSimple', 'automaticCheckEnabled']
    }
  },
  {
    id: 'mobileconfig-showfullname',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Set the login window to show a list of users instead of name and password fields.',
    readByEye: 'SHOWFULLNAME false is what shows the user list.  Confirm valueMeaning explains that, rather than describing false as turning something off.',
    // CANARY.  All-caps key in the same payload domain as a PascalCase one, and the value is
    // inverted -- false shows the list.
    expect: { mustContain: ['SHOWFULLNAME', '<false/>'], mustNotContainElement: [['string', 'false']] }
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
    expect: { mustContain: ['LoginwindowText', 'Authorized use only'] }
  },
  {
    id: 'mobileconfig-screensaver',
    profileType: 'mobileconfig',
    instructions: 'Lock the screen after 10 minutes of inactivity and require a password immediately.',
    // The unit conversion is the point: idleTime is seconds.
    expect: { mustContain: ['idleTime', 'askForPassword'], mustContainElement: [['integer', '600']] }
  },
  {
    id: 'mobileconfig-firewall',
    profileType: 'mobileconfig',
    instructions: 'Turn on the firewall in stealth mode and block all incoming connections.',
    readByEye: 'All three keys must sit in ONE dict of PayloadType com.apple.security.firewall.  Count the dicts inside PayloadContent.',
    // Not checkable here: that all three keys landed in ONE dict rather than three dicts of the
    // same PayloadType.  Count the <dict> blocks by eye.
    expect: { mustContain: ['EnableFirewall', 'EnableStealthMode', 'com.apple.security.firewall'] }
  },
  {
    id: 'mobileconfig-two-payloads',
    profileType: 'mobileconfig',
    instructions: 'Disable AirDrop and turn off Siri.',
    readByEye: 'Two dicts, each with a distinct uppercase PayloadUUID and an identifier that is the root identifier plus a suffix.  Duplicate UUIDs install unpredictably and nothing here can detect them.',
    // Two payload domains, so two dicts -- which means distinct PayloadUUIDs and distinct
    // identifier suffixes.  Neither is substring-checkable; read them.
    expect: { mustContain: ['PayloadUUID', 'PayloadIdentifier'] }
  },
  {
    id: 'mobileconfig-diagnostics',
    profileType: 'mobileconfig',
    instructions: 'Stop sending diagnostic and usage data to Apple.',
    expect: { mustContain: ['com.apple.SubmitDiagInfo', 'AutoSubmit'] }
  },
  {
    id: 'mobileconfig-dock-lowercase-keys',
    profileType: 'mobileconfig',
    instructions: 'Set the Dock to auto-hide and pin it to the left side of the screen.',
    // All-lowercase keys -- fails if the model PascalCases.
    expect: { mustContain: ['com.apple.dock', 'autohide', 'orientation'], mustNotContain: ['Autohide', 'Orientation'] }
  },
  {
    id: 'mobileconfig-chrome-third-party',
    profileType: 'mobileconfig',
    instructions: 'Block Chrome\'s incognito mode.',
    // PayloadType is the preference domain; keys come from ProfileManifests, not Apple.
    expect: { mustContain: ['com.google.Chrome', 'IncognitoModeAvailability'] }
  },

  //  ╔╦╗╔╦╗╔╦╗
  //   ║║ ║║║║║
  //  ═╩╝═╩╝╩ ╩
  {
    id: 'ddm-passcode-alphanumeric',
    profileType: 'ddm',
    canary: true,
    instructions: 'Require a 10-character alphanumeric passcode and wipe the device after 10 failed attempts.',
    readByEye: 'Identifier must not be a copy of Type, and must be 64 bytes or fewer.  Also confirm RequireAlphanumericPasscode is present -- the assertions only cover RequirePasscode and MinimumLength.',
    // CANARY.  PascalCase payload keys, and no bleed-through from the .mobileconfig names for the
    // same settings.
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', 'RequirePasscode', 'MinimumLength'],
      mustNotContain: ['requirePasscode', 'forcePIN', 'minLength']
    }
  },
  {
    id: 'ddm-beta-enroll',
    profileType: 'ddm',
    instructions: 'Enroll in the public beta program.',
    expect: { mustContain: ['com.apple.configuration.softwareupdate.settings'] }
  },
  {
    id: 'ddm-beta-block',
    profileType: 'ddm',
    instructions: 'Block enrollment in any beta program.',
    // The inverse of the case above.  One of the pair likely maps to an enum rather than a
    // boolean, and the wrong direction validates cleanly and does the opposite.
    expect: { mustContain: ['com.apple.configuration.softwareupdate.settings'] }
  },
  {
    id: 'ddm-defer-minor-updates',
    profileType: 'ddm',
    instructions: 'Defer minor updates by 30 days.',
    expect: { mustContain: ['softwareupdate.settings', '30'] }
  },
  {
    id: 'ddm-auto-install-security-responses',
    profileType: 'ddm',
    instructions: 'Automatically install security responses and system files.',
    expect: { mustContain: ['softwareupdate.settings'] }
  },
  {
    id: 'ddm-enforce-os-version',
    profileType: 'ddm',
    instructions: 'Enforce macOS 26.1 by December 15, 2026 at 6:00 PM local time.',
    // Version quoted as a string; TargetLocalDateTime carries no timezone offset.
    expect: { mustContain: ['softwareupdate.enforcement.specific', 'TargetOSVersion', 'TargetLocalDateTime', '"26.1"'] }
  },
  {
    id: 'ddm-intelligence-off',
    profileType: 'ddm',
    instructions: 'Turn off Apple Intelligence.',
    expect: { mustContain: ['intelligence.settings'] }
  },
  {
    id: 'ddm-intelligence-partial',
    profileType: 'ddm',
    instructions: 'Block Writing Tools and Image Playground but leave the rest of Apple Intelligence available.',
    expect: { mustContain: ['intelligence.settings'] }
  },
  {
    id: 'ddm-migration-assistant',
    profileType: 'ddm',
    instructions: 'Disable Migration Assistant.',
    expect: { mustContain: ['migration-assistant.settings'] }
  },
  {
    id: 'ddm-identifier-collision',
    profileType: 'ddm',
    instructions: 'Create two declarations, one for passcode settings and one for software update settings, and give them both the identifier com.acme.settings.',
    // Complying would collapse two declarations into one identifier and silently overwrite.
    expect: { expectFailure: true }
  },

  //  ╔═╗╔╗╔╔╦╗╦═╗╔═╗╦╔╦╗
  //  ╠═╣║║║ ║║╠╦╝║ ║║ ║║
  //  ╩ ╩╝╚╝═╩╝╩╚═╚═╝╩═╩╝
  // {
  //   id: 'android-max-time-to-lock',
  //   profileType: 'android',
  //   canary: true,
  //   instructions: 'Lock the device after 5 minutes of inactivity.',
  //   readByEye: 'Confirm the quotes around 300000 are in the raw profile itself, not only in the citation.  The assertion strips whitespace, so it cannot tell a quoted string from a cleverly formatted number.',
  //   // CANARY.  The schema declares maximumTimeToLock as type string with format int64, so the
  //   // value must be a QUOTED number even though it holds one.  Intuition says otherwise.
  //   expect: { mustContain: ['maximumTimeToLock', '"300000"'], mustNotContain: [':300000'] }
  // },
  // {
  //   id: 'android-developer-settings',
  //   profileType: 'android',
  //   canary: true,
  //   instructions: 'Block developer options.',
  //   readByEye: 'developerSettings must be NESTED inside advancedSecurityOverrides.  The assertion only checks both strings are present, so a flattened top-level developerSettings would still pass -- and would be silently discarded on delivery.',
  //   // CANARY.  Nested enum -- the exact shape where a typo one level down is silently discarded
  //   // rather than rejected.
  //   expect: { mustContain: ['advancedSecurityOverrides', 'developerSettings', 'DEVELOPER_SETTINGS_DISABLED'] }
  // },
  // {
  //   id: 'android-camera',
  //   profileType: 'android',
  //   instructions: 'Disable the camera.',
  //   // cameraAccess supersedes cameraDisabled, and its enum values are SCREAMING_SNAKE_CASE.
  //   expect: {
  //     mustContain: ['cameraAccess', 'CAMERA_ACCESS_DISABLED'],
  //     mustNotContain: ['cameraDisabled', 'camera_access_disabled'],
  //     deliveryNotes: ''
  //   }
  // },
  // {
  //   id: 'android-screenshots',
  //   profileType: 'android',
  //   instructions: 'Block screenshots.',
  //   expect: { mustContain: ['screenCaptureDisabled'], deliveryNotes: '' }
  // },
  // {
  //   id: 'android-factory-reset',
  //   profileType: 'android',
  //   instructions: 'Prevent users from factory resetting the device.',
  //   expect: { mustContain: ['factoryResetDisabled'], deliveryNotes: '' }
  // },
  // {
  //   id: 'android-password-length',
  //   profileType: 'android',
  //   instructions: 'Require a 10-character password.',
  //   // Nested path plus an integer that stays a JSON number.
  //   expect: { mustContain: ['passwordRequirements', 'passwordMinimumLength', '10'] }
  // },
  // {
  //   id: 'android-failed-attempts',
  //   profileType: 'android',
  //   instructions: 'Wipe the device after 10 failed password attempts.',
  //   expect: { mustContain: ['passwordRequirements', 'maximumFailedPasswordsForWipe'] }
  // },
  // {
  //   id: 'android-bluetooth-contact-sharing',
  //   profileType: 'android',
  //   instructions: 'Turn off Bluetooth contact sharing.',
  //   expect: { mustContain: ['bluetoothContactSharingDisabled'] }
  // },
  // {
  //   id: 'android-encryption-natively-managed',
  //   profileType: 'android',
  //   instructions: 'Turn on disk encryption.',
  //   // encryptionPolicy is legal Android Management API but rejected by some MDMs, so this must
  //   // SUCCEED with a warning rather than refuse.
  //   expect: { mustContain: ['encryptionPolicy'], deliveryNotesIsNotEmpty: true }
  // },
  // {
  //   id: 'android-nonexistent-geofence-alert',
  //   profileType: 'android',
  //   instructions: 'Email IT when the device leaves the office.',
  //   expect: { expectFailure: true }
  // },

];


module.exports = {


  friendlyName: 'Test llm generated configuration profile',


  description: 'Generate configuration profiles and report wall-clock time plus whether the output holds up, for one instruction or for every case in this script.',


  extendedDescription:
`The prompt is inlined at the top of this script rather than read from the action, so a model or
effort level can be swept without editing a public endpoint.  That copy will drift: --all checks
that every inlined rule still appears verbatim in the action's source and warns on any that
don't, but a rule ADDED to the action cannot be detected -- re-copy when the action changes.

Examples:
  sails run test-llm-generated-configuration-profile --profileType=csp --naturalLanguageInstructions="Require a device password"
  sails run test-llm-generated-configuration-profile --all
  sails run test-llm-generated-configuration-profile --all --profileType=android --verbose
  sails run test-llm-generated-configuration-profile --all --sweep=low,medium,high
  sails run test-llm-generated-configuration-profile --all --profileType=csp --baseModel=claude-haiku-4-5`,


  inputs: {

    profileType: {
      type: 'string',
      isIn: ['mobileconfig', 'csp', 'ddm', 'android'],
      description: 'Generate one profile of this type, or with --all, run only this type\'s cases.'
    },

    naturalLanguageInstructions: {
      type: 'string',
      description: 'The instructions to generate from.  Required unless --all is set.'
    },

    all: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Run every case defined at the top of this script.'
    },

    baseModel: {
      type: 'string',
      defaultsTo: 'claude-sonnet-5',
      description: 'The model to generate with.'
    },

    effort: {
      type: 'string',
      isIn: ['low', 'medium', 'high', 'xhigh', 'max'],
      description: 'Effort level.  Omitted means the model\'s default, which is high on Sonnet 5.'
    },

    sweep: {
      type: 'string',
      description: 'Comma-separated effort levels to run each case at, e.g. "low,medium,high".  Overrides --effort.'
    },

    verbose: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Print the full profile and citations for every case.  Failures print them regardless.'
    }

  },


  fn: async function ({profileType, naturalLanguageInstructions, all, baseModel, effort, sweep, verbose}) {

    let path = require('path');
    let util = require('util');

    if(!all && !naturalLanguageInstructions) {
      throw new Error('Either --naturalLanguageInstructions or --all is required.  See `sails run test-llm-generated-configuration-profile --help`.');
    }

    let cases;
    if(all) {
      cases = _.filter(TEST_CASES, (testCase)=>{
        return !profileType || testCase.profileType === profileType;
      });
      if(cases.length === 0) {
        throw new Error(`No cases defined for profileType "${profileType}".`);
      }
      await warnAboutAnyDrift(path.resolve(sails.config.appPath, 'api/controllers/get-llm-generated-configuration-profile.js'));
    } else {
      cases = [{ id: 'ad-hoc', profileType, instructions: naturalLanguageInstructions }];
    }

    let effortLevelsToRun = sweep ? _.map(sweep.split(','), (level)=>{ return _.trim(level); }) : [effort];
    let results = [];

    for (let effortLevel of effortLevelsToRun) {
      for (let testCase of cases) {

        let startedAt = Date.now();
        let rawResult;
        let unexpectedError;
        try {
          rawResult = await sails.helpers.ai.prompt.with({
            systemPrompt: assembleSystemPrompt(testCase.profileType),
            prompt: assembleUserPrompt(testCase.profileType, testCase.instructions),
            baseModel,
            effort: effortLevel,
            expectJson: true,
          });
        } catch (err) {
          unexpectedError = err;
        }
        let elapsedMs = Date.now() - startedAt;

        // Mirror the action's own acceptance test: an abstention, or a response missing any
        // required key, is not a usable profile.
        let abstained = !unexpectedError && (
          rawResult.couldNotGenerateProfile ||
          !rawResult.configurationProfile ||
          !rawResult.profileFilename ||
          !rawResult.settingsEnforced
        );
        let generatedProfile = (unexpectedError || abstained) ? undefined : {
          profile: rawResult.configurationProfile,
          profileFilename: rawResult.profileFilename,
          deliveryNotes: rawResult.deliveryNotes,
          items: rawResult.settingsEnforced,
        };

        let expectations = testCase.expect || {};
        let checkFailures;
        if(unexpectedError) {
          // Distinguished from an abstention on purpose: collapsing them would hide a
          // misconfigured anthropicSecret as a model refusal.
          checkFailures = [`unexpected error: ${unexpectedError.message}`];
        } else if(abstained) {
          checkFailures = expectations.expectFailure ? [] : [
            `abstained but a profile was expected -- reason given: ${JSON.stringify(rawResult.reasonWhyAProfileCouldNotBeGenerated || '(none)')}`
          ];
        } else {
          checkFailures = checkExpectations(expectations, generatedProfile);
        }

        results.push({
          id: testCase.id,
          profileType: testCase.profileType,
          canary: !!testCase.canary,
          effort: effortLevel || '(default)',
          elapsedMs,
          checkFailures,
          generatedProfile,
        });

        // Report as each case finishes.  A full sweep takes minutes, and a summary-only script
        // makes a slow run indistinguishable from a hung one.
        sails.log(
          `${String(elapsedMs).padStart(6)}ms  ` +
          `${String(effortLevel || 'default').padEnd(8)} ` +
          `${testCase.profileType.padEnd(13)} ` +
          `${String(testCase.id).padEnd(38)} ` +
          `${checkFailures.length === 0 ? 'ok' : 'FAILED'}`
        );
        for (let checkFailure of checkFailures) {
          sails.log(`         └─ ${checkFailure}`);
        }

        // Print the output for anything that failed, for every canary, and for everything when
        // --verbose.  A failure you can't see is a failure you can't act on -- and a canary that
        // passes its substring checks still needs a human, because the properties that matter most
        // on those cases are the ones substrings can't express.
        if(generatedProfile && (verbose || testCase.canary || checkFailures.length > 0)) {
          let banner = testCase.canary && checkFailures.length === 0 ? 'CANARY, automated checks passed -- confirm by eye' : testCase.canary ? 'CANARY, FAILED' : checkFailures.length > 0 ? 'FAILED' : 'ok';
          sails.log(`\n──── ${testCase.id} @ effort ${effortLevel || 'default'} (${baseModel}) -- ${banner} ────`);
          sails.log(`instructions: ${testCase.instructions}`);
          if(testCase.readByEye) {
            sails.log(`CONFIRM BY EYE: ${testCase.readByEye}`);
          }
          sails.log(`profileFilename: ${generatedProfile.profileFilename}`);
          sails.log(`deliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}`);
          sails.log(`\n${generatedProfile.profile}\n`);
          sails.log(`settingsEnforced:\n${util.inspect(generatedProfile.items, {depth: 4, colors: false})}`);
          sails.log('────────\n');
        }
      }
    }

    printResultsTable(results, effortLevelsToRun, baseModel);

    sails.log('\n=== Summary ===');
    for (let effortLevel of effortLevelsToRun) {
      let resultsForThisEffort = _.where(results, { effort: effortLevel || '(default)' });
      let passing = _.filter(resultsForThisEffort, (result)=>{ return result.checkFailures.length === 0; });
      let elapsedTimes = _.pluck(resultsForThisEffort, 'elapsedMs').sort((a, b)=>{ return a - b; });
      sails.log(
        `${baseModel}  effort ${String(effortLevel || 'default').padEnd(8)} ` +
        `passed ${passing.length}/${resultsForThisEffort.length}  ` +
        `median ${elapsedTimes[Math.floor(elapsedTimes.length / 2)]}ms  ` +
        `slowest ${_.last(elapsedTimes)}ms`
      );
    }

    // Citation resolution can't be checked automatically -- it needs a human against the published
    // reference -- but it degrades before the pass rate does, so surface the raw material.
    let citations = _.uniq(_.flatten(_.map(_.filter(results, 'generatedProfile'), (result)=>{
      return _.map(result.generatedProfile.items || [], (item)=>{ return `${result.id}: ${item.schemaReference}`; });
    })));
    sails.log(`\n${citations.length} citation(s) to spot-check against the published reference:`);
    for (let citation of citations) {
      sails.log(`  ${citation}`);
    }

    // Canaries are the real gate, not the aggregate.  They fail quietly -- a profile that deploys
    // and enforces nothing still looks like a pass to someone skimming output -- so report them
    // separately rather than letting them average out against 30-odd ordinary cases.
    let canaryResults = _.filter(results, 'canary');
    let failedCanaries = _.filter(canaryResults, (result)=>{ return result.checkFailures.length > 0; });
    if(canaryResults.length > 0) {
      if(failedCanaries.length === 0) {
        sails.log(`\nAll ${canaryResults.length} canary run(s) passed their automated checks.  Their full output is printed above -- confirm the "CONFIRM BY EYE" items before calling this a pass, since the properties that matter most on those cases are the ones substring checks cannot express.`);
      } else {
        sails.log.warn(`\n${failedCanaries.length} of ${canaryResults.length} canary run(s) FAILED.  Treat this as blocking regardless of the aggregate pass rate:`);
        for (let failedCanary of failedCanaries) {
          sails.log.warn(`  ${failedCanary.id} @ ${failedCanary.effort}: ${failedCanary.checkFailures.join('; ')}`);
        }
      }
    }

    let failedResults = _.filter(results, (result)=>{ return result.checkFailures.length > 0; });
    if(failedResults.length > 0) {
      sails.log.warn(`\n${failedResults.length} of ${results.length} run(s) failed:`);
      for (let failedResult of failedResults) {
        sails.log.warn(`  ${failedResult.id} @ ${failedResult.effort}: ${failedResult.checkFailures.join('; ')}`);
      }
      sails.log.warn('A speed change that raises this number is not a win.');
    }

  }


};


/**
 * Print the run as a table.
 *
 * A single effort level gets a flat table.  A sweep gets a pivot -- one row per case, one column
 * per effort -- because the question a sweep exists to answer is "how far down can this go before
 * it breaks", and that is read by scanning left to right along a row.
 *
 * Uses console.log rather than sails.log: the log prefix would break column alignment.
 *
 * @param  {Array} results
 * @param  {Array} effortLevelsToRun
 * @param  {String} baseModel
 */
function printResultsTable(results, effortLevelsToRun, baseModel) {
  let labelFor = (effortLevel)=>{ return effortLevel || '(default)'; };
  let cellFor = (result)=>{
    if(!result) {
      return '--';
    }
    return `${result.elapsedMs}ms ` + (result.checkFailures.length === 0 ? 'ok' : `FAIL ${result.checkFailures.length}`);
  };
  // An asterisk rather than a separate column: it keeps the case name and its weight together, and
  // the name column is already the widest thing in the table.
  let nameFor = (result)=>{ return result.canary ? `${result.id} *` : result.id; };

  console.log(`\n=== Results (${baseModel}) ===`);

  if(effortLevelsToRun.length === 1) {
    printTable(
      ['case', 'type', 'ms', 'result'],
      _.map(results, (result)=>{
        return [nameFor(result), result.profileType, String(result.elapsedMs), result.checkFailures.length === 0 ? 'ok' : `FAIL ${result.checkFailures.length}`];
      }),
      ['left', 'left', 'right', 'left']
    );
  } else {
    // One row per case, in the order the cases were defined.
    let caseIdsInOrder = _.uniq(_.pluck(results, 'id'));
    printTable(
      ['case'].concat(_.map(effortLevelsToRun, labelFor)),
      _.map(caseIdsInOrder, (caseId)=>{
        let resultsForThisCase = _.where(results, { id: caseId });
        return [nameFor(_.first(resultsForThisCase))].concat(
          _.map(effortLevelsToRun, (effortLevel)=>{
            return cellFor(_.find(resultsForThisCase, { effort: labelFor(effortLevel) }));
          })
        );
      }),
      ['left'].concat(_.map(effortLevelsToRun, ()=>{ return 'left'; }))
    );
  }

  if(_.some(results, 'canary')) {
    console.log('* canary -- these fail quietly, so read their output rather than trusting the pass count.');
  }
}


/**
 * Render a box-drawn table, sizing each column to its widest cell.
 *
 * @param  {Array} headers
 * @param  {Array} rows  array of arrays of strings
 * @param  {Array} alignments  'left' or 'right', one per column
 */
function printTable(headers, rows, alignments) {
  let columnWidths = _.map(headers, (header, columnIndex)=>{
    return _.reduce(rows, (widestSoFar, row)=>{
      return Math.max(widestSoFar, String(row[columnIndex]).length);
    }, header.length);
  });

  let padCell = (value, columnIndex)=>{
    let cell = String(value);
    return alignments[columnIndex] === 'right' ? cell.padStart(columnWidths[columnIndex]) : cell.padEnd(columnWidths[columnIndex]);
  };
  let horizontalRule = (left, join, right)=>{
    return left + _.map(columnWidths, (width)=>{ return '─'.repeat(width + 2); }).join(join) + right;
  };
  let renderRow = (cells)=>{
    return '│ ' + _.map(cells, (cell, columnIndex)=>{ return padCell(cell, columnIndex); }).join(' │ ') + ' │';
  };

  console.log(horizontalRule('┌', '┬', '┐'));
  console.log(renderRow(headers));
  console.log(horizontalRule('├', '┼', '┤'));
  for (let row of rows) {
    console.log(renderRow(row));
  }
  console.log(horizontalRule('└', '┴', '┘'));
}


/**
 * Build the system prompt for one profile type, the same way the action does.
 *
 * @param  {String} profileType
 * @returns {String}
 */
function assembleSystemPrompt(profileType) {
  let promptConfig = PROMPT_CONFIG_BY_PROFILE_TYPE[profileType];
  if(!promptConfig) {
    throw new Error(`No inlined prompt config for profileType "${profileType}".`);
  }

  let providedSchema = '';
  if(promptConfig.usesPolicySchema) {
    let policySchema = sails.config.builtStaticContent.androidManagementPolicySchema;
    if(!_.isObject(policySchema) || !_.isObject(policySchema.Policy)) {
      throw new Error(
        'Android cases need sails.config.builtStaticContent.androidManagementPolicySchema, which is\n' +
        'missing or has no "Policy" key.  Run `sails run get-android-management-policy-schema`, then\n' +
        '`sails run build-static-content`.'
      );
    }
    providedSchema = `
Provided context (the Android Management API Policy schema, pruned).  A property whose value is a "$ref" expands into a nested object using that named schema's properties:
\`\`\`
${JSON.stringify(policySchema)}
\`\`\`
`;
  }

  let numberedRules = SHARED_RULES.concat(promptConfig.rules, DELIVERY_NOTES_RULES)
    .map((rule, idx)=>`${idx + 1}. ${rule}`)
    .join('\n    ');

  return `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

You generate a ${promptConfig.description} from an IT admin's instructions.

Draw setting names, types, and allowed values from these published references:
${promptConfig.references.map((reference)=>`- ${reference}`).join('\n    ')}
${providedSchema}
When generating the profile:
${numberedRules}

${RESPONSE_SHAPE}`;
}


/**
 * Build the user-turn prompt, the same way the action does.
 *
 * @param  {String} profileType
 * @param  {String} naturalLanguageInstructions
 * @returns {String}
 */
function assembleUserPrompt(profileType, naturalLanguageInstructions) {
  return `Given these instructions from an IT admin, generate a ${PROMPT_CONFIG_BY_PROFILE_TYPE[profileType].description}.

    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``;
}


/**
 * Warn about any inlined rule that no longer appears in the action's source.
 *
 * Catches edits and removals exactly, because the rules are copied verbatim.  It cannot catch a
 * rule ADDED to the action -- nothing here would know to look for it.
 *
 * @param  {String} actionPath
 */
async function warnAboutAnyDrift(actionPath) {
  let actionSource;
  try {
    actionSource = await sails.helpers.fs.read(actionPath);
  } catch (err) {
    sails.log.warn(`Could not read the action to check for prompt drift (${err.message}).  Proceeding.`);
    return;
  }

  let staleRules = [];
  let checkTheseRules = SHARED_RULES.concat(DELIVERY_NOTES_RULES);
  for (let profileType in PROMPT_CONFIG_BY_PROFILE_TYPE) {
    if(!PROMPT_CONFIG_BY_PROFILE_TYPE[profileType].notYetInTheAction) {
      checkTheseRules = checkTheseRules.concat(PROMPT_CONFIG_BY_PROFILE_TYPE[profileType].rules);
    }
  }

  // Compare with backslashes and whitespace removed from both sides.  The rules here are evaluated
  // strings while the action is raw source, so an apostrophe is `'` in one and `\'` in the other,
  // and a literal newline escape is `\n` versus `\\n`.  Without this, every rule containing either
  // would be reported as drifted on the first run.
  let normalizeForComparison = (str)=>{ return String(str).replace(/\\/g, '').replace(/\s+/g, ''); };
  let normalizedActionSource = normalizeForComparison(actionSource);

  for (let rule of checkTheseRules) {
    if(!_.contains(normalizedActionSource, normalizeForComparison(rule))) {
      staleRules.push(rule);
    }
  }

  if(staleRules.length > 0) {
    sails.log.warn(`\n${staleRules.length} of ${checkTheseRules.length} inlined rule(s) no longer appear in the action, so this script is testing a prompt production does not send:`);
    for (let staleRule of staleRules) {
      sails.log.warn(`  - ${staleRule.slice(0, 110)}…`);
    }
    sails.log.warn('Re-copy the prompt from the action before trusting these results.  (A rule added to the action cannot be detected here.)\n');
  } else {
    sails.log(`Prompt drift check: all ${checkTheseRules.length} inlined rules still present in the action.  (Additions can't be detected.)`);
  }
}


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
      failures.push(`missing expected element: <${tagName}>${textContent}</${tagName}>`);
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

  if(expectations.deliveryNotes === '' && generatedProfile.deliveryNotes !== '') {
    failures.push(`expected empty deliveryNotes, got: ${JSON.stringify(generatedProfile.deliveryNotes)}`);
  }
  if(expectations.deliveryNotesIsNotEmpty && !generatedProfile.deliveryNotes) {
    failures.push('expected a deliveryNotes entry, got an empty string');
  }

  return failures;
}
