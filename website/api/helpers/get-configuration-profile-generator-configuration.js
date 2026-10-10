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
    },

    useApplePayloadTypeLookup: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether or not to send a lookup prompt first that narrows the .mobileconfig schema to the payload types this request needs.',
      extendedDescription: 'Without it, the full .mobileconfig schema is provided, without key descriptions.'
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Configuration profile generator configuration',
    },

  },


  fn: async function ({profileType, naturalLanguageInstructions, useLighterResponseShape, useApplePayloadTypeLookup}) {

    let path = require('path');

    // Both Apple schemas are generated from apple/device-management by
    // `sails run regenerate-apple-profile-schemas` rather than maintained here by hand.  The
    // hand-maintained versions gave key names and types only, which stops the model inventing a key but
    // not picking the wrong real one -- and they had already drifted, still listing VPPType after Apple
    // removed it.
    //
    // The DDM schema goes in whole: it renders to ~6k tokens.  The .mobileconfig one is looked up first,
    // the way the Windows reference below is.  Whole, it rendered to ~13k tokens of mostly bare key names,
    // and the model still picked keys by what their names suggested -- AutomaticallyInstallAppUpdates
    // under com.apple.appstore, a `position` key read off position-immutable -- because nothing next to a
    // name said what the key does.  Narrowed to the few payload types a request needs, every key can carry
    // Apple's description of it.
    let appleSchema;
    let appleSchemaDescription;
    let applePayloadTypesProvided = [];
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
        // their own lines would lose which parent they belong to.
        //
        // An array's subkeys describe its ELEMENT rather than keys inside it, and Apple gives that element
        // a name of its own: PayloadContentItem, RangesItem, AllowedExtensionsItem.  Rendering that name
        // as though it were a key both invites the model to write it into the profile and, since braces
        // mean dictionary here, describes an array as a dictionary.  So an array keeps its `[]` and its
        // element is unwrapped: a dictionary element contributes its own keys, and a plain-value element
        // contributes whatever constrains its values.
        let inlineText;
        let arrayElement = (shape === '[]' && key.subkeys && key.subkeys.length > 0) ? _.first(key.subkeys) : undefined;
        // Checked by type rather than by "has subkeys": ACME and SCEP subjects are arrays whose element is
        // itself an array, and those render as the plain `Subject[]` the hand-maintained schemas gave them
        // rather than growing a brace that would claim they are dictionaries.
        let arrayElementIsADictionary = arrayElement && /dictionary/.test(String(arrayElement.type || '')) && arrayElement.subkeys && arrayElement.subkeys.length > 0;
        if(arrayElementIsADictionary) {
          let renderedElementKeys = [];
          if(depth < MAX_DEPTH) {
            for (let elementKey of arrayElement.subkeys) {
              renderedElementKeys.push(renderKey(elementKey, depth + 1).inlineText);
            }
          }
          inlineText = `${labelText}{${renderedElementKeys.join(', ')}}`;
        } else if(arrayElement) {
          let elementConstraint = renderKey(arrayElement, depth).constraint;
          inlineText = `${labelText}${elementConstraint ? `:${elementConstraint}` : ''}`;
        } else if(key.subkeys && key.subkeys.length > 0) {
          // A dictionary: the bare name rather than the label, since its braces already give its shape.
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
      //
      // With describeKeys, which is only affordable once the payload types have been narrowed, every key
      // gets a line and Apple's description of it.  A gloss wins over the description where there is one,
      // since glosses are kept longer precisely because the constraint is in their second sentence.
      let renderEntries = (entries, {describeKeys})=>{
        let lines = [];
        for (let entry of entries) {
          lines.push(entry.name);

          let plainKeys = [];
          let keysWithTheirOwnLine = [];
          for (let key of entry.keys) {
            let renderedKey = renderKey(key, 0);
            let hasSubkeys = key.subkeys && key.subkeys.length > 0;
            if(describeKeys) {
              let valueConstraint = renderedKey.constraint !== key.gloss ? renderedKey.constraint : undefined;
              keysWithTheirOwnLine.push('    ' + _.compact([
                hasSubkeys ? renderedKey.inlineText : renderedKey.labelText,
                valueConstraint,
                key.gloss || key.description,
              ]).join('  '));
            } else if(hasSubkeys) {
              plainKeys.push(renderedKey.inlineText);
            } else if(renderedKey.constraint) {
              keysWithTheirOwnLine.push(`    ${renderedKey.labelText}  ${renderedKey.constraint}`);
            } else {
              plainKeys.push(renderedKey.labelText);
            }
          }

          if(plainKeys.length > 0) {
            lines.push('    ' + plainKeys.join(', '));
          }
          lines = lines.concat(keysWithTheirOwnLine);
        }
        return lines.join('\n');
      };

      let ALWAYS_PROVIDED_ENTRY_NAMES = ['CommonPayloadKeys', 'TopLevel'];
      if(profileType === 'mobileconfig' && useApplePayloadTypeLookup) {

        // Key names are in the index, not just titles and descriptions, for the reason the Windows lookup
        // gives: picking from names and descriptions alone means recalling Apple's taxonomy, which is the
        // failure this exists to remove.  Asked about App Store app updates, a description-only index
        // offers com.apple.appstore ("configures macOS App Store restrictions") and nothing pointing at
        // com.apple.SoftwareUpdate, which is where AutomaticallyInstallAppUpdates is.  The index is ~35KB
        // and identical on every request, so it wants to be a cached prompt prefix.
        let payloadTypeIndex = _.map(_.reject(schemaFile.entries, (entry)=>{ return _.contains(ALWAYS_PROVIDED_ENTRY_NAMES, entry.name); }), (entry)=>{
          let about = _.compact([entry.title].concat(entry.platforms || [])).join('; ');
          return `${entry.name} (${about}): ${entry.description || ''} Keys: ${_.pluck(entry.keys, 'key').join(' ')}`;
        }).join('\n');

        // The small model on purpose: this is a lookup rather than a judgement, and it sits in front of the
        // generation, so every second it takes is a second added to the request.
        let picked = await sails.helpers.ai.prompt.with({
          systemPrompt:
`Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not
include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

Below is every payload type Apple publishes a .mobileconfig manifest for: its name, its title and the
platforms it applies to, Apple's description of it, and the top-level keys it accepts.  An IT admin has
asked for a configuration profile, and another model is about to write it -- but it will only be shown
the payload types you name, so your job is to say which those should be.

Find the keys that would actually satisfy the request and name the payload types that list them.  Read
the key lists to do it: a payload type whose name matches the topic of the request is often not the one
holding the key, and the one that does can sound unrelated.

Name one to four payload types.  This is a shortlist, not an answer -- the model that writes the profile
picks keys out of what you send, so a second candidate costs it little, while naming only the payload
type you thought of first is how the right one gets left out.  Every name must appear verbatim below; do
not invent one.

Return an empty array when no payload type below could satisfy the request -- a third-party application's
settings, for one.  An empty array is a real answer here, not a failure.

${payloadTypeIndex}

Respond in JSON with this data shape:
{
  "payloadTypes": ["TODO"]
}`,
          prompt: `Here are the instructions from an IT admin:
\`\`\`
${naturalLanguageInstructions}
\`\`\``,
          baseModel: 'claude-haiku-5-5',
          expectJson: true,
        })
        .tolerate((err)=>{
          // Not fatal: the full schema below is what the generator used before this lookup existed.
          sails.log.warn(`When trying to work out which Apple payload types a request needs, an error occurred.  The profile will be generated from the full schema.  Full error: ${require('util').inspect(err, {depth: 2})}`);
          return undefined;
        });

        // Filtered against the real names rather than trusted, as the Windows lookup is.  The index line's
        // "(title; platforms)" is cut off first, since the model sometimes copies it: "com.apple.MCX (Time
        // Server; macOS)" matched nothing and was dropped, and with it the request's whole schema.
        if(picked && _.isArray(picked.payloadTypes)) {
          let realEntryNames = _.uniq(_.pluck(schemaFile.entries, 'name'));
          for (let candidate of picked.payloadTypes) {
            let candidateName = String(candidate).split(' (')[0].trim();
            let matched = _.find(realEntryNames, (entryName)=>{ return entryName.toLowerCase() === candidateName.toLowerCase(); });
            if(matched && !_.contains(ALWAYS_PROVIDED_ENTRY_NAMES, matched) && !_.contains(applePayloadTypesProvided, matched)) {
              applePayloadTypesProvided.push(matched);
            }
          }
        }

        // Restrictions is Apple's catch-all -- 210 keys under a title and description that match almost no
        // request -- so the lookup passes it over for payloads whose titles sound closer (Content Caching
        // Service over allowContentCaching, Time Server over forceAutomaticDateAndTime).  Software Update is
        // missed the same way: asked to install security responses and system data files, the lookup names
        // Restrictions for allowRapidSecurityResponseInstallation and leaves out ConfigDataInstall and
        // CriticalUpdateInstall.  Added only when the lookup found something, so a request it found nothing
        // for still falls back to the full schema.
        if(applePayloadTypesProvided.length > 0) {
          applePayloadTypesProvided = _.union(applePayloadTypesProvided, ['com.apple.applicationaccess', 'com.apple.SoftwareUpdate']);
        }


        // sails.log.warn(`[payload-type lookup] instructions: ${JSON.stringify(naturalLanguageInstructions)} | lookup returned: ${JSON.stringify(picked ? picked.payloadTypes : '(lookup failed)')} | provided: ${JSON.stringify(applePayloadTypesProvided)}`);
      }

      if(applePayloadTypesProvided.length > 0) {
        // Every entry with a picked name, since six payloads share com.apple.MCX.
        let entriesToRender = _.filter(schemaFile.entries, (entry)=>{
          return _.contains(ALWAYS_PROVIDED_ENTRY_NAMES, entry.name) || _.contains(applePayloadTypesProvided, entry.name);
        });
        appleSchema = `${renderEntries(entriesToRender, {describeKeys: true})}\n\nAll payload types: ${_.uniq(_.pluck(schemaFile.entries, 'name')).sort().join(' ')}`;
        appleSchemaDescription =
`Provided context: the payload types this request looks like it needs, and every key each one accepts,
taken from Apple's own manifests.  Format is a payload type followed by one key per line: the key, what
constrains its value where anything does -- \`(a|b|c)\` the values it accepts, \`(min-max)\` the range, \`d=\`
the default -- and Apple's description of the key.  \`*\` marks a required key, \`{}\` is a dictionary, \`[]\` is
an array, \`Parent{child, child}\` gives a dictionary's contents, and \`Parent[]{child, child}\` an array whose
elements are dictionaries with those keys.  CommonPayloadKeys and TopLevel are not payload types: the
first gives the keys every dict inside PayloadContent may carry, and the second the keys that belong on
the root dict and nowhere else.

Choose keys by their descriptions, not their names.  Apple's description decides what a key does and what
its values mean -- SHOWFULLNAME reads as though true shows a list of users, and Apple's text says the
opposite -- and a key whose name resembles the setting you want can describe something else entirely.

For each payload type below this is complete: a key not listed under it does not exist in it.  It is not
every payload type -- the rest are named, without their keys, after the detail.  If the setting belongs to
one of those, return the "couldNotGenerateProfile" shape and name it, rather than recalling its keys.  A
domain in neither list is an Apple preference domain with no MDM manifest, or a third-party domain from
the ProfileManifests reference.  Fall back to one of those only when nothing listed covers the setting,
and if you cannot recall its keys, return the "couldNotGenerateProfile" shape rather than guessing.`;
      } else {
        appleSchema = renderEntries(schemaFile.entries, {describeKeys: false});
        appleSchemaDescription =
`Provided context: every payload type Apple publishes a manifest for, and the keys each one accepts, taken
from Apple's own manifests.  Format is a payload type followed by its keys, where \`*\` marks a required key, \`{}\` is a
dictionary, \`[]\` is an array, \`Parent{child, child}\` gives a dictionary's contents, and \`Parent[]{child, child}\` an
array whose elements are dictionaries with those keys.  An array of plain values is just \`Name[]\`, with what
constrains its entries after a colon where there is anything to say: \`Name[]:(a|b)\`.  CommonPayloadKeys and
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
unlisted preference domain, return the "couldNotGenerateProfile" shape rather than guessing.`;
      }
    }


    // The tail of the system prompt.
    let RESPONSE_SHAPE;
    if(!useLighterResponseShape) {

      RESPONSE_SHAPE = `Respond in JSON with this data shape:
      {
        "settingsEnforced": [// For each setting enforced by the configuration profile.
          {
            // The name (key) of the setting that is enforced. e.g., LoginwindowText
            name: "TODO",
            // The value of the setting that is enforced
            value: "TODO",
            // Where this setting comes from: the CSP node path, the Apple payload domain and key, or the declaration type.
            schemaReference: "TODO",
          },
          {...}
        ]
        "profileFilename": "TODO",
        "configurationProfile": "TODO",
        // Things the admin must do or decide that are not visible in the profile itself.
        // Empty string when there is nothing exceptional, which is the common case.
        "deliveryNotes": "",
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
          'Use Replace for Policy CSP leaf nodes that hold a value.  Reserve Add for nodes that do not exist until you create them, such as ADMXInstall, certificate installs, and WiFi or VPN profile instances.  A WiFi profile instance is created by the Add of its WlanXml leaf -- never Add the WiFi/Profile/{SSID} node on its own.  When a node\'s AccessType permits both, choose Replace, because the profile may reach a host where the value is already set and Add can fail there.',
          'Only use an Add or Replace verb on a node whose AccessType permits it.  Roughly 500 leaf nodes accept only Get, and a Replace against one is accepted, deploys, and then fails on the device.',
          'Include the nodes the requested setting depends on.  DeviceLock/MinDevicePasswordLength has no effect unless DeviceLock/DevicePasswordEnabled is also set.',
          'Beyond the nodes the request names and the nodes those depend on, a node\'s presence in the provided list is not a reason to set it.  A WiFi profile never needs ProfileSource, for one: it defaults to Enterprise.',
          'Record the minimum OS build and the Windows editions each node applies to in "caveats".  A valid setting aimed at the wrong SKU is a silent no-op, not an error.',
          // Embedded values.
          'When a node\'s value is embedded XML -- WiFi/Profile/*/WlanXml, ADMXInstall, and similar -- wrap it in <![CDATA[ ... ]]> rather than escaping it as entities, and emit that value as a single line with no line breaks or indentation between its elements.  This applies only to the embedded value; the surrounding SyncML keeps its normal indentation.',
          'Be consistent within a profile about optional <Meta> children.  If you emit <Type> for one item, emit it for all of them, and namespace every <Meta> child as xmlns="syncml:metinf".',
          'For WiFi profiles, emit <hex> and then <name> inside <SSID> -- the WLAN_profile schema enforces element order, and an <SSID> with <name> first is rejected.  Take <hex> from the encodings supplied with the instructions.  Never work a hex encoding out yourself: Windows and some MDMs treat the hex form as authoritative when both are present, so one wrong byte connects to nothing.  If no encoding was supplied for the SSID, emit <name> alone.',
          'The WLAN_profile schema is not in the provided list, and it enforces element order and closed value lists, so build WlanXml from these facts rather than from memory.  Security settings sit at WLANProfile > MSM > security, with authEncryption (authentication, encryption, useOneX) first.  A personal network (WPA2PSK, WPA3SAE) follows it with sharedKey (keyType, protected, keyMaterial); an enterprise network (useOneX true) has no sharedKey and carries a OneX element instead.  <encryption> accepts only none, WEP, TKIP, AES, or GCMP256 -- WPA2 and WPA3SAE networks use AES, never CCMP, which is the 802.11 name for the same cipher.',
        ],
      },

      'mobileconfig': {
        description: 'XML .mobileconfig profile that enforces OS settings on macOS/iOS/ipadOS devices',
        providedSchema: appleSchema,
        providedSchemaDescription: appleSchemaDescription,
        references: [
          'First-party Apple payloads: the payload types and their top-level keys are provided below -- https://github.com/apple/device-management/tree/release/mdm/profiles is where a human can check them.',
          'Third-party Apple payloads: https://github.com/ProfileManifests/ProfileManifests',
        ],
        rules: [
          'A configuration profile sets preferences and restrictions. It cannot start, stop, load, or unload a system service or daemon, run a command, change a file or its permissions, or change a local account\'s state. If the only way to satisfy the request is one of those, return `couldNotGenerateProfile` and name the mechanism that does it (a script, launchctl, systemsetup).',
          'A key that stops users changing a setting (an allow*Modification key) leaves the setting in whatever state it is already in.  When the request is to turn that setting on or off and the lock is the only documented key for it, generate the lock and say in "deliveryNotes" that it does not change the current state, naming what does (a script, launchctl, systemsetup).  Hiding a whole System Settings pane with DisabledSystemSettings is not a lock on one setting, so never use it to stand in for one.',
          // Third-party payloads.
          'If this is an attempt to change a third-party application\'s settings, use that application\'s preference domain -- com.google.Chrome, us.zoom.config and its keys must come from the ProfileManifests reference.',
          // Document shape.
          'Emit valid property list XML: the plist DOCTYPE, plist version="1.0", and correctly typed values.',
          'Include PayloadIdentifier, PayloadType, PayloadUUID, PayloadVersion, and PayloadDisplayName on the root dict and on every dict inside PayloadContent.  The root PayloadType is "Configuration" and PayloadVersion is 1.',
          'Keep the plist indented and readable across multiple lines.  Do not collapse it onto one line.',
          // Key fidelity.
          'Apple payload keys are not consistently cased, and the inconsistency is inside a single dict.  The passcode payload uses forcePIN, minLength, and allowSimple -- lowercase first letter -- beside PascalCase PayloadIdentifier and PayloadType.  Reproduce every key exactly as documented for its payload type.  Never normalize casing in either direction.',
          'Use only keys documented for the payload type you chose.  An invented key is written into the profile and nothing downstream rejects it, so the profile looks right and does nothing.',
          'Choose the payload type by finding the key, not by the payload type\'s name.  Locate the key that does what was asked in the provided list, then use the payload type it is listed under -- even when another payload type\'s name sounds closer to the request.  App Store app updates are AutomaticallyInstallAppUpdates in com.apple.SoftwareUpdate, not anything in com.apple.appstore.  Before returning, confirm every key in each PayloadContent dict is listed under that dict\'s PayloadType.',
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
          'PayloadDisplayName on the root is what an end user sees in System Settings, and some MDMs use it as the profile name.  Make it human-readable and specific to what the profile actually enforces, which is not always what was asked for: a profile that only sets allowBluetoothModification is "Bluetooth Settings Locked", not "Disable Bluetooth".  PayloadDescription follows the same rule.',
          // Structure.
          'Put every key for one payload domain in a single dict inside PayloadContent.  Do not emit several dicts with the same PayloadType.',


        ],
      },

      'ddm': {
        description: 'Apple DDM declaration in JSON format that enforces OS settings on macOS devices',
        providedSchema: appleSchema,
        providedSchemaDescription: `Provided context: every declaration Apple publishes, and every key each one accepts, taken from Apple's
own declaration definitions.  Format is a declaration type followed by its keys, where \`*\` marks a required key,
\`[]\` is an array, \`Parent{child, child}\` gives a nested dictionary's contents, and \`Parent[]{child, child}\` an array
whose elements are dictionaries with those keys.  An array of plain values is just \`Name[]\`, with what constrains its
entries after a colon where there is anything to say: \`Name[]:(a|b)\`.

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
          'A declaration file holds exactly one declaration: one JSON object with Type, Identifier, and Payload.  Never return an array or several objects.  Settings that share a declaration type go in that one Payload.  When the request needs more than one declaration type, return the "couldNotGenerateProfile" shape, name each declaration type the request needs, and say that each one must be generated and uploaded as its own file.',
        ],
      },

    };


    let promptConfig = promptConfigByProfileType[profileType];

    // Windows is the one profile type whose settings were never provided, only pointed at: the model was
    // given a documentation URL it cannot open and asked to recall node paths, formats and polarities from
    // memory, and it recalled them wrong often enough that CSP passed 12/45 of its cases where the two
    // types with a provided schema passed ~90%.  The whole table renders to ~310KB, too much to put in front
    // of the model that writes the profile, so the areas this request needs are looked up first and only
    // those go in.  An area averages under 1KB, and the largest (VPNv2) is ~18KB.
    let windowsCspAreasProvided = [];
    let admxTemplatesProvided = [];
    if(profileType === 'csp') {

      // Scraped from Microsoft's published reference by `sails run regenerate-windows-csp-nodes`.
      let nodeFilePath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-csp-nodes.json');
      let nodeFile;
      try {
        nodeFile = require(nodeFilePath);
      } catch (err) {
        throw new Error(
          `Could not read the Windows CSP node reference at ${nodeFilePath}.  Run ` +
          `\`sails run regenerate-windows-csp-nodes\` to build it.  Full error: ${err.message}`
        );
      }
      // Third-party apps' policies (Chrome, Firefox, ...) exist as CSP nodes only once their ADMX template is
      // ingested, so Microsoft's reference has none of them.  They are merged in as Policy areas named the way
      // Windows names them after ingestion -- GoogleChrome~Policy~googlechrome~Startup -- so the lookup, keyword
      // search and rendering below treat them like any other area.
      let admxTemplateFilePath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-admx-templates.json');
      let admxTemplateFile;
      try {
        admxTemplateFile = require(admxTemplateFilePath);
      } catch (err) {
        throw new Error(
          `Could not read the Windows ADMX template reference at ${admxTemplateFilePath}.  Run ` +
          `\`sails run regenerate-windows-admx-templates\` to build it.  Full error: ${err.message}`
        );
      }
      // Index the templates by id, so a policy's template can be looked up from its admxTemplate value.
      let admxTemplatesById = _.indexBy(admxTemplateFile.templates, 'id');
      // Add the templates' policies to the Windows CSP nodes, in a copy, since `require` hands every caller the same cached object.
      nodeFile = Object.assign({}, nodeFile, { nodes: nodeFile.nodes.concat(admxTemplateFile.nodes) });

      // A template's policies are offered only when the request names its app.  They share most of their
      // words with Windows' own (Firefox has a DisableTelemetry beside System/AllowTelemetry), and a request
      // that names no app is asking about Windows, so offering them there only gives the lookup and the
      // keyword search a lookalike to pick.  It also leaves every other request's index exactly as it was.

      let lowercaseInstructions = naturalLanguageInstructions.toLowerCase();
      // Find the templates whose keywords the request mentions, matched as whole words.
      let admxTemplateIdsTheRequestNames = _.pluck(_.filter(admxTemplateFile.templates, (template)=>{
        return _.any(template.keywords || [], (keyword)=>{ return new RegExp(`\\b${_.escapeRegExp(keyword)}\\b`).test(lowercaseInstructions); });
      }), 'id');
      // Policy CSP nodes are grouped by area, since that is how the Policy CSP is organized and how its
      // LocURIs are built.  Every other CSP is one group of its own, suffixed so it cannot collide with a
      // Policy area: DeviceLock, Defender, Update and Wifi are each both, and matching is case-insensitive.
      //
      // A standalone CSP's Get-only nodes are dropped here, since no profile can set them: DevDetail,
      // DeviceStatus and the other inventory CSPs are nothing but, and they are half of the largest CSPs.
      // Every Policy CSP node is settable.
      // Chrome publishes each policy twice, once enforced and once under *_recommended as a default the user
      // can change.  Offered both, the lookup named the recommended twin for "make Chrome reopen the last
      // session", so it is only offered when the request asks for a default.
      let requestAsksForADefault = /\b(recommend\w*|default|suggest\w*|user can change|users can change)\b/.test(lowercaseInstructions);
      // Keep the nodes a profile can set: a template's policies only when the request names its app (leaving out its _recommended policies unless the request asks for a default), and every other node a profile can write to.
      let nodesAProfileCanSet = _.filter(nodeFile.nodes, (node)=>{
        if(node.admxTemplate) {
          return _.contains(admxTemplateIdsTheRequestNames, node.admxTemplate) && (requestAsksForADefault || !/_recommended(~|$)/.test(node.area));
        }
        return node.csp === 'Policy' || /Add|Replace|Exec/.test(node.accessType || '');
      });
      // Group the nodes by Policy CSP area, or by CSP for every other CSP, e.g., DeviceLock or "WiFi CSP".
      let nodesByArea = _.groupBy(nodesAProfileCanSet, (node)=>{ return node.csp === 'Policy' ? node.area : `${node.csp} CSP`; });

      // Node names rather than area names, deliberately.  Picking from an index of area names alone was
      // measured at 10/27 on the generator's csp cases, against 21/27 for picking from node names:
      // Microsoft's area naming does not follow from what a policy does (RemovableDiskDenyWriteAccess is in
      // Storage, not ADMX_RemovableStorage; the sign-in banner is in LocalPoliciesSecurityOptions, not any
      // of the four areas with "Logon" in the name), so an area-name index asks the model to recall the
      // taxonomy -- which is the failure this whole reference exists to remove.  Given node names it can
      // find the setting and read off the area instead.  The index is ~140KB and identical on every request,
      // so it wants to be a cached prompt prefix.
      //
      // A standalone CSP's interior nodes are left out, since the leaves under them already spell out the
      // path.  Names are deduplicated because a CSP published in both scopes (WiFi) would otherwise list
      // each one twice.
      //
      // A third-party template's group says which app it belongs to, since its area name is built from the
      // template's own category names (Cat_GoogleUpdate~Cat_Applications) and does not always say.

      // Build the lookup's index: one line per group, naming the group and every node in it.
      let nodeNameIndex = _.map(nodesByArea, (nodesInThisArea, areaName)=>{
        let leafNodes = _.filter(nodesInThisArea, (node)=>{ return node.csp === 'Policy' || node.format !== 'node'; });
        // Find the template this group's policies come from, if any, so the group can be labelled with its app.
        let admxTemplate = admxTemplatesById[_.first(nodesInThisArea).admxTemplate];
        return `${areaName}${admxTemplate ? ` (${admxTemplate.displayName} template)` : ''}: ${_.uniq(_.pluck(leafNodes, 'name')).sort().join(' ')}`;
      }).join('\n');

      let thirdPartyTemplateGuidance = '';
      // Only explain third-party template groups to the lookup when the request names one of their apps.
      if(admxTemplateIdsTheRequestNames.length > 0) {
        thirdPartyTemplateGuidance = `
Groups marked "(<App> template)" hold a third-party application's own policies, from the ADMX template
that application publishes, rather than anything in Windows.  When the request is about that application's
behavior, name its groups rather than a Windows area that sounds similar.  Groups with _recommended in
their name set a default the user can change; name them only when the request asks for a default or a
recommendation rather than an enforced setting.
`;
      }

      // The small model on purpose: this is a lookup rather than a judgement.
      let picked = await sails.helpers.ai.prompt.with({
        systemPrompt:
`Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not
include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

Below is every Windows CSP node a profile can set, in groups.  Policy CSP nodes are grouped by the area
they belong to (DeviceLock, Experience, ...); every other CSP is a single group named "<Name> CSP" (WiFi
CSP, Firewall CSP, BitLocker CSP, ...).  An IT admin has asked for a configuration profile, and another
model is about to write it -- but it can only be shown a few groups' worth of nodes, so your job is to say
which groups those should be.  Answer with group names, each called an area below.

Find the nodes that would actually satisfy the request and name the areas holding them.  Read the node
lists to do it: an area's name frequently does not follow from what its nodes do, so an area that sounds
right often is not the one, and the area that is right often sounds unrelated.

Name three to five areas, not one.  This is a shortlist, not an answer -- the model that writes the
profile picks the node out of what you send, so a second and third candidate cost it nothing, while
naming only the area you thought of first is how the right one gets left out.  After the area you are
most confident in, add the ones holding any other node that could plausibly do what was asked, including
ones you half-rejected.  Every area you name must appear verbatim below; do not invent one.

Some of the most common requests belong to a standalone CSP rather than the Policy CSP: provisioning a
Wi-Fi network is the WiFi CSP, a VPN is the VPNv2 CSP, installing a certificate is ClientCertificateInstall
CSP or RootCATrustedCertificates CSP, and turning the firewall on per network profile is the Firewall CSP.
The Policy CSP has adjacent-sounding areas -- Wifi holds AllowWiFi and AllowWiFiDirect -- that allow or
block a feature rather than configure it.  For a request to set something up, name the standalone CSP
first; name the Policy area too only if the request also asks to allow or block the feature.

Return an empty array, naming nothing, only when no group below could satisfy the request.  An empty
array is a real answer here, not a failure.
${thirdPartyTemplateGuidance}
${nodeNameIndex}

Respond in JSON with this data shape:
{
  "areas": ["TODO"]
}`,
        prompt: `Here are the instructions from an IT admin:
\`\`\`
${naturalLanguageInstructions}
\`\`\``,
        baseModel: 'claude-haiku-5-5',
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
        // Loop through the groups the lookup named, and keep each one that really exists.
        for (let candidate of picked.areas) {
          // Cut off the "(<App> template)" label the lookup sometimes copies from the index, leaving just the group name.
          let candidateName = String(candidate).split(' (')[0].trim();
          let matched = _.find(realAreaNames, (areaName)=>{ return areaName.toLowerCase() === candidateName.toLowerCase(); });
          if(matched && !_.contains(windowsCspAreasProvided, matched)) {
            windowsCspAreasProvided.push(matched);
          }
        }
      }

      // A keyword search over every node's name and description, to catch what the lookup misses.  The
      // lookup names about two areas however many it is asked for, and the ones it misses are those whose
      // names sound unrelated to the request even though their descriptions use its words: "security
      // questions for password reset" is ADMX_CredUI, "text messages backed up to the cloud" is Messaging,
      // "enhanced anti-spoofing for Windows Hello face" is PassportForWork CSP.  So up to two of the
      // best-scoring areas the lookup did not name are added, when they score close to the best match.
      // Weighted by rarity, so a word that appears in hundreds of nodes ("security") counts for little.
      const WORDS_THAT_SAY_NOTHING = ['the', 'and', 'for', 'from', 'with', 'without', 'this', 'that', 'these', 'those', 'into', 'over', 'under', 'when', 'where', 'which', 'while', 'what', 'how', 'any', 'all', 'can', 'cannot', 'not', 'only', 'also', 'than', 'then', 'them', 'they', 'their', 'there', 'here', 'has', 'have', 'had', 'was', 'were', 'been', 'being', 'are', 'its', 'let', 'allow', 'enable', 'disable', 'turn', 'off', 'set', 'setting', 'policy', 'configure', 'device', 'user', 'windows', 'microsoft', 'value', 'specify', 'whether', 'determine', 'control', 'option', 'feature', 'stop', 'people', 'machine', 'sure'];
      let wordsOf = (text)=>{
        return _.uniq(_.map(_.filter(String(text || '').replace(/([a-z])([A-Z])/g, '$1 $2').replace(/_/g, ' ').toLowerCase().split(/[^a-z0-9]+/), (word)=>{
          return word.length > 2;
        }), (word)=>{ return word.replace(/(ing|ed|es|s)$/, ''); })).filter((word)=>{ return !_.contains(WORDS_THAT_SAY_NOTHING, word); });
      };
      let wordsByNode = _.map(nodesAProfileCanSet, (node)=>{ return {node, words: wordsOf(`${node.name} ${node.description || ''}`)}; });
      let nodeCountByWord = _.countBy(_.flatten(_.pluck(wordsByNode, 'words')));
      let requestWords = wordsOf(naturalLanguageInstructions);
      let bestScoreByArea = {};
      for (let {node, words} of wordsByNode) {
        let score = _.sum(_.map(_.intersection(requestWords, words), (word)=>{ return Math.log(wordsByNode.length / (1 + nodeCountByWord[word])); }));
        let areaName = node.csp === 'Policy' ? node.area : `${node.csp} CSP`;
        bestScoreByArea[areaName] = Math.max(bestScoreByArea[areaName] || 0, score);
      }
      let areasBySearch = _.sortBy(Object.keys(bestScoreByArea), (areaName)=>{ return -bestScoreByArea[areaName]; });
      let topScore = bestScoreByArea[_.first(areasBySearch)] || 0;
      let bestScoreTheLookupFound = _.max(_.map(windowsCspAreasProvided, (areaName)=>{ return bestScoreByArea[areaName] || 0; }).concat([0]));
      // Added only on a strong match that the lookup's own areas lack.  Every miss this fixes scores 18 or
      // more; a vague request ("Require a password to unlock the device", 9.7) matches nothing in particular,
      // and there the added areas were lookalikes that displaced the right node.  When the lookup already
      // named an area holding an equally good match (speech and typing personalization, in Privacy), the
      // additions -- TextInput, Speech -- were likewise lookalikes and won.
      if(topScore >= 12 && bestScoreTheLookupFound < 0.9 * topScore) {
        let areasAddedBySearch = _.filter(areasBySearch, (areaName)=>{ return !_.contains(windowsCspAreasProvided, areaName) && bestScoreByArea[areaName] >= 0.6 * topScore; }).slice(0, 2);
        windowsCspAreasProvided = windowsCspAreasProvided.concat(areasAddedBySearch);
      }

      // A request that names an app is about that app, so when neither the lookup nor the search above
      // provided any of its template's groups, the template's best match is added rather than leaving the
      // model to recall a third-party path -- which is exactly what it gets wrong.
      // Loop through the templates the request names.
      for (let templateId of admxTemplateIdsTheRequestNames) {
        // Find every group of this template's policies.
        let areasOfThisTemplate = _.uniq(_.pluck(_.filter(nodesAProfileCanSet, {admxTemplate: templateId}), 'area'));
        // Skip a template the lookup or the keyword search already provided a group for.
        if(_.intersection(areasOfThisTemplate, windowsCspAreasProvided).length > 0) {
          continue;
        }
        // Find this template's best-scoring group in the keyword search.
        let bestAreaOfThisTemplate = _.find(areasBySearch, (areaName)=>{ return _.contains(areasOfThisTemplate, areaName) && bestScoreByArea[areaName] > 0; });
        // Add that group to the ones provided to the model.
        if(bestAreaOfThisTemplate) {
          windowsCspAreasProvided.push(bestAreaOfThisTemplate);
        }
      }

      // Record which templates' policies the model is being shown, for the caller.
      admxTemplatesProvided = _.uniq(_.compact(_.map(windowsCspAreasProvided, (areaName)=>{ return _.first(nodesByArea[areaName]).admxTemplate; })));

      if(windowsCspAreasProvided.length > 0) {

        // Rendered in area order, off a copy: windowsCspAreasProvided goes back to the caller in the order
        // the model named the areas, and sorting it in place would throw that away.
        //
        // Value descriptions are the reason this reference exists -- a node like DevicePasswordEnabled takes
        // 0 to mean enabled, and no amount of prose in the prompt stops a model inferring the opposite from
        // the name -- so they are kept, but trimmed to their first clause.  The full sentence is
        // documentation, not a constraint.
        let schemaLines = [];

        // An ingested template's policy takes a fragment of <data> elements rather than a single value, so
        // its line shows that fragment, with a {placeholder} for each value.  Listing the elements as
        // id=kind instead left the model writing the bare value as <Data>, the way every Policy CSP node
        // above it takes one.  Each placeholder carries the element's label, the text beside it in the Group
        // Policy editor, which is often the only thing saying what an element is for.
        // Create a helper function that renders a template policy's elements as the fragment that enables it, e.g., <enabled/><data id="HomepageLocation" value="{text: Home page URL}"/>.
        let renderAdmxFragment = (elements)=>{
          return '<enabled/>' + _.map(elements, (element)=>{
            let shape;
            // Show an enum's choices as value=label pairs, e.g., {5=Open New Tab Page|1=Restore the last session}.
            if(element.type === 'enum') {
              shape = _.map(element.items, (item)=>{ return `${item.value}=${_.trunc(String(item.label || '').replace(/\s+/g, ' '), {length: 40})}`; }).join('|');
            } else if(element.type === 'decimal' || element.type === 'longDecimal') {
              // Show a number element's allowed range when it has one, e.g., {number 0-100}.
              shape = (element.min !== undefined || element.max !== undefined) ? `number ${element.min !== undefined ? element.min : ''}-${element.max !== undefined ? element.max : ''}` : 'number';
            } else if(element.type === 'list') {
              // Show whether a list takes values alone or name,value pairs.
              shape = element.explicitValue ? 'list of name,value' : 'list';
            } else {
              // Show any other element by its type, e.g., {text} or {boolean}.
              shape = element.type;
            }
            let label = element.label ? `: ${_.trunc(element.label, {length: 60})}` : '';
            // Return the element as a <data> element with a {placeholder} value, followed by * when the element is required.
            return `<data id="${element.id}" value="{${shape}${label}}"/>${element.required ? '*' : ''}`;
          }).join('');
        };

        // Loop through the groups provided to the model, in alphabetical order, and write each one's heading and one line per node into the schema the model is given.
        for (let areaName of _.clone(windowsCspAreasProvided).sort()) {
          // Six names are both a Policy area and a standalone CSP (Accounts, BitLocker, CloudDesktop, Defender,
          // Update, WiFi).  Given the bare area name, the model built the standalone CSP's path for a Policy node
          // (./Device/Vendor/MSFT/Wifi/AllowAutoConnectToWiFiSenseHotspots), so those areas spell theirs out.
          let sharesItsNameWithACsp = !_.endsWith(areaName, ' CSP') && _.any(Object.keys(nodesByArea), (otherName)=>{ return otherName.toLowerCase() === `${areaName} CSP`.toLowerCase(); });
          // Find the template this group's policies come from, if any.
          let admxTemplateOfThisArea = admxTemplatesById[_.first(nodesByArea[areaName]).admxTemplate];
          // Head a template's group with its app's name, e.g., "GoogleChrome~Policy~googlechrome~Startup  (Google Chrome template)".
          if(admxTemplateOfThisArea) {
            schemaLines.push(`${areaName}  (${admxTemplateOfThisArea.displayName} template)`);
          } else {
            // Head any other group with its name, spelling out the Policy CSP path when the name is shared with a standalone CSP.
            schemaLines.push(sharesItsNameWithACsp ? `${areaName}  (Policy CSP area: ./Device/Vendor/MSFT/Policy/Config/${areaName}/<NodeName>)` : areaName);
          }

          for (let node of _.sortBy(nodesByArea[areaName], (node)=>{ return node.csp === 'Policy' ? node.name : node.locUri; })) {
            // A standalone CSP's paths follow no single pattern, so its nodes are listed by full LocURI,
            // which also carries the scope that @User marks on a Policy node.
            let parts = [node.csp === 'Policy' ? node.name : node.locUri, node.format];
            if(node.csp === 'Policy' && !_.contains(node.scopes, 'Device')) {
              parts.push('@User');
            }
            if(node.deprecated) {
              parts.push('deprecated');
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
            } else if(node.allowedRange !== undefined) {
              parts.push(`range=${node.allowedRange}${node.defaultValue !== undefined ? ` d=${node.defaultValue}` : ''}`);
            } else if(node.defaultValue !== undefined) {
              parts.push(`d=${node.defaultValue}`);
            }
            if(node.dependsOn) {
              parts.push(`needs ${node.dependsOn.locUri.split('/').slice(-2).join('/')}${node.dependsOn.allowedValue !== undefined ? `=${node.dependsOn.allowedValue}` : ''}`);
            }
            // Add the fragment that enables a template's policy, since it takes <data> elements rather than a single value.
            if(node.admxElements) {
              parts.push(renderAdmxFragment(node.admxElements));
            }
            // What the node does, only where its values do not already say: NotifyMalicious and
            // NotifyPasswordReuse both read "0=Disabled 1=Enabled", and MaxDevicePasswordFailedAttempts has no
            // values at all, so nothing told the model that it wipes the device.  First sentence only, since
            // longer lines have pulled the model towards whichever node shares a word with the request.
            let valuesAreGeneric = _.every(node.allowedValues || [], (allowedValue)=>{
              return /^(disabled|enabled|allowed|not allowed|allow|block|blocked|off|on|true|false|disable|enable|not configured)\.?$/i.test(String(allowedValue.description || '').trim());
            });
            if(node.description && valuesAreGeneric) {
              // Microsoft's boilerplate opening goes first: it is a third of the sentence and identical across
              // lookalikes, so a length cap would otherwise cut exactly the words that tell them apart.
              let firstSentence = node.description.split(/(?<=[a-z0-9)]\.)\s+(?=[A-Z])/)[0]
              .replace(/^This (?:policy setting|policy|setting) (?:determines|specifies|controls|allows you to (?:specify|configure)|lets you (?:specify|configure)|configures|enables or disables|allows or disallows)(?: whether(?: or not)?)?\s*/i, '');
              parts.push(`-- ${firstSentence.length <= 140 ? firstSentence : firstSentence.slice(0, 140).replace(/\s+\S*$/, '') + '…'}`);
            }
            schemaLines.push('  ' + parts.join(' '));
          }
        }

        promptConfig.providedSchemaDescription =
`Provided context: the Windows CSP nodes this request looks like it needs, straight from Microsoft's
published reference.  Each group is either a Policy CSP area or a whole standalone CSP, named "<Name> CSP".

In a Policy CSP area, a node's LocURI is ./Device/Vendor/MSFT/Policy/Config/<Area>/<NodeName> -- never a
shortened form of that path, and never an area segment you inferred from a policy's name.  In a standalone
CSP, each line starts with the node's full LocURI instead, because those paths follow no single pattern:
Firewall's start ./Vendor/MSFT/Firewall/, WiFi's ./Device/Vendor/MSFT/WiFi/ or ./User/Vendor/MSFT/WiFi/.
Copy them exactly.  A {Braced} segment is a dynamic node: replace it, braces included, with the instance
name the request calls for (an SSID, a VPN profile name, a rule ID), and create the instance with Add.

Format is a group, then one node per line:
  <NodeName or LocURI> <format> [flags] [values, range or default] [dependency] [-- what it does]
\`-- …\` is Microsoft's description of what the node does, given where its values alone do not say; when
two nodes' names both fit the request, it decides between them.  \`*\` marks a value as that node's default.  \`@User\` marks a Policy node that exists only under ./User/ --
writing it under ./Device/ deploys cleanly and enforces nothing.  \`atomic\` marks a node Microsoft
documents as requiring an <Atomic> wrapper.  \`deprecated\` marks a node Microsoft has retired; prefer
another node that does the same thing.  \`range=[a-b]\` is the node's allowed numeric range.  \`needs X=Y\`
is a node the setting depends on, which belongs in the profile alongside it.  The format on each line is
the node's declared DFFormat (\`node\` is an interior node that holds children, not a value): emit it
verbatim and make <Data> legal for it, rather than reasoning about the value's type from its name.

This is authoritative for the groups below and settles their node names, paths, formats and values.  It
is not the whole CSP reference, and what is missing decides between two different answers.  If no node here
enforces what was asked, return the "couldNotGenerateProfile" shape and name what you were looking for,
rather than recalling a node from a group you were not given.  But if a node here does enforce it and the
request merely describes the effect in words the node does not use -- asking to wipe after failed
passcode attempts, where the node sets the failed-attempt threshold that produces that outcome -- then
that is the node, so use it and put what it actually does, and anything the admin should know about the
gap, in "valueMeaning" and "caveats".  Abstaining is for a setting you cannot find, not for one whose
published name is less specific than the request.

Every group that exists, with its node count, is listed after the detail.  Use it only to tell whether
a setting you cannot find lives somewhere that was not provided -- the names alone do not tell you what
is in a group, and a group's name is often not what you would guess.`;

        // Written out here rather than left to the ADMX-backed rule in the CSP rules, because that rule is
        // about Windows' own ADMX-backed nodes and says nothing about how each element's value is encoded --
        // which is where a third-party policy goes wrong, since every element type encodes differently.
        if(admxTemplatesProvided.length > 0) {
          promptConfig.providedSchemaDescription += `

A group marked "(<App> template)" is a third-party application's policies, from the ADMX template it
publishes, rather than part of Windows.  Its LocURI follows the Policy CSP pattern with the group name as
the area, ~ characters included: ./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation,
or under ./User/ for one marked @User.  Each one is DFFormat chr, is set with <Replace>, and never takes a
bare value: its <Data> is always a CDATA-wrapped policy fragment.  Its line shows the fragment that enables
it -- copy that into <Data><![CDATA[...]]></Data>, keep a <data> element for each value the request gives
and drop the rest, and replace each {placeholder}, braces included, with the value.  A <data> element
followed by * must be kept whenever the policy is enabled.  To turn a policy off, the fragment is <disabled/>
alone.

Encode each value by the kind its placeholder names: text as given; number as digits within its range;
boolean as true or false; a choice (a=label|b=label) as the value before the = sign -- never the label,
and never its position in the list; multiText as its strings joined by &#xF000;; a list as name/value pairs
joined by &#xF000; -- for a plain list, name the entries 1, 2, 3 (1&#xF000;first&#xF000;2&#xF000;second);
for a list of name,value, use the names the request gives.  The fragment is parsed as XML, so write & < > "
inside a value as &amp; &lt; &gt; &quot;.

These nodes exist only once the template is installed on the device.  The command that installs it is added
to the profile automatically after you return it.`;
        }

        promptConfig.providedSchema = `${schemaLines.join('\n')}\n\nAll groups: ${_.map(Object.keys(nodesByArea).sort(), (areaName)=>{ return `${areaName}(${nodesByArea[areaName].length})`; }).join(' ')}`;

        // Ahead of the existing rules on purpose.  Several of those tell the model how to decide a format
        // or a polarity from memory -- sound advice when nothing was provided, and a licence to override
        // the provided data if it is left to rank itself against them.
        promptConfig.rules = [
          'Every node name, LocURI, format and allowed value in the provided list is authoritative.  Where the list and your own recollection differ, the list is right and you are wrong -- copy the path, the format and the value from it character for character.  The rules below about choosing a format or working out what a value means apply only to a setting the list does not cover.',
        ].concat(admxTemplatesProvided.length > 0 ? [
          // The template is ~250-450KB of XML, far past what a response can hold, so the model cannot write
          // the install command -- and what it writes in trying is a truncated or invented template.
          'Never write a ConfigOperations/ADMXInstall item, and never reproduce any part of a template.  The install command for each third-party template a profile uses is added after you return it.',
        ] : []).concat(promptConfig.rules);
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

    // Supplied for the same reason as the UUIDs: a WiFi SSID's <hex> is authoritative over its <name>, and the
    // model's own hex encoding of "CorpNet" came back with a byte inserted or changed in two runs of three.
    // Only quoted strings, since that is how an SSID arrives in the instructions, and double quotes only, so an
    // apostrophe does not open one.
    let hexEncodingsToUse = '';
    if(profileType === 'csp') {
      let quotedStrings = _.uniq(_.map(naturalLanguageInstructions.match(/["“”][^"“”]+["“”]/g) || [], (quoted)=>{ return quoted.slice(1, -1); }));
      if(quotedStrings.length > 0) {
        hexEncodingsToUse = `
    Uppercase hex encodings of the quoted strings in the instructions, for <hex> inside a WiFi <SSID>:
    ${quotedStrings.map((quoted)=>`- ${JSON.stringify(quoted)}: ${Buffer.from(quoted, 'utf8').toString('hex').toUpperCase()}`).join('\n    ')}
`;
      }
    }

    let userPrompt = `Given these instructions from an IT admin, generate a ${promptConfig.description}.
${uuidsToUse}${hexEncodingsToUse}
    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``;

    // promptConfig comes back because the action's settings-preview call needs description; suppliedPayloadUuids so a caller can check what the model was given.
    // windowsCspAreasProvided so a caller can tell a profile that was written from the wrong areas from one
    // written from the right areas badly -- the two look identical in the generated profile, and the first
    // is a lookup problem while the second is a prompt problem.
    // applePayloadTypesProvided for the same reason: empty means the lookup was off, failed, or found
    // nothing, and the full schema went in.
    // admxTemplatesProvided is every third-party template whose policies the model was shown, which is a
    // superset of the ones the profile ends up using.
    return { systemPrompt, userPrompt, promptConfig, suppliedPayloadUuids, windowsCspAreasProvided, applePayloadTypesProvided, admxTemplatesProvided };

  }


};

