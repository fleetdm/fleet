module.exports = {


  friendlyName: 'Generate configuration profile',


  description: 'Generate a configuration profile from an IT admin\'s instructions.',


  extendedDescription: 'Builds the prompts, generates with the model this profile type is generated with in production, checks that what came back is a profile of the requested type, and adds the ADMX install commands a Windows profile needs.',


  inputs: {

    profileType: {
      type: 'string',
      isIn: ['mobileconfig', 'ddm', 'csp', 'android'],
      description: 'The kind of profile to generate.  Pass this or a platform, not both.',
    },

    platform: {
      type: 'string',
      isIn: ['apple', 'windows'],
      description: 'The platform to generate a profile for, leaving the kind of profile to be decided.  Pass this or a profileType, not both.',
      extendedDescription: 'A Windows profile is always a CSP profile.  An Apple one is a declaration or a .mobileconfig, whichever Apple publishes the settings in.',
    },

    forFleet: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether Fleet is asking, rather than someone using the profile generator page.',
    },

    naturalLanguageInstructions: {
      type: 'string',
      required: true,
      description: 'What the IT admin asked for, in their own words.'
    },

    baseModel: {
      type: 'string',
      description: 'Generate with this model instead of the one this profile type is generated with in production.',
      extendedDescription: 'For measuring other models against the profile generator\'s test cases.',
    },

    effort: {
      type: 'string',
      description: 'Generate at this effort level instead of the one this profile type is generated with in production.',
    },

    disableThinking: {
      type: 'boolean',
      description: 'true to generate with thinking off, false to generate with it on, instead of the way this profile type is generated in production.',
    },

    useLighterResponseShape: {
      type: 'boolean',
      defaultsTo: true,
      description: 'Whether or not to request less information back with the generated profile.',
    },

    useApplePayloadTypeLookup: {
      type: 'boolean',
      defaultsTo: true,
      description: 'Whether or not to narrow the .mobileconfig schema to the payload types this request needs before generating.',
    },

    writeProfilesFromSettings: {
      type: 'boolean',
      defaultsTo: true,
      description: 'Whether to have the model return a .mobileconfig\'s payloads, or a Windows profile\'s commands, and their values, and write the document from them, rather than have it write the document.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Generation result',
      outputDescription: '{profileType, couldNotGenerateProfile, reasonWhyAProfileCouldNotBeGenerated?, contentDidNotMatchProfileType, generatedProfile?: {profile, profileFilename, deliveryNotes, items, additionalProfiles}, rawResult, generatedWith: {baseModel, effort, disableThinking}, windowsCspAreasProvided, applePayloadTypesProvided, admxTemplatesProvided, route?, timings: {routingMs, configurationMs, generationMs}}',
    },

  },


  fn: async function ({profileType, platform, forFleet, naturalLanguageInstructions, baseModel, effort, disableThinking, useLighterResponseShape, useApplePayloadTypeLookup, writeProfilesFromSettings}) {

    if(!profileType === !platform) {
      throw new Error('Pass either a profileType or a platform to generate a configuration profile for, not both.');
    }

    let startedAt = Date.now();
    let route;
    if(platform === 'windows') {
      profileType = 'csp';
    } else if(platform === 'apple') {
      route = await sails.helpers.routeAppleConfigurationProfileRequest.with({ naturalLanguageInstructions, forFleet });
      if(route.couldNotGenerateProfile) {
        return {
          profileType: undefined,
          couldNotGenerateProfile: true,
          reasonWhyAProfileCouldNotBeGenerated: withoutLinks(route.reasonWhyAProfileCouldNotBeGenerated),
          contentDidNotMatchProfileType: false,
          route,
          timings: { routingMs: Date.now() - startedAt, configurationMs: 0, generationMs: 0 },
        };
      }
      profileType = route.profileType;
    }
    let routedAt = Date.now();

    // Each profile type is generated with the cheapest configuration measured to clear the test suite's 95% bar
    // and its latency budget.  Sonnet 5.5 at low effort was the only model to clear both for every type when
    // models were compared.  Haiku 5.5 with thinking off clears both for Android, at a tenth of the price and
    // about half the time, and falls well short on every other type.  Turning Sonnet's thinking off roughly
    // halves the slowest Windows and .mobileconfig generations without losing accuracy; DDM declarations are
    // short enough that thinking does not make them slow.
    const GENERATION_SETTINGS_BY_PROFILE_TYPE = {
      mobileconfig: { baseModel: 'claude-sonnet-5-5', effort: 'low', disableThinking: true },
      ddm: { baseModel: 'claude-sonnet-5-5', effort: 'low', disableThinking: false },
      csp: { baseModel: 'claude-sonnet-5-5', effort: 'low', disableThinking: true },
      android: { baseModel: 'claude-haiku-5-5', effort: 'low', disableThinking: true },
    };

    let productionSettings = GENERATION_SETTINGS_BY_PROFILE_TYPE[profileType];
    let generatedWith = {
      baseModel: baseModel || productionSettings.baseModel,
      effort: effort || productionSettings.effort,
      disableThinking: disableThinking !== undefined ? disableThinking : productionSettings.disableThinking,
    };

    let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
      profileType,
      naturalLanguageInstructions,
      useLighterResponseShape,
      useApplePayloadTypeLookup,
      // The router already worked out which payload types a .mobileconfig needs, so the lookup would only
      // ask the same question again.
      applePayloadTypes: useApplePayloadTypeLookup && route && route.profileType === 'mobileconfig' ? route.applePayloadTypes : undefined,
      writeProfilesFromSettings,
      forFleet,
    });
    let configuredAt = Date.now();

    // A refusal is the model declining to answer at all, which the admin sees the same way as the model
    // declining to write this particular profile.
    let generate = async ()=>{
      return await sails.helpers.ai.prompt.with({
        systemPrompt: generatorConfiguration.systemPrompt,
        systemPromptCacheTtl: generatorConfiguration.systemPromptIsTheSameForEveryRequest ? '1h' : undefined,
        prompt: generatorConfiguration.userPrompt,
        baseModel: generatedWith.baseModel,
        effort: generatedWith.effort,
        disableThinking: generatedWith.disableThinking,
        expectJson: true,
      })
      .tolerate('refused', ()=>{
        return { couldNotGenerateProfile: true };
      });
    };
    // A rate limit or an overloaded API usually clears within a second, and one retry fits in the latency
    // budget where a second would not.  Only early on, though: one that starts after about 4 seconds, with a
    // generation taking up to 5, would finish after Fleet has stopped waiting.
    let rawResult = await generate()
    .catch(async (err)=>{
      let isWorthRetrying = err.statusCode === 429 || err.statusCode >= 500;
      if(!isWorthRetrying || Date.now() - startedAt > 4000) {
        throw err;
      }
      sails.log.warn(`Generating a ${profileType} profile failed with a ${err.statusCode} from the LLM API, so it is being retried once.`);
      await new Promise((resolve)=>{ setTimeout(resolve, 1000); });
      return await generate();
    });
    let generatedAt = Date.now();

    let result = {
      profileType,
      couldNotGenerateProfile: false,
      reasonWhyAProfileCouldNotBeGenerated: undefined,
      contentDidNotMatchProfileType: false,
      generatedProfile: undefined,
      rawResult,
      generatedWith,
      windowsCspAreasProvided: generatorConfiguration.windowsCspAreasProvided,
      applePayloadTypesProvided: generatorConfiguration.applePayloadTypesProvided,
      admxTemplatesProvided: generatorConfiguration.admxTemplatesProvided,
      route,
      // The configuration step's time is its lookups' time, so this splits a slow generation into slow
      // routing, a slow lookup, or a slow generation.
      timings: { routingMs: routedAt - startedAt, configurationMs: configuredAt - routedAt, generationMs: generatedAt - configuredAt },
    };

    // What the model returned, with the document written from its settings where it returned settings, so
    // rawResult stays exactly what the model said.
    let response = rawResult;
    if(writeProfilesFromSettings && (profileType === 'mobileconfig' || profileType === 'csp') && _.isObject(rawResult) && !rawResult.couldNotGenerateProfile) {
      response = _.extend({}, rawResult, profileType === 'mobileconfig' ? mobileconfigWrittenFromPayloadSettings(rawResult) : windowsProfileWrittenFromCommands(rawResult));
    }

    // A response missing any of these is not a usable profile, whatever else it holds.
    if(!_.isObject(response) ||
        response.couldNotGenerateProfile ||
        !response.configurationProfile ||
        !response.profileFilename ||
        !response.settingsEnforced) {
      result.couldNotGenerateProfile = true;
      result.reasonWhyAProfileCouldNotBeGenerated = _.isObject(response) && response.reasonWhyAProfileCouldNotBeGenerated ? withoutLinks(response.reasonWhyAProfileCouldNotBeGenerated) : undefined;
      return result;
    }

    let profile = withDoubleEscapedLineBreaksRepaired(response.configurationProfile, profileType);

    // The type is promised to whoever asked for it, so content of another type is a failed generation rather
    // than a profile: a .mobileconfig handed back as a declaration is rejected by an MDM, or worse, accepted.
    if(!doesContentMatchProfileType(profile, profileType)) {
      // The instructions themselves stay out of the log: admins put passwords and enrollment tokens in them.
      sails.log.warn(`A generated ${profileType} profile did not look like one, so it was treated as a failed generation.  The instructions were ${naturalLanguageInstructions.length} characters long.`);
      result.couldNotGenerateProfile = true;
      result.contentDidNotMatchProfileType = true;
      return result;
    }

    result.generatedProfile = {
      profile,
      profileFilename: response.profileFilename,
      deliveryNotes: response.deliveryNotes,
      items: response.settingsEnforced,
      additionalProfiles: [],
    };

    if(profileType === 'ddm' && forFleet && _.isString(result.generatedProfile.profile)) {
      let withUserScope = await sails.helpers.addUserPayloadScopeToDeclaration.with({
        profile: result.generatedProfile.profile,
        deliveryNotes: result.generatedProfile.deliveryNotes || '',
        platformsNamed: route ? route.platformsNamed : sails.helpers.getApplePlatformsNamed.with({ naturalLanguageInstructions }),
      });
      result.generatedProfile.profile = withUserScope.profile;
      result.generatedProfile.deliveryNotes = withUserScope.deliveryNotes;
    }

    if(profileType === 'mobileconfig' || profileType === 'ddm') {
      let wasWrittenFromPayloads = profileType === 'mobileconfig' && writeProfilesFromSettings && _.isArray(rawResult.payloads);
      let lifecycleNotes = sails.helpers.addLifecycleNotesToAppleProfile.with({
        profileType,
        profile: result.generatedProfile.profile,
        platformsNamed: route ? route.platformsNamed : sails.helpers.getApplePlatformsNamed.with({ naturalLanguageInstructions }),
        mobileconfigAlternative: route ? route.mobileconfigAlternative : [],
        payloads: wasWrittenFromPayloads ? rawResult.payloads : undefined,
        profileSettings: wasWrittenFromPayloads && _.isPlainObject(rawResult.profileSettings) ? rawResult.profileSettings : undefined,
      });
      result.generatedProfile.deliveryNotes = _.compact([result.generatedProfile.deliveryNotes].concat(lifecycleNotes)).join(' ');
    }

    if(profileType === 'csp') {
      let withAdmxInstalls = await sails.helpers.addAdmxInstallCommandsToWindowsProfile.with({ profile: result.generatedProfile.profile });
      result.generatedProfile.profile = withAdmxInstalls.profile;
      result.generatedProfile.additionalProfiles = withAdmxInstalls.admxInstallProfiles || [];
      result.generatedProfile.deliveryNotes = _.compact([result.generatedProfile.deliveryNotes].concat(withAdmxInstalls.deliveryNotes)).join(' ');
    }

    return result;

  }


};


