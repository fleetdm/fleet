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
    },
    profileGenerationFailed: {
      description: 'The OpenAI API could not generate a configuration profile from the provided instructions.',
      responseType: 'badRequest'
    }
  },


  fn: async function ({profileType, naturalLanguageInstructions}) {

    let filteredSettings;
    if(profileType === 'csp'){
      let cspSettingsJson = await sails.helpers.fs.readJson(require('path').resolve(sails.config.appPath, 'CSP-policy-settings.json'));
      let prunedSettings = cspSettingsJson.map((setting)=>{
        let newSetting = {
          cspAreaName: setting.NodeName,
          settingsInCSPArea: setting.Nodes.map((node)=>{
            return node.NodeName;
          })
        };
        return newSetting;
      });
      // console.log(prunedSettings);
      // Filter down CSP settings.
      let settingsFiltrationPrompt = `Given this question from an IT admin, and using the provided context (a List of Microsoft CSP Policies setting names), return the subset of settings that might be relevant for designing a CSP policy profile to enforce the requested instructions on a windows device.

      Here is the question:
      \`\`\`
      ${naturalLanguageInstructions}
      \`\`\`

      Provided context:
      \`\`\`
      ${JSON.stringify(prunedSettings)}
      \`\`\`

      `;
      // throw new Error('stopping :)');
      filteredSettings = await sails.helpers.ai.prompt(settingsFiltrationPrompt, 'gpt-4o', true)
      .intercept((err)=>{
        return new Error(`When trying to get a subset of tables to use to generate a query for an Admin user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      });

      filteredSettings =
      `Provided context:
      \`\`\`
      ${JSON.stringify(filteredSettings)}
      \`\`\`
      `;
    }

    let promptStringByProfileType = {
      'csp': 'CSP profile that enforces OS settings on Windows devices using only settings documented on https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider',
      'mobileconfig': 'XML .mobileconfig profile that enforces OS settings on Apple devices using only documented settings from https://developer.apple.com/documentation/devicemanagement',
      'ddm': 'JSON Apple JSON DDM JSON in JSON MDM command JSON in JSON format that JSON is JSON and enforces JSON OS settings on Apple devices using only JSON and documented settings from https://developer.apple.com/documentation/devicemanagement using JSON. DDM commands should be in JSON.',
    };

    let configurationProfilePrompt = `Given this question from an IT admin, generate a ${promptStringByProfileType[profileType]}.

    Here are the instructions:
    \`\`\`
    ${naturalLanguageInstructions}
    \`\`\`


    ${filteredSettings ? filteredSettings : ''}

    When generating configuration profiles, follow these rules strictly:
    ${profileType === 'ddm' ? '- DDM commands should be formatted as JSON' : ''}
    - Use only officially supported, documented settings for the specified platform.
    - For any example variables in XML profiles, insert a comment on the line immediately above explaining what the user should replace, unless the generated result is formatted as JSON, this will break the formatting.
    - Output ONLY valid JSON with no extra text, markdown, or formatting.
    - Do not just transform the user's instructions into a configuration profile 1:1, only use officially documented setting and values for those settings for the specified platform.
    - The JSON must exactly match the following schema:

    {
      "reliabilityPercentage": "A realistic percentage (0-100) representing your confidence that the profile is correctly formatted and uses only documented settings and values.",
      "configurationProfile": "The complete configuration profile as a string",
      "profileFilename": "A suggested filename for saving the profile",
      "caveatsAboutThisProfile": "A list as an array of any potential caveats or limitations",
      "settingsEnforced": [
        {
          "name": "The key name of the enforced setting (e.g., 'LoginwindowText')",
          "value": "The value enforced for the setting"
        }
        // Additional settings objects as needed
      ]
    }

    If a valid configuration profile cannot be generated from the provided instructions, output this JSON:

    {
      "couldNotGenerateProfile": true
      "reason": "A one sentence simple explanation of this configuration profile could not be generated from the provided instructions."
    }`;

    // console.log(configurationProfilePrompt);
    let configurationProfileGenerationResult = await sails.helpers.ai.prompt.with({
      prompt: configurationProfilePrompt,
      // baseModel: 'gpt-4o',
      baseModel: 'o3-mini-2025-01-31',
      // baseModel: 'o1-preview',
      // expectJson: true
    }).intercept((err)=>{
      return new Error(`When trying generate a configuration profile for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
    });
    console.log(configurationProfileGenerationResult);
    let jsonResult = JSON.parse(configurationProfileGenerationResult);
    // console.log(configurationProfileGenerationResult);
    // All done.
    if(jsonResult.couldNotGenerateProfile && jsonResult.reason) {
      throw {'profileGenerationFailed': jsonResult.reason};
    }
    if(!jsonResult.configurationProfile || !jsonResult.profileFilename || !jsonResult.settingsEnforced){
      throw 'couldNotGenerateProfile';
    }
    return {
      profile: jsonResult.configurationProfile,
      profileFilename: jsonResult.profileFilename,
      items: jsonResult.settingsEnforced,
      caveats: jsonResult.caveatsAboutThisProfile,
    };

  }


};
