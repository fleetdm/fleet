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
      example: 'claude-sonnet-5',
      // 'claude-sonnet-5'
      // 'claude-haiku-4-5'
      // 'claude-opus-4-8'
      moreInfoUrl: 'https://docs.anthropic.com/en/docs/about-claude/models',
      defaultsTo: 'claude-sonnet-5',
    },
    expectJson: { type: 'boolean', defaultsTo: false },
    systemPrompt: { type: 'string', example: 'Here is data about each computer, as JSON: ```[ … ]```' },
    systemPromptCacheTtl: {
      type: 'string',
      isIn: ['5m', '1h'],
      description: 'Cache the system prompt for this long, so later requests with the same system prompt skip reprocessing it.',
      extendedDescription: 'Only worth setting when the system prompt is byte-for-byte identical across requests: anything that varies per request belongs in uncachedSystemPromptSuffix or the prompt instead.  The cache is per model, and a system prompt shorter than the model\'s minimum (4096 tokens on Claude Haiku 4.5, 512 on Claude Sonnet 5.5 and Claude Haiku 5.5) is silently not cached.',
      moreInfoUrl: 'https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching',
    },
    uncachedSystemPromptSuffix: {
      type: 'string',
      description: 'More system prompt, sent after the cached part, for the parts of a system prompt that vary per request.',
    },
    effort: {
      type: 'string',
      description: 'Optional effort level for adaptive thinking (controls thinking depth vs. token/latency cost).  Only supported on Anthropic models with output_config.effort support (e.g. Claude Sonnet 5.5, Claude Haiku 5.5, Claude Opus 4.6+).  Ignored for Claude Haiku 4.5, which does not support it.  When omitted, the model\'s own default applies, which is "medium" on Claude Haiku 5.5.',
      example: 'low'
    },
    disableThinking: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Turn thinking off.  Claude Haiku 5.5 and Claude Sonnet 5.5 accept this at "high" effort or below.',
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
      extendedDescription: 'Claude Sonnet 5.5 requests carry a server-side fallback for "cyber" and "frontier_llm" declines, so for those this only fires when the fallback declined too.  Every other refusal category, and every Claude Haiku 5.5 refusal, fires it directly.',
      outputDescription: 'The `stop_details` from the API response, when present.',
    }

  },


  fn: async function ({prompt, baseModel, expectJson, systemPrompt, systemPromptCacheTtl, uncachedSystemPromptSuffix, effort, disableThinking}) {

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
    if (systemPrompt && systemPromptCacheTtl) {
      // The breakpoint goes on the system prompt itself rather than through top-level automatic caching,
      // which would put it after the per-request user message and write an entry no later request reads.
      requestData.system = [
        { type: 'text', text: systemPrompt, 'cache_control': { type: 'ephemeral', ttl: systemPromptCacheTtl } },
      ];
      if (uncachedSystemPromptSuffix) {
        requestData.system.push({ type: 'text', text: uncachedSystemPromptSuffix });
      }
    } else if (systemPrompt) {
      requestData.system = _.compact([systemPrompt, uncachedSystemPromptSuffix]).join('\n\n');
    }
    if (effort) {
      // Claude Haiku 4.5, under its dated model IDs too, does not accept output_config.effort, and attaching it
      // would cause a 4xx from Anthropic.
      if (_.startsWith(baseModel, 'claude-haiku-4-5')) {
        sails.log.warn(`The prompt helper received an "effort" input, but the specified baseModel (${baseModel}) does not support output_config.effort. This input will be ignored in this LLM generation.`);
      } else {
        requestData.output_config = { effort };// eslint-disable-line camelcase
      }
    }
    // Claude Sonnet 5.5 rejects "disabled" and turns thinking off with "between_tools" instead.  Claude Haiku
    // 4.5 only thinks when asked to, so there is nothing to turn off.
    if (disableThinking && baseModel === 'claude-sonnet-5-5') {
      requestData.thinking = { type: 'between_tools' };
    } else if (disableThinking && !_.startsWith(baseModel, 'claude-haiku-4-5')) {
      requestData.thinking = { type: 'disabled' };
    }

    let requestHeaders = {
      'x-api-key': sails.config.custom.anthropicSecret,
      'anthropic-version': '2023-06-01',
      'content-type': 'application/json',
    };
    // Claude Sonnet 5.5's safety classifiers can decline benign security-adjacent prompts.  Server-side fallback
    // re-runs a "cyber" or "frontier_llm" decline on Claude Sonnet 5; other categories are not retried and
    // surface as the `refused` exit.  Claude Haiku 5.5 has no server-side fallback, and older models don't
    // accept the parameter.
    if (baseModel === 'claude-sonnet-5-5') {
      requestData.fallbacks = 'default';
      requestHeaders['anthropic-beta'] = 'server-side-fallback-2026-07-01';
    }

    let anthropicResponse = await sails.helpers.http.post('https://api.anthropic.com/v1/messages', requestData, requestHeaders)
    .intercept('non200Response', (non200Error)=>{
      let err = new Error(`Failed to generate result.  The LLM API responded with a ${non200Error.raw.statusCode} status code: ${non200Error.raw.body}`);
      // So a caller can tell a rate limit (429) or an overloaded API (529) from a request that was wrong.
      err.statusCode = non200Error.raw.statusCode;
      return err;
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
        // A model sometimes writes a note, or a corrected second answer, after its JSON.  The last complete
        // object is the answer it means.
        parsedPromptResponse = lastJsonObjectIn(jsonResponse);
        if (parsedPromptResponse === undefined) {
          throw new Error('Expecting JSON result from LLM, but when attemting to JSON.parse(…) it, an error occurred: '+err.stack+'\n P.S. Here is what the LLM returned (and what we were *trying* to parse as valid JSON):'+rawPromptResponse);
        }
      }
    } else {
      parsedPromptResponse = rawPromptResponse;
    }

    return parsedPromptResponse;

  }


};


/**
 * The last complete JSON object in some text, or undefined when there is none.
 *
 * @param  {String} text
 * @returns {Dictionary|undefined}
 */
function lastJsonObjectIn(text) {
  let objects = [];
  let depth = 0;
  let start;
  let inString = false;
  for (let i = 0; i < text.length; i++) {
    let character = text[i];
    if (inString) {
      if (character === '\\') {
        i++;
      } else if (character === '"') {
        inString = false;
      }
      continue;
    }
    if (character === '"') {
      inString = true;
    } else if (character === '{') {
      if (depth === 0) {
        start = i;
      }
      depth++;
    } else if (character === '}' && depth > 0) {
      depth--;
      if (depth === 0) {
        objects.push(text.slice(start, i + 1));
      }
    }
  }
  for (let candidate of objects.reverse()) {
    try {
      return JSON.parse(candidate);
    } catch (unusedErr) {
      // Not JSON after all; try the one before it.
    }
  }
  return undefined;
}