/**
 * The configurationProfile and settingsEnforced of a .mobileconfig response, written from the payloads and
 * settings it holds instead.
 *
 * A response with an unusable payload or value is a failed generation, with the reason when there is one, even when
 * the model also wrote a document of its own: a profile missing part of what the model chose is not the profile that
 * was asked for.
 *
 * @param  {Dictionary} rawResult  {payloads, profileSettings, profileFilename, profileDisplayName, profileDescription}
 * @returns {Dictionary}  {configurationProfile, settingsEnforced}, {couldNotGenerateProfile, reasonWhyAProfileCouldNotBeGenerated}, or {}
 */
function mobileconfigWrittenFromPayloadSettings(rawResult) {
  let isUsablePayload = (payload)=>{
    return _.isPlainObject(payload) && _.isString(payload.payloadType) && _.isPlainObject(payload.settings);
  };
  if(!_.isArray(rawResult.payloads) || rawResult.payloads.length === 0 || !_.every(rawResult.payloads, isUsablePayload)) {
    return {};
  }
  let filenameStem = _.kebabCase(String(rawResult.profileFilename || '').replace(/\.mobileconfig$/i, '')) || 'profile';
  let built = sails.helpers.buildMobileconfigFromPayloadSettings.with({
    payloads: rawResult.payloads,
    profileSettings: _.isPlainObject(rawResult.profileSettings) ? rawResult.profileSettings : {},
    // A random last segment, so two profiles generated for the same request never share a PayloadIdentifier: an MDM
    // that keys profiles on it, as Fleet does, rejects the second.
    profileIdentifier: `com.example.${filenameStem}.${require('crypto').randomBytes(4).toString('hex')}`,
    profileDisplayName: _.isString(rawResult.profileDisplayName) && rawResult.profileDisplayName.trim() !== '' ? rawResult.profileDisplayName : filenameStem,
    profileDescription: _.isString(rawResult.profileDescription) ? rawResult.profileDescription : '',
  });
  if(built.unusableValues.length > 0) {
    return { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: built.unusableValues.join(' ') };
  }
  return { configurationProfile: built.profile, settingsEnforced: built.settingsEnforced };
}


