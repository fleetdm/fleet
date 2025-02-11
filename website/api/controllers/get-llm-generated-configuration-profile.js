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

    let promptStringByProfileType = {
      'csp': 'CSP XML profile that enforces OS settings on Windows devices',
      'mobileconfig': 'XML .mobileconfig profile that enforces OS settings on macOS devices',
      'ddm': 'Apple DDM MDM profile in XML format that enforces OS settings on macOS devices',
    };

    let configurationProfilePrompt = `Given this question from an IT admin, generate a ${promptStringByProfileType[profileType]}.

    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\`


    Please give me all of the above in JSON, with this data shape:

    {
      "configurationProfile": "TODO"
      "profileFilename": "TODO"
      "settingsEnforced": [// For each setting enforced by the configuration profile.
        {
          // The name (key) of the settings that is enforced. e.g., LoginwindowText
          name: "TODO",
          // The value of the setting that is enforced
          value: "TODO",
        },
        {...}
      ]
    }

    If a configuration profile cannot be generated from the provided instructions do not return the datashape above and instead return this JSON in this exact data shape:

    {
      "couldNotGenerateProfile": true
    }
    `;

    // console.log(configurationProfilePrompt);
    let configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
      prompt: configurationProfilePrompt,
      baseModel: 'o3-mini-2025-01-31',
      // expectJson: true
    }).intercept((err)=>{
      return new Error(`When trying generate a configuration profile for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
    });
    let jsonResult = JSON.parse(configurationProfileGenerationResult);
    // console.log(configurationProfileGenerationResult);
    // All done.
    if(jsonResult.couldNotGenerateProfile) {
      throw 'couldNotGenerateProfile';
    }
    if(!jsonResult.configurationProfile || !jsonResult.profileFilename || !jsonResult.settingsEnforced){
      throw 'couldNotGenerateProfile';
    }
    return {
      profile: jsonResult.configurationProfile,
      profileFilename: jsonResult.profileFilename,
      items: jsonResult.settingsEnforced
    };

  }


};
