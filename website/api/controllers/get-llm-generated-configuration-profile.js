module.exports = {


  friendlyName: 'Get llm generated configuration profile',


  description: '',


  inputs: {
    profileType: {
      type: 'string',
      isIn: [
        'mobileconfig',
        'csp',
        'ddm',
      ],
      required: true
    },
    naturalLanguageInstructions: {
      type: 'string',
      required: true
    }
  },


  exits: {
    success: {
      description: 'A configuration profile was successfully generated for a user.',
    },
    couldNotGenerateProfile: {
      description: 'A configuration profile could not be generated for a user using the provided instructions.',
      responseType: 'badRequest'
    }
  },


  fn: async function ({profileType, naturalLanguageInstructions}) {

    // Generate a random room name.
    let roomId = await sails.helpers.strings.random();
    if(this.req.isSocket) {
      // Add the requesting socket to the room.
      sails.sockets.join(this.req, roomId);
    }

    // Rules that apply to every profile type.  Ordered by consequence: a violation of an
    // early rule produces a profile that deploys cleanly and does nothing.
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

    // "deliveryNotes" defaults to noise unless it is aggressively constrained.  An empty
    // string is a weaker affordance than an empty array, so these rules carry more of the load.
    let deliveryNotesRules = [
      '"deliveryNotes" is for exceptions only: something the admin has to do or decide that is not visible in the profile itself.  Use an empty string when nothing applies.  An empty string is the right answer for most profiles -- prefer it whenever you are unsure whether a note earns its place.',
      'Never write a sentence stating that a condition does not apply.  "No credentials or secrets are embedded" and "no companion declaration is required" are not notes -- leaving them out already says that.',
      'Never restate what the profile is, which platform it targets, or how that platform is normally delivered.  The admin chose the format and already knows.',
      'One sentence per action, addressed to the admin, and no more than two sentences in total.  For example: "Replace the passphrase with a secret variable before committing this to a repository."',
    ];

    let promptConfigByProfileType = {

      'csp': {
        description: 'CSP XML profile that enforces OS settings on Windows devices',
        // Reference corpora, mirrored from the Configuration Profiles and DDM sections of
        // .claude/skills/fleet-gitops/SKILL.md at the repo root.  Kept in sync by hand: the
        // skill is written for an agent that can fetch these, and will evolve for that
        // purpose, so it is not a safe runtime dependency for this prompt.
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
        references: [
          'First-party Apple payloads: https://github.com/apple/device-management/tree/release/mdm/profiles',
          'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
        ],
        rules: [
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
          'PayloadUUID is a distinct uppercase UUID in 8-4-4-4-12 form on every dict, including the root.  Two dicts sharing a UUID is a profile that installs unpredictably.',
          'PayloadIdentifier is reverse-DNS.  Each payload dict\'s identifier is the root identifier plus a distinguishing suffix, and no two identifiers in the profile are the same.',
          'PayloadDisplayName on the root is what an end user sees in System Settings, and some MDMs use it as the profile name.  Make it human-readable and specific to what the profile does.',
          // Structure.
          'Put every key for one payload domain in a single dict inside PayloadContent.  Do not emit several dicts with the same PayloadType.',
          // Third-party payloads.
          'For a third-party application\'s settings, PayloadType is that application\'s preference domain -- com.google.Chrome, us.zoom.config, and so on -- and its keys come from the ProfileManifests reference rather than Apple\'s payload reference.',
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
    let numberedRules = sharedRules.concat(promptConfig.rules, deliveryNotesRules)
      .map((rule, idx)=>`${idx + 1}. ${rule}`)
      .join('\n    ');


    let systemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

You generate a ${promptConfig.description} from an IT admin's instructions.

Draw setting names, types, and allowed values from these published references:
${promptConfig.references.map((reference)=>`- ${reference}`).join('\n    ')}

When generating the profile:
${numberedRules}

Respond in JSON with this data shape:
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
  // Explain why a profile could not be generated, naming the specific setting, node, or key you could not confirm.
  "reasonWhyAProfileCouldNotBeGenerated": TODO
}
`;


    let configurationProfilePrompt = `Given these instructions from an IT admin, generate a ${promptConfig.description}.

    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``;
    // console.log(configurationProfilePrompt);
    let configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
      systemPrompt: systemPrompt,
      prompt: configurationProfilePrompt,
      baseModel: 'claude-sonnet-5',
      expectJson: true,
    })
    .intercept((err)=>{
      sails.log.warn(`When trying generate a configuration profile for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      if(this.req.isSocket){
        // If this request was from a socket and an error occurs, broadcast an 'error' event and unsubscribe the socket from this room.
        sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateProfile'});
        sails.sockets.leave(this.req, roomId);
      }
      return 'couldNotGenerateProfile';
    });
    // sails.log(configurationProfileGenerationResult);
    // let jsonResult = JSON.parse(configurationProfileGenerationResult);
    // console.log(configurationProfileGenerationResult);
    // All done.
    if(
      configurationProfileGenerationResult.couldNotGenerateProfile ||
      !configurationProfileGenerationResult.configurationProfile ||
      !configurationProfileGenerationResult.profileFilename ||
      !configurationProfileGenerationResult.settingsEnforced
    ) {
      if(this.req.isSocket){
        sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateProfile'});
        sails.sockets.leave(this.req, roomId);
        return;
      } else {
        throw 'couldNotGenerateProfile';
      }
    }

    let generatedProfile = {
      profile: configurationProfileGenerationResult.configurationProfile,
      profileFilename: configurationProfileGenerationResult.profileFilename,
      deliveryNotes: configurationProfileGenerationResult.deliveryNotes,
      items: configurationProfileGenerationResult.settingsEnforced
    };

    // If this request was from a socket, we'll broadcast a 'profileGenerated' event with the generated profile and unsubscribe the socket.
    if(this.req.isSocket){
      sails.sockets.broadcast(roomId, 'profileGenerated', {result: generatedProfile});
      sails.sockets.leave(this.req, roomId);
    } else {
      // Otherwise, return the generated profile as JSON.
      return generatedProfile;
    }

  }


};
