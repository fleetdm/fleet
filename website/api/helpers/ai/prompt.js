module.exports = {


  friendlyName: 'Prompt',


  description: 'Prompt a large language model (LLM).',


  inputs: {
    prompt: { type: 'string', required: true, example: 'Who is running macOS 15?' },
    baseModel: {
      type: 'string',
      description: 'The base model to use.',
      moreInfoUrl: 'https://platform.openai.com/docs/models/o1',
      defaultsTo: 'gpt-3.5-turbo',
      isIn: ['gpt-3.5-turbo', 'gpt-4o', 'o1-preview', 'o3-mini-2025-01-31', 'gpt-4o-2024-08-06', 'gpt-4o-mini-2024-07-18'],
    },
    expectJson: { type: 'boolean', defaultsTo: false },
    systemPrompt: { type: 'string', example: 'Here is data about each computer, as JSON: ```[ … ]```' },
  },


  exits: {

    success: {
      description: 'All done.',
      outputDescription: 'The output from the model, parsed as JSON, if appropriate.',
      outputExample: '*',
    },

    jsonExpectationFailed: {
      description: 'The model was supposed to respond with valid JSON, but it didn\'t.',
      extendedDescription: `It can be useful to call .prompt.with({expectJson: true, prompt:'How many fingers am I holding up?'}).retry('jsonExpectationFailed')`
    }

  },


  fn: async function ({prompt, baseModel, expectJson, systemPrompt}) {

    if (!sails.config.custom.openAiSecret) {
      throw new Error('sails.config.custom.openAiSecret not set.');
    }//•

    let JSON_PROMPT_SUFFIX = `

Please do not add any text outside of the JSON report or wrap it in a code fence.  Never use newline characters within double quotes.`;

    // The request data to send to openAI varies based on whether a system prompt was provided.
    let openAiResponse = await sails.helpers.http.post('https://api.openai.com/v1/chat/completions', {// [?] API: https://platform.openai.com/docs/api-reference/chat/create
      model: baseModel,
      messages: ((await sails.helpers.flow.build(()=>{
        if(systemPrompt && [// The specified baseModel might not support system prompts.
          'o1-preview',
          'o3-mini-2025-01-31'
        ].includes(baseModel)){
          sails.log.warn(`The prompt helper recieved a system prompt input, but the specified baseModel (${baseModel}) does not support a system prompt. This input will be ignored in this LLM generation, please remove the system prompt or use a different base model.`);
        } else if (systemPrompt) {// But it also might.
          return [
            { role: 'system', content: systemPrompt },
            { role: 'user', content: prompt+(expectJson? JSON_PROMPT_SUFFIX : '') }
          ];
        } else {//There might not BE a system prompt.
          return [
            { role: 'user', content: prompt+(expectJson? JSON_PROMPT_SUFFIX : '') }
          ];
        }
      })))
    }, {
      Authorization: `Bearer ${sails.config.custom.openAiSecret}`
    })
    .intercept('non200Response', (serverResponse)=>{
      return new Error('Failed to generate result.  Error details from LLM: '+serverResponse);
    })
    .intercept((err)=>{
      return new Error('Failed to generate result.  Error communicating with LLM: '+err.stack);
    });

    // The response to our prompt might be JSON.
    let rawPromptResponse = openAiResponse.choices[0].message.content;
    let parsedPromptResponse;
    if (expectJson) {
      try {
        parsedPromptResponse = JSON.parse(rawPromptResponse);
      } catch (err) {
        throw new Error('Expecting JSON result from LLM, but when attemting to JSON.parse(…) it, an error occurred: '+err.stack+'\n P.S. Here is what the LLM returned (and what we were *trying* to parse as valid JSON):'+rawPromptResponse);
      }
    } else {
      parsedPromptResponse = rawPromptResponse;
    }

    return parsedPromptResponse;

  }


};

