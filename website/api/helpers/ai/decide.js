module.exports = {


  friendlyName: 'Decide',


  description: 'Make an intelligent determination about some data from the provided choices.',


  inputs: {

    choices: {
      type: [{ predicate: 'string', value: 'json' }],
      required: true,
      description: 'The choices to pick from.',
      extendedDescription: 'Each choice includes a `.predicate`, i.e. a phrase describing the data that could either be true or false, and a JSON `.value` that will be returned if this choice is selected.  You can think of this similar to how many popular `<select>`/UI dropdown components work in web frontend libraries.  One of the choices *MUST MATCH*, so be sure to include an "Anything else" option, lest you run into errors for any data that doesn\'t match your other provided choices.',
      example: [
        {
          predicate: 'A social media post that is both (a) interesting and (b) in reasonably good taste',
          value: 'Top post'
        },
        {
          predicate: 'Anything else',
          value: 'n/a'
        },
      ]
    },

    data: {
      type: 'json',
      required: true
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Decision',
      outputType: 'json',
      outputDescription: 'The `.value` from the choice that was decided upon.',
      extendedDescription: 'The LLM will pick the choice that is the "most true".'
    },

  },


  fn: async function ({choices, data}) {
    let prompt = 'Given some data and a set of possible choices, decide which choice most accurately classifies the data.';

    // FUTURE: Add an option to first validate `choices` (e.g. for non-production envs or where accuracy is critical and the massive trade-off in increased response time is worthwhile) using a prompt that verifies it is an appropriately-formatted predicate.

    prompt += 'Data: ```\n';
    prompt += `${JSON.stringify(data)}\n`;
    prompt += '```\n';
    prompt += '\n';
    prompt += 'Choices:\n';
    for (let choice of choices) {
      prompt += ` • ${choice.predicate}\n`;
    }//∞
    prompt += '\n';
    prompt += 'Decide based on which choice\'s `.predicate` is the most correct for the given data.  Respond only with the JSON `.value` of the choice.';

    let decision = await sails.helpers.flow.build(async ()=>{
      let parsedPromptResponse = await sails.helpers.ai.prompt.with({
        expectJson: true,
        baseModel: 'o4-mini-2025-04-16',
        prompt: prompt,
      })
      .retry('jsonExpectationFailed');

      let choice = _.find(choices, { predicate: parsedPromptResponse });
      if (!choice) {
        throw new Error('Response from LLM does not match provided choices.  The LLM said: \n```\n'+parsedPromptResponse+'\n```\n\nBut the provided choices to pick from were: \n```\n'+require('util').inspect(choices, {depth: null})+'\n```');
      }

      return choice.value;

    }).retry();

    return decision;

  }


};

