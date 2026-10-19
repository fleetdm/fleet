module.exports = {


  friendlyName: 'Get llm generated configuration profile name and description',


  description: 'Name and describe the configuration profile an IT admin\'s instructions ask for, quickly enough to show while the profile itself is generated.',


  inputs: {
    platform: {
      type: 'string',
      isIn: ['apple', 'windows'],
      required: true
    },
    naturalLanguageInstructions: {
      type: 'string',
      required: true
    }
  },


  exits: {
    success: {
      outputDescription: '{name, description}',
      outputExample: {name: 'Disable AirDrop', description: 'Prevents AirDrop from being turned on, so files can\'t be shared with nearby devices over it.'},
    },
    invalidInput: {
      description: 'The instructions are too long.',
      responseType: 'badRequest'
    },
    couldNotNameProfile: {
      description: 'The model declined to name and describe a profile for these instructions.',
      responseType: 'badRequest'
    }
  },


  fn: async function ({platform, naturalLanguageInstructions}) {

    // Counted the same way as for the profile itself.
    const MAX_INSTRUCTIONS_LENGTH = 2000;
    if(Array.from(naturalLanguageInstructions).length > MAX_INSTRUCTIONS_LENGTH) {
      throw {invalidInput: {error: 'invalidInput', reason: `Instructions must be ${MAX_INSTRUCTIONS_LENGTH} characters or fewer.`}};
    }

    // Fleet's limits on a profile's name and description.
    const MAX_NAME_LENGTH = 255;
    const MAX_DESCRIPTION_LENGTH = 1023;

    // An Apple profile becomes a declaration or a .mobileconfig only once it is generated, which happens at the
    // same time as this, so a description that named a key or a payload could contradict the profile it is
    // shown beside.  It describes the effect instead.  A Windows profile is always CSP, so it can name nodes.
    let whatToDescribe = {
      apple: `an Apple configuration profile.  Describe only what the profile does for the people and devices it
reaches.  Never name a key, payload type, declaration type, or file format: whether the profile ends up as
a declaration or a .mobileconfig is decided elsewhere, after you answer.`,
      windows: `a Windows CSP configuration profile.  The description may name the CSP node it sets, in backticks, e.g.
\`./Device/Vendor/MSFT/Policy/Config/Storage/RemovableDiskDenyWriteAccess\`.`,
    }[platform];

    let result = await sails.helpers.ai.prompt.with({
      systemPrompt: `Return ONLY a raw JSON object.  Do not include \`\`\`json, \`\`\`, or any markdown formatting.  Do not include any explanation or text before or after the JSON.  Your entire response must be valid JSON.

An IT admin has asked for ${whatToDescribe}

Another model is writing the profile.  You name it and describe it, so the admin has something to read while it is written.

"name" is a short title an IT admin would give the profile, for what it enforces: "Disable AirDrop", "Require 12-character passwords".  Keep it under 60 characters.
"description" is one or two plain sentences saying what the profile changes.  Keep it under 300 characters.

Respond in JSON with this data shape:
{
  "name": "TODO",
  "description": "TODO"
}`,
      prompt: `Here are the instructions from an IT admin:
\`\`\`
${naturalLanguageInstructions}
\`\`\``,
      baseModel: 'claude-haiku-5-5',
      effort: 'low',
      disableThinking: true,
      expectJson: true,
    })
    .tolerate('refused', ()=>{
      return undefined;
    });

    // A refusal is a decline, reported the way the profile endpoint reports one, rather than a failure.
    if(result === undefined) {
      throw {couldNotNameProfile: {error: 'couldNotNameProfile', reason: 'These instructions could not be named and described as a configuration profile.'}};
    }

    if(!_.isObject(result) || !_.isString(result.name) || !_.isString(result.description) || !result.name.trim()) {
      throw new Error(`The model's name and description were not usable: ${require('util').inspect(result, {depth: 2})}`);
    }
    // Cut to Fleet's limits as a backstop, counted in code points like the instructions are.
    return {
      name: Array.from(result.name.trim()).slice(0, MAX_NAME_LENGTH).join(''),
      description: Array.from(result.description.trim()).slice(0, MAX_DESCRIPTION_LENGTH).join(''),
    };

  }


};
