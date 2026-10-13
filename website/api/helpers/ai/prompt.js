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
      example: 'claude-sonnet-5-5',
      // 'claude-sonnet-5-5'
      // 'claude-haiku-5-5'
      // 'claude-opus-4-8'
      moreInfoUrl: 'https://docs.anthropic.com/en/docs/about-claude/models',
      defaultsTo: 'claude-sonnet-5-5',
    },
    expectJson: { type: 'boolean', defaultsTo: false },
    systemPrompt: { type: 'string', example: 'Here is data about each computer, as JSON: ```[ … ]```' },
    effort: {
      type: 'string',
      description: 'Optional effort level for adaptive thinking (controls thinking depth vs. token/latency cost).  Only supported on Anthropic models with output_config.effort support (e.g. Claude Sonnet 5.5, Claude Haiku 5.5, Claude Opus 4.6+).  Ignored for models that don\'t support it (e.g. Claude Haiku 4.5).  Claude Haiku 5.5 runs at "low" when this is omitted.',
      example: 'low'
    },
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
    },

    refused: {
      description: 'The model declined to respond (stop_reason "refusal").',
      extendedDescription: 'Claude Sonnet 5.5 requests carry a server-side fallback, so this only fires when the fallback declined too or the refusal category is not retried.  Claude Haiku 5.5 has no server-side fallback, so a caller that must answer should handle this exit by retrying on a Sonnet model.',
      outputDescription: 'The `stop_details` from the API response, when present.',
    }

  },


  fn: async function ({prompt, baseModel, expectJson, systemPrompt, effort}) {

    // TODO: Write a comprehensive test suite that prompts hundreds of times in parallel to see which combo
    //       of JSON prompt suffix + base model works the best, through actual experimentation.  Then document
    //       those results, have them included in a benchmark script whose usage is documented here in the code
    //       for this .prompt() helper, and edit the prompt helper to automatically suggest using the correct
    //       base model when using `expectJson: true` (and of course, change it to use the best JSON prompt suffix).
    //      (^This would be a good starter task for a summer internship project)
    let JSON_PROMPT_SUFFIX = `

Please do not add any text outside of the JSON or wrap it in a code fence.  Never use newline characters within double quotes.`;

    let rawPromptResponse;

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // Anthropic API  [?]: https://docs.anthropic.com/en/api/messages
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    if (!sails.config.custom.anthropicSecret) {
      throw new Error('sails.config.custom.anthropicSecret not set.  (To play around, run `sails_custom__anthropicSecret=\'…\' sails console`.  You can get your API secret at https://console.anthropic.com/settings/keys.)');
    }//•

    let requestData = {
      model: baseModel,
      // Bumped from 4096 so that models with adaptive thinking on by default (e.g. Claude Sonnet 5.5)
      // have enough headroom for thinking tokens without truncating the actual response.
      max_tokens: 8192,// eslint-disable-line camelcase
      messages: [
        { role: 'user', content: prompt+(expectJson? JSON_PROMPT_SUFFIX : '') }
      ]
    };
    if (systemPrompt) {
      requestData.system = systemPrompt;
    }
    // Claude Haiku 5.5 thinks by default (at "medium" effort), and every Haiku call on the website is
    // there for speed, so it runs at "low" unless the caller asks for more.
    let effortLevel = effort;
    if (!effortLevel && baseModel === 'claude-haiku-5-5') {
      effortLevel = 'low';
    }
    if (effortLevel) {
      // output_config.effort (adaptive thinking) is only supported on some Anthropic models.
      // Claude Haiku 4.5, for example, does not accept it, so attaching it would cause a 4xx
      // from Anthropic.  Ignore the input for models that don't support it.
      if (baseModel === 'claude-haiku-4-5') {
        sails.log.warn(`The prompt helper received an "effort" input, but the specified baseModel (${baseModel}) does not support output_config.effort. This input will be ignored in this LLM generation.`);
      } else {
        requestData.output_config = { effort: effortLevel };// eslint-disable-line camelcase
      }
    }

    let requestHeaders = {
      'x-api-key': sails.config.custom.anthropicSecret,
      'anthropic-version': '2023-06-01',
      'content-type': 'application/json',
    };
    // Claude Sonnet 5.5's safety classifiers can decline benign security-adjacent prompts (e.g. osquery
    // SQL for finding vulnerable hosts).  Server-side fallback re-runs a declined request on the model
    // Anthropic recommends for that refusal category.  Claude Haiku 5.5 has no server-side fallback (its
    // declines surface as the `refused` exit), and older models don't accept the parameter.
    if (baseModel === 'claude-sonnet-5-5') {
      requestData.fallbacks = 'default';
      requestHeaders['anthropic-beta'] = 'server-side-fallback-2026-07-01';
    }

    let anthropicResponse = await sails.helpers.http.post('https://api.anthropic.com/v1/messages', requestData, requestHeaders)
    .intercept('non200Response', (serverResponse)=>{
      return new Error('Failed to generate result.  Error details from LLM: '+serverResponse);
    })
    .intercept((err)=>{
      return new Error('Failed to generate result.  Error communicating with LLM: '+err.stack);
    });

    if (anthropicResponse.stop_reason === 'refusal') {
      throw { refused: anthropicResponse.stop_details || {} };
    }

    // With adaptive thinking enabled, the first content block can be a `thinking` (or
    // `redacted_thinking`) block rather than the actual answer, so scan for the first `text`
    // block instead of assuming it is at index 0.
    let textBlock = _.find(anthropicResponse.content, { type: 'text' });
    if (!textBlock) {
      throw new Error('The LLM responded, but its response did not contain a text block.  Full response content: '+require('util').inspect(anthropicResponse.content, {depth: 3}));
    }
    rawPromptResponse = textBlock.text;

    // The response to our prompt might be JSON.
    let parsedPromptResponse;
    if (expectJson) {
      // If the JSON response is wrapped in a code fence, remove it before trying to parse it.
      let jsonResponse = rawPromptResponse.trim();
      if (jsonResponse.startsWith('```')) {
        jsonResponse = jsonResponse.replace(/^```(?:json)?\n?/, '').replace(/\n?```$/, '');
      }
      try {
        parsedPromptResponse = JSON.parse(jsonResponse);
      } catch (err) {
        throw new Error('Expecting JSON result from LLM, but when attemting to JSON.parse(…) it, an error occurred: '+err.stack+'\n P.S. Here is what the LLM returned (and what we were *trying* to parse as valid JSON):'+rawPromptResponse);
      }
    } else {
      parsedPromptResponse = rawPromptResponse;
    }

    return parsedPromptResponse;

  }


};

