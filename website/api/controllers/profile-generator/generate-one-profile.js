module.exports = {


  friendlyName: 'Generate one profile',


  description: 'Generate a configuration profile from an IT admin\'s instructions, for a Fleet server.',


  extendedDescription: 'Fleet Premium is not limited.  A Fleet Free registration is limited per UTC month, counting each request once it is sent to the AI service, and giving it back if the AI service never answers.',


  inputs: {

    platform: {
      type: 'string',
      isIn: ['apple', 'windows'],
      required: true,
    },

    naturalLanguageInstructions: {
      type: 'string',
      required: true,
    },

    fleetLicenseKey: {
      type: 'string',
      description: 'The Fleet server\'s license key.  Without a valid Fleet Premium license, the request is authenticated by its registration secret instead.',
    },

  },


  exits: {

    success: {
      description: 'A configuration profile was generated for a Fleet server.',
    },

    invalidInput: {
      description: 'The instructions are too long.',
      responseType: 'badRequest',
    },

    couldNotGenerateProfile: {
      description: 'A configuration profile could not be generated using the provided instructions.',
      responseType: 'badRequest',
    },

    unauthorized: {
      description: 'Neither a valid Fleet Premium license key nor a known registration secret was sent.',
      responseType: 'unauthorized',
    },

    generationLimitReached: {
      description: 'This Fleet server has reached its limit on profile generation for the month.',
      statusCode: 429,
    },

  },


  fn: async function ({platform, naturalLanguageInstructions, fleetLicenseKey}) {

    // Counted the same way as on the configuration generator page's route.
    const MAX_INSTRUCTIONS_LENGTH = 2000;
    if(Array.from(naturalLanguageInstructions).length > MAX_INSTRUCTIONS_LENGTH) {
      throw {invalidInput: {error: 'invalidInput', reason: `Instructions must be ${MAX_INSTRUCTIONS_LENGTH} characters or fewer.`}};
    }

    let fleetServer = await sails.helpers.profileGenerator.authenticateFleetServer.with({ req: this.req, fleetLicenseKey })
    .intercept('couldNotAuthenticate', ()=>{
      return 'unauthorized';
    });

    let usage;
    if(fleetServer.registration) {
      usage = await sails.helpers.profileGenerator.countOneRequest.with({ registrationId: fleetServer.registration.id, requestType: 'profile' })
      .intercept('limitReached', (err)=>{
        return {generationLimitReached: {error: 'generationLimitReached', resetsAt: err.raw}};
      });
      sails.log.info(`Generating a configuration profile for Fleet Free registration ${fleetServer.registration.id} (${usage.profileGenerationsThisMonth} of ${sails.config.custom.profileGeneratorMonthlyProfileLimit} this month).`);
    } else {
      sails.log.info(`Generating a configuration profile for Fleet Premium license ${fleetServer.licenseSubject}.`);
    }

    let generationResult;
    try {
      generationResult = await sails.helpers.generateConfigurationProfile.with({
        platform,
        forFleet: true,
        naturalLanguageInstructions,
      });
    } catch(err) {
      sails.log.warn(`When trying to generate a configuration profile for a Fleet server, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      // Only a request the AI service never answered is given back.  An answer that could not be used still
      // cost a generation, and a caller could write instructions that produce one on purpose.
      if(usage && err.llmDidNotAnswer) {
        await sails.helpers.profileGenerator.giveBackOneRequest.with({ registrationId: fleetServer.registration.id, requestType: 'profile', usageMonth: usage.usageMonth });
      }
      throw err;
    }

    if(generationResult.contentDidNotMatchProfileType) {
      // A generation that went wrong rather than a decline, so the caller gets the same 500 as any other failure.
      throw new Error(`The generated profile did not look like a ${generationResult.profileType} profile.`);
    }
    if(generationResult.couldNotGenerateProfile) {
      throw {couldNotGenerateProfile: {
        error: 'couldNotGenerateProfile',
        reason: generationResult.reasonWhyAProfileCouldNotBeGenerated || 'These instructions could not be turned into a configuration profile.',
      }};
    }

    // The type first, since for a platform request it is the only place the caller learns what it got.
    return _.extend({profileType: generationResult.profileType}, generationResult.generatedProfile);

  }


};
