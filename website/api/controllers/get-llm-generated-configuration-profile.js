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


    // Start generating the profile right away.  Sonnet 5.5 at low effort is the only configuration that
    // met the 95%-per-type and 10 s gates in the profile generator test suite (Haiku 5.5 was both less
    // accurate and slower, so there is no longer a speculative draft).  The preview call below only feeds
    // the settings preview the UI shows while this runs, so the two calls are in flight together.
    let generationPromise = (async ()=>{
      return await sails.helpers.ai.prompt.with({
        systemPrompt: systemPrompt,
        prompt: configurationProfilePrompt,
        baseModel: 'claude-sonnet-5-5',
        expectJson: true,
        effort: 'low',
      })
      .tolerate((err)=>{
        sails.log.warn(`When trying to generate a configuration profile for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
        return undefined;
      });
    })();

    // Build a prompt that names the settings the profile will probably enforce, so the admin has something to read while it is generated.
    let previewSystemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

An IT admin has asked for a ${promptConfig.description}, and another model is already writing it.  You do not write the profile.  You name, quickly, the settings the profile is likely to enforce, so the admin has something to read while it is generated.

"anticipatedSettings" is a preview, replaced by the real settings the moment the profile arrives.  A best guess is useful there and a long list is not, so name the settings this request asks for and nothing else, and return an empty array when you cannot name them.

Respond in JSON with this data shape:
{
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
    let previewResult = await sails.helpers.ai.prompt.with({
      systemPrompt: previewSystemPrompt,
      prompt: `Here are the instructions from an IT admin:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``,
      baseModel: 'claude-haiku-5-5',
      expectJson: true,
    })
    .tolerate((err)=>{
      sails.log.warn(`When trying to generate the settings preview for a user's configuration profile instructions, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      return undefined;
    });
    // Send the anticipatedSettings, anticipatedName, and anticipatedDescription to the socket with a 'settingsPreview' event.
    let anticipatedSettings = _.filter(previewResult ? previewResult.anticipatedSettings || [] : [], (setting)=>{
      return _.isObject(setting) && setting.name;
    });
    let anticipatedName = previewResult ? previewResult.anticipatedName : '';
    let anticipatedDescription = previewResult ? previewResult.anticipatedDescription : '';
    if(this.req.isSocket && anticipatedSettings.length > 0) {
      sails.sockets.broadcast(roomId, 'settingsPreview', {settings: anticipatedSettings, description: anticipatedDescription, name: anticipatedName});
    }

    let configurationProfileGenerationResult = await generationPromise;

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
      items: configurationProfileGenerationResult.settingsEnforced,
      additionalProfiles: [],
    };

    // Run the profile through the addAdmxInstallCommandsToWindowsProfile helper before returning it to the user.
    if(profileType === 'csp') {
      let withAdmxInstalls = await sails.helpers.addAdmxInstallCommandsToWindowsProfile.with({ profile: generatedProfile.profile });
      generatedProfile.profile = withAdmxInstalls.profile;
      generatedProfile.additionalProfiles = withAdmxInstalls.admxInstallProfiles || [];
      generatedProfile.deliveryNotes = _.compact([generatedProfile.deliveryNotes].concat(withAdmxInstalls.deliveryNotes)).join(' ');
    }

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
