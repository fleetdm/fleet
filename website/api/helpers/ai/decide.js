module.exports = {


  friendlyName: 'Decide',


  description: 'Make an intelligent determination about some data from the provided choices.',


  extendedDescription: 'e.g. for sentiment analysis for social network, or implementing "top posts" or featured content.',


  sideEffects: 'cacheable',


  inputs: {

    data: {
      type: 'json',
      required: true
    },

    choices: {
      type: {},
      required: true,
      description: 'The choices to pick from.',
      extendedDescription: 'Each choice consists of a key (a string that will be returned if this choice is selected) and a value (the "predicate", i.e. a phrase describing the data that could either be true or false).  You can think of this similar to how many popular `<select>`/UI dropdown components work in web frontend libraries.  One of the choices *MUST MATCH*, so be sure to include an "Anything else" option, lest you run into errors for any data that doesn\'t match your other provided choices.',
      example: {
        'Top post': 'A social media post…',
        'n/a': 'Anything else'
      }
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Decision',
      outputType: 'string',
      outputDescription: 'The choice that was decided upon.',
      outputExample: 'Top post',
      extendedDescription: 'The LLM will pick the choice that is the "most true", using the key from the provided dictionary.'
    },

  },


  fn: async function ({data, choices: predicatesByValue}) {
    let prompt = 'Given some data and a set of possible choices, decide which choice most accurately classifies the data.';

    // FUTURE: Add an option to first validate `choices` (e.g. for non-production envs or where accuracy is critical and the massive trade-off in increased response time is worthwhile) using a prompt that verifies it is an appropriately-formatted predicate.

    prompt += 'Data: ```\n';
    prompt += `${JSON.stringify(data)}\n`;
    prompt += '```\n';
    prompt += '\n';
    prompt += 'Choices:\n';
    for (let value in predicatesByValue) {
      prompt += ` • ${predicatesByValue[value]}\n`;
    }//∞
    prompt += '\n';
    prompt += 'Decide based on which choice is the most correct for the given data.  Respond only with the exact string value for the choice provided.';

    let decision = await sails.helpers.flow.build(async ()=>{
      let parsedPromptResponse = await sails.helpers.ai.prompt.with({
        baseModel: 'o4-mini-2025-04-16',
        prompt: prompt,
      });

      let chosenValue;
      for (let value in predicatesByValue) {
        if (predicatesByValue[value] === parsedPromptResponse) {
          chosenValue = value;
        }
      }//∞
      if (!chosenValue) {
        throw new Error('Response from LLM does not match provided choices.  The LLM said: \n```\n'+require('util').inspect(parsedPromptResponse,{depth:null})+'\n```\n\nBut the provided choices to pick from were: \n```\n'+require('util').inspect(predicatesByValue, {depth: null})+'\n```');
      }

      return chosenValue;

    }).retry();

    return decision;

  }


};

