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
      defaultsTo: false,
      description: 'Whether or not to request less information back with the generated profile.'
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Configuration profile generator configuration',
    },

  },


  fn: async function ({profileType, naturalLanguageInstructions, useLighterResponseShape}) {

    let path = require('path');

    // Both Apple schemas are generated from apple/device-management by
    // `sails run regenerate-apple-profile-schemas` rather than maintained here by hand.  The
    // hand-maintained versions gave key names and types only, which stops the model inventing a key but
    // not picking the wrong real one -- and they had already drifted, still listing VPPType after Apple
    // removed it.
    //
    // The whole schema goes in.  That is the difference from the Windows reference below, which has to
    // look up the areas a request needs because its table is 155KB: these render to ~13k and ~6k tokens,
    // so narrowing them would only remove context the model can already see, and add a lookup that can
    // pick wrong.
    let appleSchema;
    if(profileType === 'mobileconfig' || profileType === 'ddm') {

      let filename = profileType === 'ddm' ? 'apple-ddm-declarations.json' : 'apple-payload-manifests.json';
      let schemaFilePath = path.resolve(sails.config.appPath, `profile-generator/schema/${filename}`);
      let schemaFile;
      try {
        schemaFile = require(schemaFilePath);
      } catch (err) {
        throw new Error(
          `Could not read the Apple ${profileType} schema at ${schemaFilePath}.  Run ` +
          `\`sails run regenerate-apple-profile-schemas\` to build it.  Full error: ${err.message}`
        );
      }

      // A nested key renders inside its parent's braces, so no parent can be rendered until every one of
      // its children has been, which is what the recursion below is for: a key's subkeys are rendered on
      // the way to rendering the key that has to quote them.
      //
      // MAX_DEPTH is where nesting stops paying for itself.  Keys at that depth still render, as
      // `Parent{}`, but their contents do not: past three levels the braces describe the shape of Apple's
      // YAML rather than anything the model has to get right.
      const MAX_DEPTH = 3;
      let renderKey = (key, depth)=>{

        // The name, with the suffixes that mark its shape.
        let type = String(key.type || '');
        let shape = /array/.test(type) ? '[]' : (/dictionary/.test(type) ? '{}' : '');
        let labelText = `${key.key}${shape}${key.required ? '*' : ''}`;

        // Whatever constrains the key's value, or nothing when its name is the whole story.  The default
        // rides along with an allowed-value list or a numeric range but not with a gloss, because a gloss
        // is a sentence of Apple's prose and `d=` tacked onto the end of one reads as part of the sentence.
        let constraint;
        if(key.allowedValues && key.allowedValues.length > 0) {
          constraint = `(${key.allowedValues.join('|')})`;
        } else if(key.min !== undefined || key.max !== undefined) {
          constraint = `(${key.min !== undefined ? key.min : ''}-${key.max !== undefined ? key.max : ''})`;
        }
        if(constraint !== undefined) {
          if(key.default !== undefined) {
            constraint = `${constraint} d=${key.default}`;
          }
        } else {
          constraint = key.gloss;
        }

        // How this key reads when it appears inside a parent's braces.  A key with subkeys renders as
        // `Parent{child, child}`, the way the hand-maintained schemas did -- pulling nested keys onto
        // their own lines would lose which parent they belong to.  Note the bare name rather than the
        // label: a parent's shape is already given by its braces.
        let inlineText;
        if(key.subkeys && key.subkeys.length > 0) {
          let renderedSubkeys = [];
          if(depth < MAX_DEPTH) {
            for (let subkey of key.subkeys) {
              renderedSubkeys.push(renderKey(subkey, depth + 1).inlineText);
            }
          }
          inlineText = `${key.key}${key.required ? '*' : ''}{${renderedSubkeys.join(', ')}}`;
        } else {
          inlineText = `${labelText}${constraint ? `:${constraint}` : ''}`;
        }

        return {labelText, constraint, inlineText};
      };

      // Most keys render as a bare name in a comma-separated run, the way the hand-maintained schemas
      // did.  A key earns its own line only when it carries something a name cannot: an allowed-value
      // list, a numeric range, or a sentence of Apple's own prose about what its values mean.  That keeps
      // the cost where it buys something -- SHOWFULLNAME earns a line because false shows a list of
      // users, while allowCamera does not earn one.
      let appleSchemaLines = [];
      for (let entry of schemaFile.entries) {
        appleSchemaLines.push(entry.name);

        let plainKeys = [];
        let keysWithTheirOwnLine = [];
        for (let key of entry.keys) {
          let renderedKey = renderKey(key, 0);
          if(key.subkeys && key.subkeys.length > 0) {
            plainKeys.push(renderedKey.inlineText);
          } else if(renderedKey.constraint) {
            keysWithTheirOwnLine.push(`    ${renderedKey.labelText}  ${renderedKey.constraint}`);
          } else {
            plainKeys.push(renderedKey.labelText);
          }
        }

        if(plainKeys.length > 0) {
          appleSchemaLines.push('    ' + plainKeys.join(', '));
        }
        appleSchemaLines = appleSchemaLines.concat(keysWithTheirOwnLine);
      }

      appleSchema = appleSchemaLines.join('\n');
    }


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
        // Things the admin must do or decide that are not visible in the profile itself.
        // Empty string when there is nothing exceptional, which is the common case.
        "deliveryNotes": "",
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
        description: 'XML .mobileconfig profile that enforces OS settings on macOS/iOS/ipadOS devices',
        firstPartySettingDescription: 'a key in an Apple-published payload',
        providedSchema: appleSchema,
        providedSchemaDescription: `Provided context: every payload type Apple publishes a manifest for, and the keys each one accepts, taken
from Apple's own manifests.  Format is a payload type followed by its keys, where \`*\` marks a required key, \`{}\` is a
dictionary, \`[]\` is an array, and \`Parent{child, child}\` gives a dictionary's contents.  CommonPayloadKeys and
TopLevel are not payload types: the first gives the keys every dict inside PayloadContent may carry, and the second
the keys that belong on the root dict and nowhere else.

Most keys are listed by name alone, which settles their spelling and casing.  A key appears on its own line when it
carries something its name does not: \`(a|b|c)\` are the values it accepts, \`(min-max)\` the range, \`d=\` the default,
and a sentence is Apple's own description of what its values mean.  Where such a sentence is given, it decides the
value -- SHOWFULLNAME reads as though true shows a list of users, and Apple's text says the opposite.

For a listed payload type this is complete: a key not listed under it does not exist in it.  Absence of a payload type
is not proof a domain is unusable -- Apple preference domains with no MDM manifest are managed as preference domains
and are not listed, and neither are third-party domains, which come from the ProfileManifests reference.  What absence
does rule out is a first-party payload type of your own invention.  When a listed payload type covers the setting, use
it, and fall back to an unlisted preference domain only when nothing listed does.  If you cannot recall the keys of an
unlisted preference domain, return the "couldNotGenerateProfile" shape rather than guessing.`,
        references: [
          'First-party Apple payloads: the payload types and their top-level keys are provided below -- https://github.com/apple/device-management/tree/release/mdm/profiles is where a human can check them.',
          'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
        ],
        rules: [
          // Third-party payloads.
          'If this is an attempt to change a third-party application\'s settings, use that application\'s preference domain -- com.google.Chrome, us.zoom.config and its keys must come from the ProfileManifests reference.',
          // Document shape.
          'Emit valid property list XML: the plist DOCTYPE, plist version="1.0", and correctly typed values.',
          'Include PayloadIdentifier, PayloadType, PayloadUUID, PayloadVersion, and PayloadDisplayName on the root dict and on every dict inside PayloadContent.  The root PayloadType is "Configuration" and PayloadVersion is 1.',
          'Keep the plist indented and readable across multiple lines.  Do not collapse it onto one line.',
          // Key fidelity.
          'Apple payload keys are not consistently cased, and the inconsistency is inside a single dict.  The passcode payload uses forcePIN, minLength, and allowSimple -- lowercase first letter -- beside PascalCase PayloadIdentifier and PayloadType.  Reproduce every key exactly as documented for its payload type.  Never normalize casing in either direction.',
          'Use only keys documented for the payload type you chose.  An invented key is written into the profile and nothing downstream rejects it, so the profile looks right and does nothing.',
          'When more than one key in a payload could plausibly satisfy the request, choose by documented meaning, say what the chosen value actually does in valueMeaning, and return couldNotGenerateProfile rather than guessing between them.',
          'When the payload type you need appears in the provided list, copy it and its keys character for character, casing included, and use no key the list does not give it.  A key you remember differently than the list spells it is the list\'s spelling, not yours.',
          'Where the provided list describes what a key\'s values mean, that description decides the value and your own reading of the key name does not.  Several of these keys are named in a way that implies the opposite of what they do.',
          'The provided list does not cover preference domains that have no payload manifest.  If you cannot recall the keys of an unlisted preference domain, do not guess and do not adapt a key from a DDM declaration -- return the "couldNotGenerateProfile" shape and name the payload type you were unsure about.',
          // Value typing.
          'Type every value as plist: <true/> or <false/> for booleans, never <string>true</string>; <integer> for whole numbers; <real> for decimals; <data> with base64 for binary; <date> with an ISO 8601 timestamp.',
          // Dependencies
          'Include the keys the requested setting depends on. minLength and allowSimple enforce nothing unless forcePIN is also set.',
          'Beyond the keys the request names and the keys those depend on, a key\'s presence in the provided list is not a reason to set it.',
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
        providedSchema: appleSchema,
        providedSchemaDescription: `Provided context: every declaration Apple publishes, and every key each one accepts, taken from Apple's
own declaration definitions.  Format is a declaration type followed by its keys, where \`*\` marks a required key,
\`[]\` is an array, and \`Parent{child, child}\` gives a nested dictionary's contents.

Most keys are listed by name alone, which settles their spelling and casing.  A key appears on its own line when it
carries something its name does not: \`(a|b|c)\` are the values it accepts, \`(min-max)\` the range, \`d=\` the default,
and a sentence is Apple's own description of what the value must look like.  Where such a sentence is given, it
decides the value -- ExcludedPaths reads as though it takes any path, and Apple's text says entries are relative to
the home directory and directories need a trailing slash.

This is the complete set.  A declaration type or key that does not appear here does not exist.`,
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

    // Windows is the one profile type whose settings were never provided, only pointed at: the model was
    // given a documentation URL it cannot open and asked to recall node paths, formats and polarities from
    // memory, and it recalled them wrong often enough that CSP passed 12/45 of its cases where the two
    // types with a provided schema passed ~90%.  The whole table is ~155KB, too much to put in front of the
    // model that writes the profile, so the areas this request needs are looked up first and only those go
    // in.  An area averages 587 bytes, so three or four cost less than a tenth of what the whole table would.
    let windowsCspAreasProvided = [];
    if(profileType === 'csp') {

      // Scraped from Microsoft's published reference by `sails run regenerate-windows-csp-policy-nodes`.
      let nodeFilePath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-csp-policy-nodes.json');
      let nodeFile;
      try {
        nodeFile = require(nodeFilePath);
      } catch (err) {
        throw new Error(
          `Could not read the Windows CSP node reference at ${nodeFilePath}.  Run ` +
          `\`sails run regenerate-windows-csp-policy-nodes\` to build it.  Full error: ${err.message}`
        );
      }
      let nodesByArea = _.groupBy(nodeFile.nodes, 'area');

      // Node names rather than area names, deliberately.  Picking from an index of area names alone was
      // measured at 10/27 on the generator's csp cases, against 21/27 for picking from node names:
      // Microsoft's area naming does not follow from what a policy does (RemovableDiskDenyWriteAccess is in
      // Storage, not ADMX_RemovableStorage; the sign-in banner is in LocalPoliciesSecurityOptions, not any
      // of the four areas with "Logon" in the name), so an area-name index asks the model to recall the
      // taxonomy -- which is the failure this whole reference exists to remove.  Given node names it can
      // find the setting and read off the area instead.  The index is ~95KB and identical on every request,
      // so it wants to be a cached prompt prefix.
      let nodeNameIndex = _.map(nodesByArea, (nodesInThisArea, areaName)=>{
        return `${areaName}: ${_.pluck(nodesInThisArea, 'name').sort().join(' ')}`;
      }).join('\n');

      // The small model on purpose: this is a lookup rather than a judgement.
      let picked = await sails.helpers.ai.prompt.with({
        systemPrompt:
`Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not
include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

Below is every Windows Policy CSP node, grouped by the area it belongs to.  An IT admin has asked for a
configuration profile, and another model is about to write it -- but it can only be shown a few areas'
worth of nodes, so your job is to say which areas those should be.

Find the nodes that would actually satisfy the request and name the areas holding them.  Read the node
lists to do it: an area's name frequently does not follow from what its nodes do, so an area that sounds
right often is not the one, and the area that is right often sounds unrelated.

Name three to five areas, not one.  This is a shortlist, not an answer -- the model that writes the
profile picks the node out of what you send, so a second and third candidate cost it nothing, while
naming only the area you thought of first is how the right one gets left out.  After the area you are
most confident in, add the ones holding any other node that could plausibly do what was asked, including
ones you half-rejected.  Every area you name must appear verbatim below; do not invent one.

Return an empty array, naming nothing, when the request is not a Policy CSP request at all.  Windows has
many other CSPs, and some of the most common requests belong to them: provisioning a Wi-Fi network is the
WiFi CSP, a VPN is VPNv2, installing a certificate or an ADMX file has its own CSP too.  The Policy CSP
has adjacent-sounding areas -- Wifi holds AllowWiFi and AllowWiFiDirect -- and naming one of those for a
request that needs a different CSP is worse than naming nothing, because it answers a question that was
not asked and buries the one that was.  An empty array is a real answer here, not a failure.

${nodeNameIndex}

Respond in JSON with this data shape:
{
  "areas": ["TODO"]
}`,
        prompt: `Here are the instructions from an IT admin:
\`\`\`
${naturalLanguageInstructions}
\`\`\``,
        baseModel: 'claude-haiku-4-5',
        expectJson: true,
      })
      .tolerate((err)=>{
        // Not fatal: the generator falls back to the prompt it had before this reference existed, which is
        // worse but still generates.  Failing here would turn a degraded profile into no profile.
        sails.log.warn(`When trying to work out which Windows CSP areas a request touches, an error occurred.  The profile will be generated without the node reference.  Full error: ${require('util').inspect(err, {depth: 2})}`);
        return undefined;
      });

      // Filtered against the real area names rather than trusted: the model invents plausible ones
      // (Telemetry, WinLogon, WindowsStore are all things it has asked for and none of them exist).  An
      // invented name is dropped rather than thrown on, since a dropped name costs detail the model then
      // has to abstain over, while an error would cost the whole profile.
      if(picked && _.isArray(picked.areas)) {
        let realAreaNames = Object.keys(nodesByArea);
        for (let candidate of picked.areas) {
          let matched = _.find(realAreaNames, (areaName)=>{ return areaName.toLowerCase() === String(candidate).toLowerCase(); });
          if(matched && !_.contains(windowsCspAreasProvided, matched)) {
            windowsCspAreasProvided.push(matched);
          }
        }
      }

      if(windowsCspAreasProvided.length > 0) {

        // Rendered in area order, off a copy: windowsCspAreasProvided goes back to the caller in the order
        // the model named the areas, and sorting it in place would throw that away.
        //
        // Value descriptions are the reason this reference exists -- a node like DevicePasswordEnabled takes
        // 0 to mean enabled, and no amount of prose in the prompt stops a model inferring the opposite from
        // the name -- so they are kept, but trimmed to their first clause.  The full sentence is
        // documentation, not a constraint.
        let schemaLines = [];
        for (let areaName of _.clone(windowsCspAreasProvided).sort()) {
          schemaLines.push(areaName);
          for (let node of _.sortBy(nodesByArea[areaName], 'name')) {
            let parts = [node.name, node.format];
            if(!_.contains(node.scopes, 'Device')) {
              parts.push('@User');
            }
            if(node.mustBeWrappedInAtomic) {
              parts.push('atomic');
            }
            if(node.allowedValues && node.allowedValues.length > 0) {
              // Six is enough for every boolean and every small enum.  The handful of nodes with longer
              // value lists are ones whose meaning a line of prompt was never going to settle anyway.
              parts.push(_.map(node.allowedValues.slice(0, 6), (allowedValue)=>{
                let firstClause = String(allowedValue.description || '').trim().replace(/\.$/, '').split(/(?<=[a-z])\.\s/)[0];
                let summarizedDescription = _.trunc(firstClause.replace(/\s+/g, ' '), {length: 52, omission: ''}).trim().replace(/[,;:]$/, '');
                return `${allowedValue.value}${allowedValue.isDefault ? '*' : ''}=${summarizedDescription}`;
              }).join(' '));
            } else if(node.defaultValue !== undefined) {
              parts.push(`d=${node.defaultValue}`);
            }
            if(node.dependsOn) {
              parts.push(`needs ${node.dependsOn.locUri.split('/').slice(-2).join('/')}=${node.dependsOn.allowedValue}`);
            }
            schemaLines.push('  ' + parts.join(' '));
          }
        }

        promptConfig.providedSchemaDescription =
`Provided context: the Windows Policy CSP nodes this request looks like it needs, straight from
Microsoft's published reference.  A node's LocURI is ./Device/Vendor/MSFT/Policy/Config/<Area>/<NodeName>
-- never a shortened form of that path, and never an area segment you inferred from a policy's name.

Format is an area, then one node per line:
  <NodeName> <format> [flags] [values or default] [dependency]
\`*\` marks a value as that node's default.  \`@User\` marks a node that exists only under ./User/ --
writing it under ./Device/ deploys cleanly and enforces nothing.  \`atomic\` marks a node Microsoft
documents as requiring an <Atomic> wrapper.  \`needs X=Y\` is a node the setting depends on, which
belongs in the profile alongside it.  The format on each line is the node's declared DFFormat: emit it
verbatim and make <Data> legal for it, rather than reasoning about the value's type from its name.

This is authoritative for the areas below and settles their node names, paths, formats and values.  It is
not the whole Policy CSP, and what is missing decides between two different answers.  If no node here
enforces what was asked, return the "couldNotGenerateProfile" shape and name what you were looking for,
rather than recalling a node from an area you were not given.  But if a node here does enforce it and the
request merely describes the effect in words the node does not use -- asking to wipe after failed
passcode attempts, where the node sets the failed-attempt threshold that produces that outcome -- then
that is the node, so use it and put what it actually does, and anything the admin should know about the
gap, in "valueMeaning" and "caveats".  Abstaining is for a setting you cannot find, not for one whose
published name is less specific than the request.

Every area that exists, with its node count, is listed after the detail.  Use it only to tell whether
a setting you cannot find lives somewhere that was not provided -- the names alone do not tell you what
is in an area, and an area's name is often not what you would guess.`;

        promptConfig.providedSchema = `${schemaLines.join('\n')}\n\nAll areas: ${_.map(Object.keys(nodesByArea).sort(), (areaName)=>{ return `${areaName}(${nodesByArea[areaName].length})`; }).join(' ')}`;

        // Ahead of the existing rules on purpose.  Several of those tell the model how to decide a format
        // or a polarity from memory -- sound advice when nothing was provided, and a licence to override
        // the provided data if it is left to rank itself against them.
        promptConfig.rules = [
          'Every node name, LocURI, format and allowed value in the provided list is authoritative.  Where the list and your own recollection differ, the list is right and you are wrong -- copy the path, the format and the value from it character for character.  The rules below about choosing a format or working out what a value means apply only to a setting the list does not cover.',
        ].concat(promptConfig.rules);
        // The URL stays useful as somewhere a human can check the work, but it is no longer where the
        // model is being told to get the settings from.
        promptConfig.references = [
          'Windows CSP nodes, formats, and allowed values: the nodes for this request are provided below -- https://learn.microsoft.com/en-us/windows/client-management/mdm/ is where a human can check them.',
        ];
      }
    }

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
${promptConfig.providedSchemaDescription}
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
    // windowsCspAreasProvided so a caller can tell a profile that was written from the wrong areas from one
    // written from the right areas badly -- the two look identical in the generated profile, and the first
    // is a lookup problem while the second is a prompt problem.
    return { systemPrompt, userPrompt, promptConfig, suppliedPayloadUuids, windowsCspAreasProvided };

  }


};

