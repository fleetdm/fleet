module.exports = {


  friendlyName: 'Improve prompt',


  description: '',


  sideEffects: 'cacheable',


  inputs: {

    prompt: { type: 'string', required: true, example: 'Who is running macOS 15?' },
    cycles: { type: 'number', defaultsTo: 1 }

  },


  exits: {

    success: {
      outputFriendlyName: 'Improved prompt',
      outputDescription: 'The new and improved prompt.',
      outputType: 'string',
      outputExample: 'Who is running macOS 15?',
    },

  },


  fn: async function ({ prompt: originalPrompt, cycles }) {

    let newPrompt = originalPrompt;
    for (let i=0; i<cycles; i++) {
      let improverPrompt = 'Given a prompt, improve it to be a more effective prompt optimized for use with an LLM, and particularly this LLM.';
      improverPrompt += 'Prompt: ```\n';
      improverPrompt += `${newPrompt}\n`;
      improverPrompt += '```\n';
      improverPrompt += '\n';
      improverPrompt += 'Respond only with the exact text of the prompt.';
      improverPrompt += '\n';

      newPrompt = await sails.helpers.flow.build(async ()=>{
        // FUTURE: Add an option to run multiple times.
        let parsedPromptResponse = await sails.helpers.ai.prompt.with({
          baseModel: 'o4-mini-2025-04-16',
          prompt: improverPrompt,
        });
        return parsedPromptResponse;
      }).retry();
    }//∞

    return newPrompt;
  }


};

