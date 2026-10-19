module.exports = {


  friendlyName: 'Route Apple configuration profile request',


  description: 'Decide whether an IT admin\'s request for an Apple profile should be a declaration or a .mobileconfig, or cannot be one profile at all.',


  extendedDescription: `Sonnet 5.5 maps each setting the request asks for to the declaration key and the .mobileconfig key
that do it, from an index of both.  The format is then chosen from that mapping in code, from Apple's own
lifecycle data rather than the model's memory: a declaration when one declaration type covers everything, a
.mobileconfig otherwise, and a decline only when a setting a declaration could enforce would have to use a
.mobileconfig key Apple has removed.`,


  inputs: {

    naturalLanguageInstructions: {
      type: 'string',
      required: true,
    },

    forFleet: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether the request comes from Fleet, which does not accept every declaration type.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Route',
      outputDescription: '{profileType: "ddm"|"mobileconfig"|undefined, couldNotGenerateProfile, reasonWhyAProfileCouldNotBeGenerated?, applePayloadTypes?, mobileconfigAlternative, settings, platformsNamed, decidedBecause, routerMs}',
    },

  },


  fn: async function ({naturalLanguageInstructions, forFleet}) {

    let path = require('path');
    let declarationSchema = require(path.resolve(sails.config.appPath, 'profile-generator/schema/apple-ddm-declarations.json'));
    let payloadSchema = require(path.resolve(sails.config.appPath, 'profile-generator/schema/apple-payload-manifests.json'));

    // Declaration types Fleet will not deliver.  Requests from anywhere else can use any declaration type.
    const DECLARATION_TYPES_FLEET_DOES_NOT_ACCEPT = [];

    // Declaration types that install or manage something rather than configure a setting.  They are indexed
    // apart from the rest, since a request that maps onto one of them by accident ("install our profile"
    // onto legacy, "turn off the camera in Firefox" onto app.settings) gets a declaration that cannot do it.
    const DECLARATION_TYPES_THAT_ARE_NOT_SETTINGS = {
      'com.apple.configuration.legacy': 'installs a .mobileconfig file from a URL',
      'com.apple.configuration.legacy.interactive': 'installs a .mobileconfig file from a URL, with the user\'s involvement',
      'com.apple.configuration.package': 'installs a package',
      'com.apple.configuration.app.managed': 'installs and manages an app',
      'com.apple.configuration.app.settings': 'configures an app installed by app.managed',
      'com.apple.configuration.management.status-subscriptions': 'subscribes the server to status reports',
      'com.apple.configuration.management.test': 'tests declarative device management',
      'com.apple.configuration.watch.enrollment': 'enrolls an Apple Watch',
    };
    const PAYLOAD_TYPES_THAT_ARE_NOT_PAYLOADS = ['CommonPayloadKeys', 'TopLevel'];

    // Each declaration key on a line of its own, with the values it takes and the first sentence of what it
    // does.  Given names alone, the router mapped settings no declaration has onto unrelated keys that
    // existed (diagnostic reports onto disk management), and split "read but not write" into two settings
    // because it could not see that one value (ReadOnly) does both.  Only the declarations get this: which
    // declaration a setting maps to is what decides the format.
    //
    // A key that exists on fewer platforms than its declaration is marked with the ones it does, the way the
    // generator's schema marks it: without that, a request about Macs was mapped onto
    // Deferrals.CombinedPeriodInDays, which macOS does not have.
    let platformsAvailableOn = (supportedOS)=>{
      return _.filter(Object.keys(supportedOS || {}), (platformName)=>{ return supportedOS[platformName].introduced && supportedOS[platformName].introduced !== 'n/a'; });
    };
    let keyWithItsValues = (key, inheritedSupportedOS)=>{
      let supportedOS = key.supportedOS || inheritedSupportedOS;
      let platformNames = platformsAvailableOn(supportedOS);
      let onlyOn = key.supportedOS && platformNames.length > 0 && platformNames.length < platformsAvailableOn(inheritedSupportedOS).length ? `[only ${platformNames.join(', ')}]` : '';
      let values = key.allowedValues && key.allowedValues.length > 0 ? `(${key.allowedValues.join('|')})` : '';
      let subkeys = key.subkeys && key.subkeys.length > 0 && /dictionary/.test(String(key.type || '')) ? `{${_.map(key.subkeys, (subkey)=>{ return keyWithItsValues(subkey, supportedOS); }).join(', ')}}` : '';
      return `${key.key}${onlyOn}${values}${subkeys}`;
    };
    let firstSentence = (text)=>{
      let sentence = String(text || '').split(/(?<=\.)\s/)[0];
      return sentence.length > 140 ? `${sentence.slice(0, 140).replace(/\s+\S*$/, '')}…` : sentence;
    };
    let declarationIndexLine = (entry)=>{
      return `${entry.name} (${_.compact([entry.title].concat(entry.platforms || [])).join('; ')}): ${entry.description || ''}\n` +
        _.map(entry.keys, (key)=>{ return `    ${keyWithItsValues(key, entry.supportedOS)}  ${firstSentence(key.gloss || key.description)}`; }).join('\n');
    };
    let [notSettings, settings] = _.partition(declarationSchema.entries, (entry)=>{ return DECLARATION_TYPES_THAT_ARE_NOT_SETTINGS[entry.name] !== undefined; });
    let declarationIndex = _.map(settings, declarationIndexLine).join('\n');
    let notSettingsIndex = _.map(notSettings, (entry)=>{ return `${entry.name}: ${DECLARATION_TYPES_THAT_ARE_NOT_SETTINGS[entry.name]}.  Keys: ${_.pluck(entry.keys, 'key').join(' ')}`; }).join('\n');
    // The same index the .mobileconfig payload type lookup reads.
    let payloadIndex = _.map(_.reject(payloadSchema.entries, (entry)=>{ return _.contains(PAYLOAD_TYPES_THAT_ARE_NOT_PAYLOADS, entry.name); }), (entry)=>{
      return `${entry.name} (${_.compact([entry.title].concat(entry.platforms || [])).join('; ')}): ${entry.description || ''} Keys: ${_.pluck(entry.keys, 'key').join(' ')}`;
    }).join('\n');

    let startedAt = Date.now();
    // Sonnet rather than the small model the lookups use: Haiku routed 91-93% of the test cases correctly,
    // missing keys it had been shown in an index this long, where Sonnet routed 96%, for about 0.8 seconds more
    // (1.8s typically, 2.6-3.3s at p95).  This sits in front of the generation, inside the same 10-second
    // budget, so a reply slower than this is given up on and the request is generated the way an unrouted one
    // always was.
    const ROUTER_TIMEOUT_MS = 5000;
    let timeout;
    let mapping = await Promise.race([
      sails.helpers.ai.prompt.with({
        systemPrompt:
`Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not
include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

An Apple device setting can usually be managed in one or both of two formats: a declaration (declarative
device management) or a .mobileconfig profile.  Below are both references: every declaration type with
its keys, then every .mobileconfig payload type with its keys.  An IT admin has asked for a configuration
profile, and another model will write it in one format -- your job is to say where each setting the
request asks for lives in each format.

Split the request into the separate settings it asks to change, leaving out anything it asks to leave as
it is.  A request for a whole group of settings -- every, all -- is one entry for each declaration type the
group spans, each with one of that type's keys, rather than an entry for every key.  For each one, give:
- "declaration": the declaration key that does it, as "<declaration type>/<Key>", or
  "<declaration type>/<Parent>.<Key>" for a key listed inside braces.  null if no declaration key does it.
  A key named ANY stands for a name the admin supplies, such as an app or extension identifier: write ANY
  itself in the path, e.g. "<declaration type>/ManagedExtensions.ANY.State".  A key marked
  [only <platforms>] does nothing on any other platform: for a request about a platform it does not list,
  use a key that platform has.
- "payload": the .mobileconfig key that does it, as "<payload type>/<key>".  null if no payload key does it.
Read the key lists rather than going by a type's name: a type whose name matches the topic of a request is
often not the one holding the key.  Copy every name exactly as it appears below; never invent one.  Both
formats often have the setting, under different names -- look in both.  A key only counts when it does
what was asked: the software update declaration's keys are for OS updates, so App Store app updates are
not among them.

Also name one to four .mobileconfig payload types the profile would need, in "payloadTypes": a shortlist
for the model that writes a .mobileconfig, so add any payload type that could plausibly hold a setting.

A setting no key below covers -- a third-party application's preference, for one -- gets null for both.

Declaration types:
${declarationIndex}

Declaration types that install or manage something rather than configure a setting.  Name one only when
the request asks for exactly what it does -- a request to install a .mobileconfig file from a URL is
com.apple.configuration.legacy/ProfileURL:
${notSettingsIndex}

.mobileconfig payload types:
${payloadIndex}

Respond in JSON with this data shape:
{
  "settings": [
    {"setting": "TODO: a few words naming the setting", "declaration": "TODO or null", "payload": "TODO or null"}
  ],
  "payloadTypes": ["TODO"]
}`,
        systemPromptCacheTtl: '1h',
        prompt: `Here are the instructions from an IT admin:
\`\`\`
${naturalLanguageInstructions}
\`\`\``,
        baseModel: 'claude-sonnet-5-5',
        disableThinking: true,
      })
      .tolerate((err)=>{
        sails.log.warn(`When trying to route an Apple configuration profile request, an error occurred.  It will be generated as a .mobileconfig from the full schema.  Full error: ${require('util').inspect(err, {depth: 2})}`);
        return undefined;
      }),
      new Promise((resolve)=>{ timeout = setTimeout(()=>{ resolve('timedOut'); }, ROUTER_TIMEOUT_MS); }),
    ]);
    clearTimeout(timeout);
    let routerMs = Date.now() - startedAt;
    if(mapping === 'timedOut') {
      sails.log.warn(`Routing an Apple configuration profile request took over ${ROUTER_TIMEOUT_MS}ms.  It will be generated as a .mobileconfig from the full schema.`);
      mapping = undefined;
    } else if(mapping !== undefined) {
      mapping = lastJsonObjectIn(mapping);
    }

    let platformsNamed = sails.helpers.getApplePlatformsNamed.with({ naturalLanguageInstructions });

    // Validated against the schemas rather than trusted, the way the lookups' answers are: a path that does not
    // exist is dropped, and one written in the wrong case is rewritten in the schema's.
    let validatedPath = (profileType, typeAndKey)=>{
      if(!_.isString(typeAndKey) || !_.contains(typeAndKey, '/')) {
        return undefined;
      }
      let [typeName, keyPath] = [typeAndKey.slice(0, typeAndKey.indexOf('/')).trim(), typeAndKey.slice(typeAndKey.indexOf('/') + 1).trim()];
      if(!typeName || !keyPath) {
        return undefined;
      }
      let lifecycle = sails.helpers.getAppleSchemaLifecycle.with({ profileType, name: typeName, keyPath });
      if(!lifecycle.found) {
        return undefined;
      }
      let entries = profileType === 'ddm' ? declarationSchema.entries : payloadSchema.entries;
      let canonicalTypeName = _.find(entries, (entry)=>{ return entry.name.toLowerCase() === typeName.toLowerCase(); }).name;
      return { typeName: canonicalTypeName, keyPath: lifecycle.keyPath, supportedOS: lifecycle.supportedOS };
    };
    let isAvailableOn = (supportedOS, platformName)=>{
      let versions = (supportedOS || {})[platformName];
      return !!versions && !!versions.introduced && versions.introduced !== 'n/a';
    };
    let isAvailableOnEveryPlatformNamed = (supportedOS)=>{
      return platformsNamed.length === 0 ? _.any(Object.keys(supportedOS || {}), (platformName)=>{ return isAvailableOn(supportedOS, platformName); }) : _.all(platformsNamed, (platformName)=>{ return isAvailableOn(supportedOS, platformName); });
    };
    let isRemovedOnAPlatformThatMatters = (supportedOS)=>{
      let platformsThatMatter = platformsNamed.length > 0 ? platformsNamed : _.filter(Object.keys(supportedOS || {}), (platformName)=>{ return isAvailableOn(supportedOS, platformName); });
      return _.any(platformsThatMatter, (platformName)=>{ return !!((supportedOS || {})[platformName] || {}).removed; });
    };

    let settingsMapped = _.map(_.filter(mapping && _.isArray(mapping.settings) ? mapping.settings : [], _.isObject), (setting)=>{
      return {
        setting: String(setting.setting || ''),
        declaration: validatedPath('ddm', setting.declaration),
        payload: validatedPath('mobileconfig', setting.payload),
      };
    });
    let applePayloadTypes = _.uniq(_.compact(_.map(mapping && _.isArray(mapping.payloadTypes) ? mapping.payloadTypes : [], (payloadType)=>{
      let matched = _.find(payloadSchema.entries, (entry)=>{ return entry.name.toLowerCase() === String(payloadType).split(' (')[0].trim().toLowerCase(); });
      return matched && !_.contains(PAYLOAD_TYPES_THAT_ARE_NOT_PAYLOADS, matched.name) ? matched.name : undefined;
    })).concat(_.map(_.filter(settingsMapped, 'payload'), (setting)=>{ return setting.payload.typeName; })));
    let mobileconfigAlternative = _.map(_.filter(settingsMapped, 'payload'), (setting)=>{ return `${setting.payload.typeName}/${setting.payload.keyPath}`; });

    let route = { couldNotGenerateProfile: false, settings: settingsMapped, platformsNamed, mobileconfigAlternative, routerMs };

    // Nothing mapped: a third-party application's settings, or a request the router could not read.  That is
    // the .mobileconfig, from the full schema, that every such request got before routing existed.
    if(!_.any(settingsMapped, (setting)=>{ return setting.declaration || setting.payload; })) {
      return _.extend(route, { profileType: 'mobileconfig', applePayloadTypes: [], decidedBecause: 'nothing mapped to a known key' });
    }

    let isADeclarationTypeThatCanBeUsed = (declaration)=>{
      if(DECLARATION_TYPES_THAT_ARE_NOT_SETTINGS[declaration.typeName] !== undefined && !/(https?:\/\/|\.mobileconfig\b)/i.test(naturalLanguageInstructions)) {
        return false;
      }
      if(forFleet && _.contains(DECLARATION_TYPES_FLEET_DOES_NOT_ACCEPT, declaration.typeName)) {
        return false;
      }
      // Fleet resolves an asset reference to an asset already uploaded to it, and a generated profile has no
      // asset to point at, so a declaration that needs one cannot be delivered from a single file.
      if(/AssetReferences?$/.test(declaration.keyPath)) {
        return false;
      }
      return isAvailableOnEveryPlatformNamed(declaration.supportedOS);
    };
    let declarationTypeNames = _.uniq(_.map(_.filter(settingsMapped, 'declaration'), (setting)=>{ return setting.declaration.typeName; }));
    let everySettingHasAUsableDeclaration = _.all(settingsMapped, (setting)=>{ return setting.declaration && isADeclarationTypeThatCanBeUsed(setting.declaration); });
    if(everySettingHasAUsableDeclaration && declarationTypeNames.length === 1) {
      return _.extend(route, { profileType: 'ddm', decidedBecause: `one declaration type covers every setting: ${declarationTypeNames[0]}` });
    }

    // From here on the profile is a .mobileconfig.  A setting a declaration could enforce, whose only
    // .mobileconfig key Apple has removed, cannot go in it: that setting needs a declaration of its own.
    let settingsThatNeedTheirOwnDeclaration = _.filter(settingsMapped, (setting)=>{
      return setting.declaration && isADeclarationTypeThatCanBeUsed(setting.declaration) && setting.payload && isRemovedOnAPlatformThatMatters(setting.payload.supportedOS);
    });
    if(settingsThatNeedTheirOwnDeclaration.length > 0) {
      // Every other setting goes where it can: a declaration-only setting in its declaration, and anything with a
      // .mobileconfig key, or with no known key at all, in the .mobileconfig.
      let [settingsForADeclaration, settingsForTheMobileconfig] = _.partition(_.difference(settingsMapped, settingsThatNeedTheirOwnDeclaration), (setting)=>{
        return setting.declaration && !setting.payload;
      });
      let settingsByDeclarationTypeName = _.groupBy(settingsForADeclaration, (setting)=>{ return setting.declaration.typeName; });
      let numberOfProfiles = _.union(_.map(settingsThatNeedTheirOwnDeclaration, (setting)=>{ return setting.declaration.typeName; }), Object.keys(settingsByDeclarationTypeName)).length + (settingsForTheMobileconfig.length > 0 ? 1 : 0);
      let quoted = (settings)=>{ return listed(_.map(settings, (setting)=>{ return `"${setting.setting}"`; })); };
      const NUMBER_WORDS = ['zero', 'one', 'two', 'three', 'four', 'five', 'six'];
      return _.extend(route, {
        couldNotGenerateProfile: true,
        decidedBecause: 'a setting a declaration covers would need a removed .mobileconfig key',
        reasonWhyAProfileCouldNotBeGenerated: _.compact([
          `This request needs ${NUMBER_WORDS[numberOfProfiles] || numberOfProfiles} ${numberOfProfiles === 1 ? 'profile' : 'profiles'}.`,
          _.map(settingsThatNeedTheirOwnDeclaration, (setting)=>{ return `"${setting.setting}" is only enforced on current OS versions by the ${setting.declaration.typeName} declaration, since Apple removed the .mobileconfig key for it (${setting.payload.keyPath}).`; }).join('  '),
          _.map(Object.keys(settingsByDeclarationTypeName), (typeName)=>{ return `${quoted(settingsByDeclarationTypeName[typeName])} ${settingsByDeclarationTypeName[typeName].length === 1 ? 'needs' : 'need'} the ${typeName} declaration.`; }).join('  '),
          settingsForTheMobileconfig.length > 0 ? `${quoted(settingsForTheMobileconfig)} ${settingsForTheMobileconfig.length === 1 ? 'needs' : 'need'} a .mobileconfig.` : '',
          'Ask for each separately.',
        ]).join('  '),
      });
    }

    // The two always-provided payload types the lookup adds, for the reason it gives: Restrictions and Software
    // Update hold keys whose payload type's name sounds unrelated to the request.
    return _.extend(route, {
      profileType: 'mobileconfig',
      applePayloadTypes: _.union(applePayloadTypes, ['com.apple.applicationaccess', 'com.apple.SoftwareUpdate']),
      decidedBecause: declarationTypeNames.length > 1 ? 'more than one declaration type would be needed' : 'not every setting has a declaration that can be used',
    });

  }


};


