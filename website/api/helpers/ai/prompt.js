module.exports = {


  friendlyName: 'Prompt',


  description: 'Prompt a large language model (LLM).',


  extendedDescription: 'e.g. chatbot, automatically fill out metadata on a user profile',


  sideEffects: 'cacheable',


  inputs: {
    prompt: { type: 'string', required: true, example: 'Who is running macOS 15?' },
    baseModel: {
      type: 'string',
      description: 'The base model to use.',
      example: 'gpt-4o',
      // 'o4-mini-2025-04-16'
      // 'o3-2025-04-16'
      // 'o1-preview'
      // 'o3-mini-2025-01-31'
      // 'gpt-4o-2024-08-06'
      // 'gpt-4o-mini-2024-07-18'
      // 'gpt-4.1-2025-04-14'
      moreInfoUrl: 'https://platform.openai.com/docs/models',
      defaultsTo: 'gpt-3.5-turbo',
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
      throw new Error('sails.config.custom.openAiSecret not set.  (To play around, run `sails_custom__openAiSecret=\'…\' sails console`.  You can get your API secret at https://platform.openai.com/settings/organization/api-keys.)');
    }//•

    // TODO: Write a comprehensive test suite that prompts hundreds of times in parallel to see which combo
    //       of JSON prompt suffix + base model works the best, through actual experimentation.  Then document
    //       those results, have them included in a benchmark script whose usage is documented here in the code
    //       for this .prompt() helper, and edit the prompt helper to automatically suggest using the correct
    //       base model when using `expectJson: true` (and of course, change it to use the best JSON prompt suffix).
    //      (^This would be a good starter task for a summer internship project)
    let JSON_PROMPT_SUFFIX = `

Please do not add any text outside of the JSON or wrap it in a code fence.  Never use newline characters within double quotes.`;

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

