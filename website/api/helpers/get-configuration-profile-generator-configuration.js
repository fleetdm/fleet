module.exports = {


  friendlyName: 'Get configuration profile generator configuration',


  description: 'Builds and returns the prompts and configuration that the configuration profile generator and related test script uses.',



  inputs: {
    profileType: {
      type: 'string',
      required: true,
      isIn: [
        'mobileconfig',
        'ddm',
        'csp',
      ],
    },
    naturalLanguageInstructions: {
      type: 'string',
      required: true,
      description: 'What the IT admin asked for, in their own words.'
    },

    useLighterResponseShape: {
      type: 'boolean',
      description: 'Whether or not to request less information back with the generated profile.'
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Configuration profile generator configuration',
    },

  },


  fn: async function ({profileType, naturalLanguageInstructions, useLighterResponseShape}) {

    // Apple's published DDM configuration declarations, pruned to what a generator must not get wrong:
    // exact key name, type, and whatever constrains the value.  Titles, prose and per-OS availability are
    // dropped.  Regenerate by walking apple/device-management's declarative/declarations/configurations.
    const DDM_DECLARATION_SCHEMA = `com.apple.configuration.account.caldav
      VisibleName:string, HostName:string*, Port:integer, Path:string, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.account.carddav
      VisibleName:string, HostName:string*, Port:integer, Path:string, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.account.exchange
      VisibleName:string, EnabledProtocolTypes[], UserIdentityAssetReference:string, HostName:string, Port:integer, Path:string, ExternalHostName:string, ExternalPort:integer, External Path:string, OAuth{Enabled:boolean*, SignInURL:string, TokenRequestURL:string}, AuthenticationCredentialsAssetReference:string, AuthenticationIdentityAssetReference:string, SMIME{Signing{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean}, Encryption{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean, PerMessageSwitchEnabled:boolean}}, MailServiceActive:boolean, LockMailService:boolean, ContactsServiceActive:boolean, LockContactsService:boolean, CalendarServiceActive:boolean, LockCalendarService:boolean, RemindersServiceActive:boolean, LockRemindersService:boolean, NotesServiceActive:boolean, LockNotesService:boolean
    com.apple.configuration.account.google
      VisibleName:string, UserIdentityAssetReference:string*
    com.apple.configuration.account.ldap
      VisibleName:string, HostName:string*, Port:integer, AuthenticationCredentialsAssetReference:string, SearchSettings[]
    com.apple.configuration.account.mail
      VisibleName:string, UserIdentityAssetReference:string, IncomingServer{ServerType:string(IMAP|POP)*, HostName:string*, Port:integer, AuthenticationMethod:string(None|Password|CRAMMD5|NTLM|HTTPMD5)*, AuthenticationCredentialsAssetReference:string, IMAPPathPrefix:string}, OutgoingServer{HostName:string*, Port:integer, AuthenticationMethod:string(None|Password|CRAMMD5|NTLM|HTTPMD5)*, AuthenticationCredentialsAssetReference:string}, SMIME{Signing{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean}, Encryption{Enabled:boolean*, IdentityAssetReference:string, UserOverrideable:boolean, IdentityUserOverrideable:boolean, PerMessageSwitchEnabled:boolean}}
    com.apple.configuration.account.subscribed-calendar
      VisibleName:string, CalendarURL:string*, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.app.managed
      AppStoreID:string, BundleID:string, ManifestURL:string, AppComposedIdentifier:string, iOSApp:boolean, InstallBehavior{Install:string(Optional|Required), License{Assignment:string(Device|User), VPPType:string(Device|User)}, Version:integer, AllowDownloadsOverCellular:string(AlwaysOn|AlwaysOff|StoreSettings)}, UpdateBehavior{AutomaticAppUpdates:string(AlwaysOn|AlwaysOff|StoreSettings)*}, IncludeInBackup:boolean, Attributes{AssociatedDomains[], AssociatedDomainsEnableDirectDownloads:boolean, CellularSliceUUID:string, ContentFilterUUID:string, DNSProxyUUID:string, Hideable:boolean, Lockable:boolean, RelayUUID:string, TapToPayScreenLock:boolean, VPNUUID:string}, AppConfig{DataAssetReference:string, Passwords[], Identities[], Certificates[]}, ExtensionConfigs{ANY{DataAssetReference:string, Passwords[], Identities[], Certificates[]}}, LegacyAppConfigAssetReference:string
    com.apple.configuration.audio-accessory.settings
      TemporaryPairing{Disabled:boolean, Configuration{UnpairingTime{Policy:string(None|Hour)*, Hour:integer(0-23)}}}
    com.apple.configuration.diskmanagement.settings
      Restrictions{ExternalStorage:string(Allowed|ReadOnly|Disallowed), NetworkStorage:string(Allowed|ReadOnly|Disallowed)}
    com.apple.configuration.external-intelligence.settings
      Enabled:boolean, AllowSignIn:boolean, AllowedWorkspaceIDs[]
    com.apple.configuration.intelligence.settings
      AllowAppleIntelligenceReport:boolean, AllowGenmoji:boolean, AllowImagePlayground:boolean, AllowImageWand:boolean, AllowPersonalizedHandwritingResults:boolean, AllowVisualIntelligenceSummary:boolean, AllowWritingTools:boolean, Apps{Mail{AllowSmartReplies:boolean, AllowSummary:boolean}, Notes{AllowTranscription:boolean, AllowTranscriptionSummary:boolean}, Safari{AllowSummary:boolean}}, ForceOnDeviceOnlyDictation:boolean, ForceOnDeviceOnlyTranslation:boolean
    com.apple.configuration.keyboard.settings
      AllowAutoCorrection:boolean, AllowDefinitionLookup:boolean, AllowDictation:boolean, AllowMathKeyboardSuggestions:boolean, AllowPredictiveText:boolean, AllowSlideToType:boolean, AllowSpellCheck:boolean, AllowTextReplacement:boolean
    com.apple.configuration.legacy
      ProfileURL:string*
    com.apple.configuration.legacy.interactive
      ProfileURL:string*, VisibleName:string*
    com.apple.configuration.management.status-subscriptions
      StatusItems[]
    com.apple.configuration.management.test
      Echo:string*, EchoDataAssetReference:string, ReturnStatus:string(Installed|Failed|Unlocked)
    com.apple.configuration.math.settings
      Calculator{BasicMode{AddSquareRoot:boolean*}, ScientificMode{Enabled:boolean*}, ProgrammerMode{Enabled:boolean*}, MathNotesMode{Enabled:boolean*}, InputModes{UnitConversion:boolean*, RPN:boolean*}}, SystemBehavior{KeyboardSuggestions:boolean*, MathNotes:boolean*}
    com.apple.configuration.migration-assistant.settings
      ShouldDoManagedMigration:boolean*, ExcludedAccounts[], ExcludedPaths[], RequiredPaths[], ShouldMigrateSecurityPrivacySettings:boolean*
    com.apple.configuration.package
      ManifestURL:string*, InstallBehavior{Install:string(Optional|Required)}
    com.apple.configuration.passcode.settings
      RequirePasscode:boolean, RequireAlphanumericPasscode:boolean, RequireComplexPasscode:boolean, MinimumLength:integer(0-16), MinimumComplexCharacters:integer(0-4), MaximumFailedAttempts:integer(2-11), FailedAttemptsResetInMinutes:integer, MaximumGracePeriodInMinutes:integer, MaximumInactivityInMinutes:integer(0-15), MaximumPasscodeAgeInDays:integer(0-730), PasscodeReuseLimit:integer(1-50), ChangeAtNextAuth:boolean, CustomRegex{Regex:string*, Description{ANY:string}}
    com.apple.configuration.safari.bookmarks
      ManagedBookmarks[]
    com.apple.configuration.safari.extensions.settings
      ManagedExtensions{ANY{State:string(Allowed|AlwaysOn|AlwaysOff), PrivateBrowsing:string(Allowed|AlwaysOn|AlwaysOff), AllowedDomains[], DeniedDomains[]}}
    com.apple.configuration.safari.settings
      AcceptCookies:string(Never|CurrentWebsite|VisitedWebsites|Always), AllowDisablingFraudWarning:boolean, AllowHistoryClearing:boolean, AllowJavaScript:boolean, AllowPrivateBrowsing:boolean, AllowPopups:boolean, AllowSummary:boolean, NewTabStartPage{PageType:string(Start|Home|Extension)*, HomepageURL:string, ExtensionIdentifier:string}
    com.apple.configuration.screensharing.connection
      ConnectionUUID:string*, DisplayName:string*, HostName:string*, Port:integer, DisplayConfiguration{DisplayType:string(Virtual1|Virtual2)*}, AuthenticationCredentialsAssetReference:string
    com.apple.configuration.screensharing.connection.group
      ConnectionGroupUUID:string*, GroupName:string*, Members[]
    com.apple.configuration.screensharing.host.settings
      MaximumVirtualDisplays:integer(0-2), PortBase:integer(1024-65535), PreventCopyFilesFromHost:boolean, PreventCopyFilesToHost:boolean, PreventHighPerformanceConnections:boolean
    com.apple.configuration.security.certificate
      CredentialAssetReference:string*
    com.apple.configuration.security.identity
      CredentialAssetReference:string*, AllowAllAppsAccess:boolean, KeyIsExtractable:boolean
    com.apple.configuration.security.passkey.attestation
      AttestationIdentityAssetReference:string*, AttestationIdentityKeyIsExtractable:boolean, RelyingParties[]
    com.apple.configuration.services.background-tasks
      TaskType:string*, TaskDescription:string, ExecutableAssetReference:string, LaunchdConfigurations[]
    com.apple.configuration.services.configuration-files
      ServiceType:string*, DataAssetReference:string*
    com.apple.configuration.siri.settings
      Enabled:boolean, AllowUserGeneratedContent:boolean, AllowWhileLocked:boolean, ForceProfanityFilter:boolean
    com.apple.configuration.softwareupdate.enforcement.specific
      TargetOSVersion:string*, TargetBuildVersion:string, TargetLocalDateTime:string*, DetailsURL:string
    com.apple.configuration.softwareupdate.settings
      Notifications:boolean, Deferrals{CombinedPeriodInDays:integer(1-90), MajorPeriodInDays:integer(1-90), MinorPeriodInDays:integer(1-90), SystemPeriodInDays:integer(1-90)}, RecommendedCadence:string(All|Oldest|Newest), AutomaticActions{Download:string(Allowed|AlwaysOn|AlwaysOff), InstallOSUpdates:string(Allowed|AlwaysOn|AlwaysOff), InstallSecurityUpdate:string(Allowed|AlwaysOn|AlwaysOff)}, RapidSecurityResponse{Enable:boolean, EnableRollback:boolean}, AllowStandardUserOSUpdates:boolean, Beta{ProgramEnrollment:string(Allowed|AlwaysOn|AlwaysOff), OfferPrograms[], RequireProgram{Description:string*, Token:string*}}
    com.apple.configuration.watch.enrollment
      EnrollmentProfileURL:string*, AnchorCertificateAssetReferences[]`;


    // The tail of the system prompt.
    let RESPONSE_SHAPE;
    if(!useLighterResponseShape) {

      RESPONSE_SHAPE = `Respond in JSON with this data shape:
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
    } else {
      RESPONSE_SHAPE = `Respond in JSON with this data shape:
      {
        "configurationProfile": "TODO",
        "profileFilename": "TODO",
        "settingsEnforced": [// For each setting enforced by the configuration profile.
          {
            name: "TODO",
            value: "TODO",
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
    }

    // Rules that apply to every profile type.  Ordered by consequence: a violation of an early rule produces a profile that deploys cleanly and does nothing.
    let sharedRules = [
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

    // "deliveryNotes" defaults to noise unless it is aggressively constrained.  An empty string is a weaker affordance than an empty array, so these rules carry more of the load.
    let deliveryNotesRules = [
      '"deliveryNotes" is for exceptions only: something the admin has to do or decide that is not visible in the profile itself.  Use an empty string when nothing applies.  An empty string is the right answer for most profiles -- prefer it whenever you are unsure whether a note earns its place.',
      'Never write a sentence stating that a condition does not apply.  "No credentials or secrets are embedded" and "no companion declaration is required" are not notes -- leaving them out already says that.',
      'Never restate what the profile is, which platform it targets, or how that platform is normally delivered.  The admin chose the format and already knows.',
      'One sentence per action, addressed to the admin, and no more than two sentences in total.  For example: "Replace the passphrase with a secret variable before committing this to a repository."',
    ];

    let promptConfigByProfileType = {

      'csp': {
        description: 'CSP XML profile that enforces OS settings on Windows devices',
        // How the triage prompt describes a setting the references below actually cover.  Kept
        // next to those references so the two cannot drift apart.
        firstPartySettingDescription: 'a node in a Microsoft-published CSP',
        references: [
          'Windows CSP nodes, formats, and allowed values: https://learn.microsoft.com/en-us/windows/client-management/mdm/',
        ],
        rules: [
          // Document shape.  Wrong here and Fleet rejects the file on upload.
          'A Windows profile is a sequence of OMA-DM command elements, not a SyncML document.  The top level must be one or more <Add>, <Replace>, <Exec>, or <Atomic> elements and nothing else.  Never wrap them in <SyncML>, <SyncBody>, <Final/>, or an invented container such as <WindowsCSP>, and never emit an XML declaration -- the SyncML envelope carries session state (SyncHdr, SessionID, MsgID) that only the MDM server can populate at delivery time.',
          'If you use <Atomic>, it must wrap every command in the profile.  Never mix an <Atomic> element with sibling top-level commands, and never nest one <Atomic> inside another.',
          // Node identity.
          'Emit only LocURI paths that exist in a published CSP.  Never construct, guess, or extrapolate a path, and name the CSP you used in "schemaReference".',
          'Percent-encode any character in a LocURI path segment that is not URI-legal.  An SSID named "Cool Network" is Cool%20Network in the LocURI, while the value inside the payload keeps its literal spaces.',
          // Type integrity.  The largest source of silent failure.
          'In the Policy CSP, settings that read as boolean are almost always DFFormat int with allowed values 0 and 1, not bool.  Default to <Format>int</Format> with numeric <Data>.  Use bool only when the node genuinely declares it, and say so in "caveats".',
          'Before returning, re-read every Item and confirm <Data> is legal for the <Format> you declared: int accepts digits only, bool accepts literally true or false, chr accepts text.  If you have written 0 or 1 as the data, the format is int, not bool.',
          'Never infer a boolean value from the node name.  DeviceLock/DevicePasswordEnabled takes 0 for enabled and 1 for disabled.  Always state what the value you chose actually does in "valueMeaning".',
          'State the node\'s declared DFFormat verbatim in "allowedValues", next to the format you emitted, so a reviewer can compare the two side by side.',
          'If you cannot recall a node\'s documented DFFormat with confidence, do not guess -- return the "couldNotGenerateProfile" shape and name the node you were unsure about.',
          'ADMX-backed Policy CSP nodes are never int.  They declare DFFormat chr and take a policy fragment as their value: <Format>chr</Format> with <Data><![CDATA[<enabled/>]]></Data> or <![CDATA[<disabled/>]]>, plus a nested <data id="..." value="..."/> element for each sub-option the admin actually asked for and none for the ones they did not.  Writing 1 or 0 to one of these nodes deploys cleanly and enforces nothing.',
          'Never derive the area segment of a Policy CSP LocURI from the name of the .admx file that backs it.  Most ADMX-backed areas are named ADMX_<AdmxFileName>, but plenty are not: PowerShell script block logging lives under WindowsPowerShell, not ADMX_PowerShellExecutionPolicy or ADMX_PowerShell, even though PowerShellExecutionPolicy.admx is the backing file.  Use the area exactly as the published node lists it, and if you cannot recall it, return the "couldNotGenerateProfile" shape rather than constructing one that looks right.',
          // Verb and dependencies.
          'Use Replace for Policy CSP leaf nodes that hold a value.  Reserve Add for nodes that do not exist until you create them, such as ADMXInstall, certificate installs, and WiFi or VPN profile instances.  When a node\'s AccessType permits both, choose Replace, because the profile may reach a host where the value is already set and Add can fail there.',
          'Only use an Add or Replace verb on a node whose AccessType permits it.  Roughly 500 leaf nodes accept only Get, and a Replace against one is accepted, deploys, and then fails on the device.',
          'Include the nodes the requested setting depends on.  DeviceLock/MinDevicePasswordLength has no effect unless DeviceLock/DevicePasswordEnabled is also set.',
          'Record the minimum OS build and the Windows editions each node applies to in "caveats".  A valid setting aimed at the wrong SKU is a silent no-op, not an error.',
          // Embedded values.
          'When a node\'s value is embedded XML -- WiFi/Profile/*/WlanXml, ADMXInstall, and similar -- wrap it in <![CDATA[ ... ]]> rather than escaping it as entities, and emit that value as a single line with no line breaks or indentation between its elements.  This applies only to the embedded value; the surrounding SyncML keeps its normal indentation.',
          'Be consistent within a profile about optional <Meta> children.  If you emit <Type> for one item, emit it for all of them, and namespace every <Meta> child as xmlns="syncml:metinf".',
          'For WiFi profiles, emit both <name> and <hex> inside <SSID>, where <hex> is the uppercase hex encoding of the SSID bytes.  Windows and some MDMs treat the hex form as authoritative when both are present.',
        ],
      },

      'mobileconfig': {
        description: 'XML .mobileconfig profile that enforces OS settings on macOS devices',
        firstPartySettingDescription: 'a key in an Apple-published payload',
        references: [
          'First-party Apple payloads: https://github.com/apple/device-management/tree/release/mdm/profiles',
          'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
        ],
        rules: [
          // Third-party payloads.
          'If this is an attempt to change a third-party application\'s settings, use that application\'s preference domain -- com.google.Chrome, us.zoom.config and its keys must come from the ProfileManifests reference.',
          // Document shape.
          'Emit valid property list XML: the plist DOCTYPE, plist version="1.0", and correctly typed values.',
          'Include PayloadIdentifier, PayloadType, PayloadUUID, PayloadVersion, and PayloadDisplayName on the root dict and on every dict inside PayloadContent.  The root PayloadType is "Configuration" and PayloadVersion is 1.',
          'Keep the plist indented and readable across multiple lines.  Do not collapse it onto one line.',
          // Key fidelity.  The mirror image of the DDM PascalCase defect.
          'Apple payload keys are not consistently cased, and the inconsistency is inside a single dict.  The passcode payload uses forcePIN, minLength, and allowSimple -- lowercase first letter -- beside PascalCase PayloadIdentifier and PayloadType.  Reproduce every key exactly as documented for its payload type.  Never normalize casing in either direction.',
          'Use only keys documented for the payload type you chose.  An invented key is written into the profile and nothing downstream rejects it, so the profile looks right and does nothing.',
          'If you cannot recall a payload type\'s exact key names, do not guess a casing and do not adapt a key from a DDM declaration -- return the "couldNotGenerateProfile" shape and name the payload type you were unsure about.',
          // Value typing.
          'Type every value as plist: <true/> or <false/> for booleans, never <string>true</string>; <integer> for whole numbers; <real> for decimals; <data> with base64 for binary; <date> with an ISO 8601 timestamp.',
          // Identifiers.
          'Take every PayloadUUID from the list of UUIDs provided with the instructions, in the order given, and never invent one.  Two dicts sharing a UUID is a profile that installs unpredictably, so use each one exactly once.',
          'PayloadIdentifier is reverse-DNS.  Each payload dict\'s identifier is the root identifier plus a distinguishing suffix, and no two identifiers in the profile are the same.',
          'PayloadDisplayName on the root is what an end user sees in System Settings, and some MDMs use it as the profile name.  Make it human-readable and specific to what the profile does.',
          // Structure.
          'Put every key for one payload domain in a single dict inside PayloadContent.  Do not emit several dicts with the same PayloadType.',


        ],
      },

      'ddm': {
        description: 'Apple DDM declaration in JSON format that enforces OS settings on macOS devices',
        firstPartySettingDescription: 'a key in an Apple-published declaration type',
        providedSchema: DDM_DECLARATION_SCHEMA,
        references: [
          'Apple DDM declaration types, keys, and values: provided in full below -- there is no other set.',
        ],
        rules: [
          'The declaration is JSON, not XML.  Include Type, Identifier, and Payload.',
          'Type must be a real declaration type, such as com.apple.configuration.passcode.settings.',
          // Key naming.  The most common DDM defect: JSON habit plus .mobileconfig bleed-through.
          'Every key inside Payload is PascalCase, with the first letter capitalized: RequirePasscode, MinimumLength, RequireAlphanumericPasscode.  A key with a lowercase first letter is an unknown key -- it is accepted as a typo and enforces nothing.',
          'Declarations do not reuse .mobileconfig payload key names, and the DDM name is not the .mobileconfig name recapitalized.  In the passcode payload, forcePIN becomes RequirePasscode and minLength becomes MinimumLength.  Never carry a .mobileconfig key into a declaration and never transform one into a declaration key.',
          'If you cannot recall a declaration\'s exact Payload key names, do not guess a casing and do not convert a .mobileconfig key -- return the "couldNotGenerateProfile" shape and name the declaration type you were unsure about.',
          // Identifier.
          'Identifier is your own reverse-DNS identifier for this declaration instance, not a copy of Type.  Copying Type conflates Apple\'s namespace with yours and collides the moment a second declaration of the same type exists.',
          'Derive Identifier from the full declaration type rather than its last component, or passcode.settings and softwareupdate.settings collapse into one identifier and silently overwrite each other.',
          'Keep Identifier to 64 bytes or fewer.  Apple\'s DeclarationBase caps it, and a longer identifier is accepted by an MDM and then rejected by the device at delivery.',
        ],
      },

    };


    let promptConfig = promptConfigByProfileType[profileType];

    // Generated list of UUIDs this profile can use.
    // Note: This is generated here and sent to the LLM to prevent it from adding invalid/placeholder UUIDs
    let suppliedPayloadUuids = [];
    if(profileType === 'mobileconfig') {
      for (let i = 0; i < 10; i++) {
        suppliedPayloadUuids.push((await sails.helpers.strings.uuid()).toUpperCase());
      }
    }


    let numberedRules = sharedRules.concat(promptConfig.rules, deliveryNotesRules)
      .map((rule, idx)=>`${idx + 1}. ${rule}`)
      .join('\n    ');

    let providedSchema = '';
    if(promptConfig.providedSchema) {
      providedSchema = `
Provided context: every declaration Apple publishes, and every key each one accepts.  Format is \`Key:type\`,
where \`*\` marks a required key, \`(a|b|c)\` lists the allowed values, \`(min-max)\` gives the allowed range,
\`{...}\` is a nested dictionary, and \`[]\` is an array.

This is the complete set.  A declaration type or key that does not appear here does not exist.
\`\`\`
${promptConfig.providedSchema}
\`\`\`
`;
    }

    let systemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

You generate a ${promptConfig.description} from an IT admin's instructions.

Draw setting names, types, and allowed values from these published references:
${promptConfig.references.map((reference)=>`- ${reference}`).join('\n    ')}
${providedSchema}
When generating the profile:
${numberedRules}

${RESPONSE_SHAPE}`;

    let uuidsToUse = '';
    if(suppliedPayloadUuids.length > 0) {
      uuidsToUse = `
    Use these UUIDs for PayloadUUID, in the order listed: the first on the root dict, then one per dict inside
    PayloadContent.  Never invent a UUID and never use one twice.  Where one payload references another payload
    (PayloadCertificateUUID, VPNUUID and similar), repeat that payload's UUID from this list exactly.
    ${suppliedPayloadUuids.map((uuid)=>`- ${uuid}`).join('\n    ')}
`;
    }

    let userPrompt = `Given these instructions from an IT admin, generate a ${promptConfig.description}.
${uuidsToUse}
    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``;

    // promptConfig comes back because the action's triage call needs description and firstPartySettingDescription; suppliedPayloadUuids so a caller can check what the model was given.
    return { systemPrompt, userPrompt, promptConfig, suppliedPayloadUuids };

  }


};