/**
 * Names joined the way a sentence lists them: "a", "a and b", "a, b, and c".
 *
 * @param  {Array} names
 * @returns {String}
 */
function listed(names) {
  if(names.length <= 2) {
    return names.join(' and ');
  }
  return `${_.initial(names).join(', ')}, and ${_.last(names)}`;
}


/**
 * The last JSON object in a model's answer.
 *
 * With thinking off, the router sometimes answers, notices a mistake, and writes "Corrected output:" and a
 * second answer after the first.  The second is the one it means, and parsing the whole reply as JSON throws
 * both away.
 *
 * @param  {String} text
 * @returns {Dictionary|undefined}
 */
function lastJsonObjectIn(text) {
  let objects = [];
  let depth = 0;
  let start;
  let inString = false;
  for (let i = 0; i < text.length; i++) {
    let character = text[i];
    if(inString) {
      if(character === '\\') {
        i++;
      } else if(character === '"') {
        inString = false;
      }
      continue;
    }
    if(character === '"') {
      inString = true;
    } else if(character === '{') {
      if(depth === 0) {
        start = i;
      }
      depth++;
    } else if(character === '}' && depth > 0) {
      depth--;
      if(depth === 0) {
        objects.push(text.slice(start, i + 1));
      }
    }
  }
  for (let candidate of objects.reverse()) {
    try {
      return JSON.parse(candidate);
    } catch (unusedErr) {
      // Not JSON after all; try the one before it.
    }
  }
  return undefined;
}