/**
 * The configurationProfile and settingsEnforced of a Windows response, written from the commands it holds instead.
 *
 * Like a .mobileconfig, a response with an unusable command is a failed generation, with the reason it could not be
 * written.
 *
 * @param  {Dictionary} rawResult  {commands, atomic}
 * @returns {Dictionary}  {configurationProfile, settingsEnforced, deliveryNotes}, {couldNotGenerateProfile, reasonWhyAProfileCouldNotBeGenerated}, or {}
 */
function windowsProfileWrittenFromCommands(rawResult) {
  if(!_.isArray(rawResult.commands) || rawResult.commands.length === 0 || !_.every(rawResult.commands, _.isPlainObject)) {
    return {};
  }
  let built = sails.helpers.buildWindowsProfileFromCommands.with({ commands: rawResult.commands, atomic: rawResult.atomic === true });
  if(built.unusableCommands.length > 0) {
    return { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: built.unusableCommands.join(' ') };
  }
  return {
    configurationProfile: built.profile,
    settingsEnforced: built.settingsEnforced,
    deliveryNotes: _.compact([rawResult.deliveryNotes].concat(built.deliveryNotes)).join(' '),
  };
}


/**
 * A generated profile whose line breaks came back as the two characters \n rather than as line breaks.
 *
 * The prompt asks for line breaks escaped as \n inside the JSON response, and the model sometimes escapes them
 * twice, which leaves backslash-n sequences in the profile itself: a declaration that is no longer JSON, or a
 * plist with stray text between its elements.  Only a profile with no real line break in it is touched, and a
 * JSON profile only when it does not already parse, since a single-line value can legitimately hold a \n.  In
 * XML, only a \n beside markup is a line break: one inside a value, as in C:\Program Files\nodejs, is text.
 *
 * @param  {String} content
 * @param  {String} profileType
 * @returns {String}
 */
