module.exports = {


  friendlyName: 'Get llm generated configuration profile',


  description: 'Generate a configuration profile from an IT admin\'s instructions, sending a preview of its settings to the requesting socket while it is generated.',


  inputs: {
    profileType: {
      type: 'string',
      isIn: [
        'mobileconfig',
        'csp',
        'ddm',
        'android',
      ],
      required: true
    },
    naturalLanguageInstructions: {
      type: 'string',
      required: true
    },
    forFleet: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether the request comes from a Fleet server, which turns on Fleet\'s own rules.'
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


  fn: async function ({profileType, naturalLanguageInstructions, forFleet}) {

    // Generate a random room name.
    let roomId = await sails.helpers.strings.random();
    if(this.req.isSocket) {
      // Add the requesting socket to the room.
      sails.sockets.join(this.req, roomId);
    }

    // If this request is from a socket, start a preview of the settings the profile is likely to enforce, so the admin has
    // something to read while it is generated.  It is started before the configuration helper is called rather than after,
    // since the helper's lookups would otherwise delay it, and it is not awaited: a preview that arrives after the profile
    // is ignored by the page.
    if(this.req.isSocket) {
      // Mirrors the description of each profile type in the getConfigurationProfileGeneratorConfiguration helper, which
      // is only available once the helper has finished.
      let profileTypeDescriptions = {
        csp: 'CSP XML profile that enforces OS settings on Windows devices',
        mobileconfig: 'XML .mobileconfig profile that enforces OS settings on macOS/iOS/iPadOS devices',
        ddm: 'Apple DDM declaration in JSON format that enforces OS settings on macOS devices',
        android: 'Android Management API Policy in JSON format that enforces OS settings on Android devices',
      };
      let previewSystemPrompt = `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

An IT admin has asked for a ${profileTypeDescriptions[profileType]}, and another model is already writing it.  You do not write the profile.  You name the settings the profile is likely to enforce so the admin has something to read while it is generated.

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
      (async ()=>{
        let previewResult = await sails.helpers.ai.prompt.with({
          systemPrompt: previewSystemPrompt,
          prompt: `Here are the instructions from an IT admin:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\``,
          baseModel: 'claude-haiku-4-5',
          expectJson: true,
        })
        .tolerate((err)=>{
          sails.log.warn(`When trying to preview the settings in a user's configuration profile, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
          return undefined;
        });
        // Send the anticipatedSettings, anticipatedName, and anticipatedDescription to the socket with a 'settingsPreview' event.
        let anticipatedSettings = _.filter(previewResult ? previewResult.anticipatedSettings || [] : [], (setting)=>{
          return _.isObject(setting) && setting.name;
        });
        if(anticipatedSettings.length > 0) {
          sails.sockets.broadcast(roomId, 'settingsPreview', {settings: anticipatedSettings, description: previewResult.anticipatedDescription, name: previewResult.anticipatedName});
        }
      })();
    }

    // Get the prompts for this profile, and the configuration for this profile type.
    let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
      profileType,
      naturalLanguageInstructions,
      useLighterResponseShape: true,
      useApplePayloadTypeLookup: true,
      forFleet,
    });

    // Generate the configuration profile with Haiku, the model the profile generator's test suite measures these prompts against.
    let configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
      systemPrompt: generatorConfiguration.systemPrompt,
      prompt: generatorConfiguration.userPrompt,
      baseModel: 'claude-haiku-4-5',
      expectJson: true,
    })
    .tolerate((err)=>{
      sails.log.warn(`When trying to generate a configuration profile with the smaller model, an error occurred. Falling back to the larger model. Full error: ${require('util').inspect(err, {depth: 2})}`);
      return undefined;
    });

    // If Haiku did not generate a profile, either because of an error or because it returned the couldNotGenerateProfile
    // shape, send the same prompts to Sonnet.
    if(!configurationProfileGenerationResult ||
        configurationProfileGenerationResult.couldNotGenerateProfile ||
        !configurationProfileGenerationResult.configurationProfile ||
        !configurationProfileGenerationResult.profileFilename ||
        !configurationProfileGenerationResult.settingsEnforced) {
      configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
        systemPrompt: generatorConfiguration.systemPrompt,
        prompt: generatorConfiguration.userPrompt,
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
    }

    // Check the generated result, and return an error if a profile could not be generated.
    if(!configurationProfileGenerationResult ||
        configurationProfileGenerationResult.couldNotGenerateProfile ||
        !configurationProfileGenerationResult.configurationProfile ||
        !configurationProfileGenerationResult.profileFilename ||
        !configurationProfileGenerationResult.settingsEnforced) {
      if(this.req.isSocket){
        // If the result has a reasonWhyAProfileCouldNotBeGenerated, broadcast an error event with a reason to the requesting user's socket, and leave the room.
        if(configurationProfileGenerationResult && configurationProfileGenerationResult.reasonWhyAProfileCouldNotBeGenerated){
          sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateProfile', reason: configurationProfileGenerationResult.reasonWhyAProfileCouldNotBeGenerated});
        } else {
          // Otherwise, broadcast an error event with no reason.
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

    if(profileType === 'mobileconfig') {
      generatedProfile.deliveryNotes = await sails.helpers.addSoftwareUpdateRemovalNote.with({ profile: generatedProfile.profile, deliveryNotes: generatedProfile.deliveryNotes || '' });
    }

    // Strip any SyncML envelope and add ADMX install commands before returning a Windows profile to the user.
    if(profileType === 'csp') {
      generatedProfile.profile = await sails.helpers.removeSyncmlEnvelopeFromWindowsProfile.with({ profile: generatedProfile.profile });
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
