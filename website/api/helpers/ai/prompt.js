module.exports = {


  friendlyName: 'Prompt',


  description: 'Prompt a large language model (LLM).',


  inputs: {
    prompt: { type: 'string', required: true, example: 'Who is running macOS 15?' },
    baseModel: { type: 'string', defaultsTo: 'gpt-3.5-turbo', isIn: ['gpt-3.5-turbo', 'gpt-4o', 'o1-preview'] },
    expectJson: { type: 'boolean', defaultsTo: false },
  },


  exits: {

    success: {
      description: 'All done.',
      outputDescription: 'The output from the model, parsed as JSON, if appropriate.',
      outputExample: '*',
    },

  },


  fn: async function ({prompt, baseModel, expectJson}) {

    if (!sails.config.custom.openAiSecret) {
      throw new Error('sails.config.custom.openAiSecret not set.');
    }//•

    // The base model to use.  https://platform.openai.com/docs/models/o1
    let failureMessage = 'Failed to generate result via generative AI.';// Fallback message in case LLM API request fails.

    let JSON_PROMPT_SUFFIX = `

Please do not add any text outside of the JSON report or wrap it in a code fence.  Never use newline characters within double quotes.`;

    // [?] API: https://platform.openai.com/docs/api-reference/chat/create
    let openAiResponse = await sails.helpers.http.post('https://api.openai.com/v1/chat/completions', {
      model: baseModel,
      messages: [ {
        content: prompt+(expectJson? JSON_PROMPT_SUFFIX : ''),
        role: 'user',
      } ],
    }, {
      Authorization: `Bearer ${sails.config.custom.openAiSecret}`
    })
    .intercept((err)=>{
      return new Error(failureMessage+'  Error details from LLM: '+err.stack);
    });

    let rawResult = openAiResponse.choices[0].message.content;
    if (!expectJson) {
      return rawResult;
    } else {
      let parsedResult;
      try {
        parsedResult = JSON.parse(rawResult);
      } catch (err) {
        throw new Error('When trying to parse a JSON result returned from the Open AI API, an error occurred. Error details from JSON.parse: '+err.stack+'\n Here is what was returned from Open AI:'+openAiResponse.choices[0].message.content);
      }
      return parsedResult;
    }
  }


};