function withDoubleEscapedLineBreaksRepaired(content, profileType) {
  if(!_.isString(content) || /\n/.test(content) || !/\\n/.test(content)) {
    return content;
  }
  if(profileType === 'ddm' || profileType === 'android') {
    try {
      JSON.parse(content);
      return content;
    } catch (unusedErr) {
      // Not JSON as written, so see whether it is once its line breaks are real ones.
    }
    try {
      return JSON.stringify(JSON.parse(content.replace(/\\n/g, '\n')), null, 2);
    } catch (unusedErr) {
      return content;
    }
  }
  return content.replace(/>\\n/g, '>\n').replace(/\\n(?=[ \t]*<)/g, '\n');
}


/**
 * A decline reason without its links.
 *
 * Every page that shows a decline already links to the OS settings docs, so a reason links nowhere else.  Done here
 * rather than asked of the model: a prompt rule about how to word a decline made the Android model decline requests
 * it can meet.
 *
 * @param  {String} reason
 * @returns {String}
 */
function withoutLinks(reason) {
  if(!_.isString(reason)) {
    return reason;
  }
  return reason
  .replace(/\[([^\]]+)\]\((?:https?:\/\/|www\.)[^)]*\)/gi, '$1')
  .replace(/\s*\(?(?:https?:\/\/|www\.)[^\s)]+\)?/gi, '')
  .replace(/\s+([.,;:])/g, '$1')
  .replace(/\s{2,}/g, ' ')
  .trim();
}


/**
 * Whether generated content is the kind of document its profile type promises.
 *
 * Deliberately shallow: an MDM validates the document itself, and this only has to tell one profile type
 * from another.
 *
 * @param  {String} content
 * @param  {String} profileType
 * @returns {Boolean}
 */
function doesContentMatchProfileType(content, profileType) {
  let parsedAsJson;
  try {
    parsedAsJson = JSON.parse(content);
  } catch (unusedErr) {
    parsedAsJson = undefined;
  }
  let isAPropertyList = /<plist[\s>]/.test(content);
  switch (profileType) {
    case 'mobileconfig': return isAPropertyList;
    case 'ddm': return _.isObject(parsedAsJson) && !_.isArray(parsedAsJson) && _.isString(parsedAsJson.Type);
    // A declaration is a JSON object too.
    case 'android': return _.isPlainObject(parsedAsJson) && !(_.isString(parsedAsJson.Type) && _.startsWith(parsedAsJson.Type, 'com.apple.'));
    case 'csp': return /^\s*<(?:Add|Replace|Exec|Atomic|Delete)[\s>]/.test(content);
    default: return false;
  }
}
