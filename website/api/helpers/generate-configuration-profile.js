module.exports = {


  friendlyName: 'Generate configuration profile',


  description: 'Generate a configuration profile from an IT admin\'s instructions.',


  extendedDescription: 'Builds the prompts, generates with the model this profile type is generated with in production, checks that what came back is a profile of the requested type, and adds the ADMX install commands a Windows profile needs.',


  inputs: {

    profileType: {
      type: 'string',
      required: true,
      isIn: ['mobileconfig', 'ddm', 'csp', 'android'],
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

  },


  exits: {

    success: {
      outputFriendlyName: 'Generation result',
      outputDescription: '{profileType, couldNotGenerateProfile, reasonWhyAProfileCouldNotBeGenerated?, contentDidNotMatchProfileType, generatedProfile?: {profile, profileFilename, deliveryNotes, items, additionalProfiles}, rawResult, generatedWith: {baseModel, effort, disableThinking}, windowsCspAreasProvided, applePayloadTypesProvided, admxTemplatesProvided, timings: {configurationMs, generationMs}}',
    },

  },


  fn: async function ({profileType, naturalLanguageInstructions, baseModel, effort, disableThinking, useLighterResponseShape, useApplePayloadTypeLookup}) {

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

    let startedAt = Date.now();
    let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
      profileType,
      naturalLanguageInstructions,
      useLighterResponseShape,
      useApplePayloadTypeLookup,
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
    // budget where a second would not.
    let rawResult = await generate()
    .catch(async (err)=>{
      if(err.statusCode !== 429 && !(err.statusCode >= 500)) {
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
      // The configuration step's time is its lookups' time, so this splits a slow generation into a slow
      // lookup or a slow generation.
      timings: { configurationMs: configuredAt - startedAt, generationMs: generatedAt - configuredAt },
    };

    // A response missing any of these is not a usable profile, whatever else it holds.
    if(!_.isObject(rawResult) ||
        rawResult.couldNotGenerateProfile ||
        !rawResult.configurationProfile ||
        !rawResult.profileFilename ||
        !rawResult.settingsEnforced) {
      result.couldNotGenerateProfile = true;
      result.reasonWhyAProfileCouldNotBeGenerated = _.isObject(rawResult) && rawResult.reasonWhyAProfileCouldNotBeGenerated ? rawResult.reasonWhyAProfileCouldNotBeGenerated : undefined;
      return result;
    }

    let profile = withDoubleEscapedLineBreaksRepaired(rawResult.configurationProfile, profileType);

    // The type is promised to whoever asked for it, so content of another type is a failed generation rather
    // than a profile: a .mobileconfig handed back as a declaration is rejected by an MDM, or worse, accepted.
    if(!doesContentMatchProfileType(profile, profileType)) {
      sails.log.warn(`A generated ${profileType} profile did not look like one, so it was treated as a failed generation.  Instructions: ${JSON.stringify(naturalLanguageInstructions)}`);
      result.couldNotGenerateProfile = true;
      result.contentDidNotMatchProfileType = true;
      return result;
    }

    result.generatedProfile = {
      profile,
      profileFilename: rawResult.profileFilename,
      deliveryNotes: rawResult.deliveryNotes,
      items: rawResult.settingsEnforced,
      additionalProfiles: [],
    };

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
    case 'android': return _.isObject(parsedAsJson) && !_.isArray(parsedAsJson);
    case 'csp': return !isAPropertyList && parsedAsJson === undefined;
    default: return false;
  }
}
