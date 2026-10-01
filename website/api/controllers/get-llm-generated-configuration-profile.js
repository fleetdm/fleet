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

    // Get the prompts for this profile, and the configuration for this profile type.
    let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
      profileType,
      naturalLanguageInstructions,
      useLighterResponseShape: true,
    });
    let promptConfig = generatorConfiguration.promptConfig;
    let systemPrompt = generatorConfiguration.systemPrompt;
    let configurationProfilePrompt = generatorConfiguration.userPrompt;


    // Start generating a configuration profile with haiku immediately, this result is only used if the triage prompt's result indicates that this profile will use
    let draftProfilePromise = (async ()=>{
      // console.time('haiku prompt')
      let draftPromise = await sails.helpers.ai.prompt.with({
        systemPrompt: systemPrompt,
        prompt: configurationProfilePrompt,
        baseModel: 'claude-haiku-4-5',
        expectJson: true,
      })
      .tolerate((err)=>{
        sails.log.warn(`When trying to generate a draft configuration profile with the smaller model, an error occurred. Falling back to the larger model. Full error: ${require('util').inspect(err, {depth: 2})}`);
        return undefined;
      });
      // console.timeEnd('haiku prompt')
      // console.log('haikuResult', draftPromise);
      return draftPromise;
    })();

    // Build a prompt that will be used to determine whether or not we need to use sonnet for this user's instreuctions.
    let triageSystemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

An IT admin has asked for a ${promptConfig.description}, and another model is already writing it.  You do not write the profile.  You answer one question about the request, quickly, plus name the settings the profile is likely to enforce so the admin has something to read while it is generated.

The question: does satisfying this request require a setting that is not ${promptConfig.firstPartySettingDescription}?

Answer true when a setting the request needs is defined by an application vendor -- Chrome, Firefox, Zoom, Slack, Microsoft Office -- and documented in that application's own preference manifest rather than in the platform vendor's published reference.  Answer false when every setting the request needs is ${promptConfig.firstPartySettingDescription}.  When you cannot tell which of the two a setting is, answer true: a wrong "false" writes the profile from a reference that does not describe the setting.

"anticipatedSettings" is a preview, replaced by the real settings the moment the profile arrives.  A best guess is useful there and a long list is not, so name the settings this request asks for and nothing else, and return an empty array when you cannot name them.

Respond in JSON with this data shape:
{
  "requiresAThirdPartyApplicationReference": true,
  // A short user-facing description of what the profile will do.
  "anticipatedDescription": "TODO",
  // The anticipated name of the configuration profile.
  "anticipatedName": "TODO",
  "anticipatedSettings": [
    {
      // The name (key) of the setting you expect the profile to enforce. e.g., LoginwindowText
      "name": "TODO",
      // The value you expect it to be set to.
      "value": "TODO"
    },
    {...}
  ]
}
`;
    // console.time('triage prompt');
    let triageResult = await sails.helpers.ai.prompt.with({
      systemPrompt: triageSystemPrompt,
      prompt: `Here are the instructions from an IT admin:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``,
      baseModel: 'claude-haiku-4-5',
      expectJson: true,
    })
    .tolerate((err)=>{
      sails.log.warn(`When trying to triage a user's configuration profile instructions, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      return undefined;
    });
    // console.timeEnd('triage prompt');
    // Send the anticipatedSettings, anticipatedName, and anticipatedDescription to the socket with a 'settingsPreview' event.
    let anticipatedSettings = _.filter(triageResult ? triageResult.anticipatedSettings || [] : [], (setting)=>{
      return _.isObject(setting) && setting.name;
    });
    let anticipatedName = triageResult ? triageResult.anticipatedName : '';
    let anticipatedDescription = triageResult ? triageResult.anticipatedDescription : '';
    if(this.req.isSocket && anticipatedSettings.length > 0) {
      sails.sockets.broadcast(roomId, 'settingsPreview', {settings: anticipatedSettings, description: anticipatedDescription, name: anticipatedName});
    }
    // console.log(`requiresAThirdPartyApplicationReference ${triageResult ? triageResult.requiresAThirdPartyApplicationReference : '(triage failed)'}`);


    let needsAReferenceTheSmallerModelIsLikelyToRecallWrong = !triageResult || !!triageResult.requiresAThirdPartyApplicationReference;
    let configurationProfileGenerationResult;
    // If the triage result indicates that this configuraiton profile does not require third party settings, wait for the configuration profile generated by haiku.
    if(!needsAReferenceTheSmallerModelIsLikelyToRecallWrong) {
      // console.log('Awaiting the speculative haiku draft');
      // console.time('waiting on haiku draft profile')
      configurationProfileGenerationResult = await draftProfilePromise;
      // console.timeEnd('waiting on haiku draft profile')
      // sails.log(configurationProfileGenerationResult);
    }

    // If the haiku draft failed, or the triage result said that this profile will use third-party settings,
    if(!configurationProfileGenerationResult ||
        configurationProfileGenerationResult.couldNotGenerateProfile ||
        !configurationProfileGenerationResult.configurationProfile ||
        !configurationProfileGenerationResult.profileFilename ||
        !configurationProfileGenerationResult.settingsEnforced) {
      // console.log('Sending instructions to sonnet.');
      // console.time('sonnet prompt');
      configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
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
      // console.timeEnd('sonnet prompt');
    }

    // Check the result from sonnet, and return an error if it failed.
    if(!configurationProfileGenerationResult ||
        configurationProfileGenerationResult.couldNotGenerateProfile ||
        !configurationProfileGenerationResult.configurationProfile ||
        !configurationProfileGenerationResult.profileFilename ||
        !configurationProfileGenerationResult.settingsEnforced) {
      if(this.req.isSocket){
        // If the sonnet result returned a reasonWhyAProfileCouldNotBeGenerated, broadcast an error event with a reason to the requesting user's socket, and leave the room.
        if(configurationProfileGenerationResult && configurationProfileGenerationResult.reasonWhyAProfileCouldNotBeGenerated){
          sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateProfile', reason: configurationProfileGenerationResult.reasonWhyAProfileCouldNotBeGenerated});
        } else {
          // Otherwise, if the sonnet generation failed, broadcast an error event with no reason.
          sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateProfile'});
        }
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
