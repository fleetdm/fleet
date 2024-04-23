module.exports = {


  friendlyName: 'Get human interpretation from osquery sql',


  description: 'Infer policy information from osquery SQL.',


  inputs: {

    fleetInstanceUrl: {
      type: 'string',
      required: true,
    },

    fleetApiKey: {
      type: 'string',
      required: true,
    },

    sql: {
      type: 'string',
      required: true
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Humanesque interpretation',
      outputDescription: 'If the call to the LLM fails, then a success response is sent with an explanation about the failure (e.g. "under heavy load", etc)',
      outputExample: {
        risks: 'Using an outdated macOS version risks exposure to security vulnerabilities and potential system instability.',
        whatWillProbablyHappenDuringMaintenance: 'We will update your macOS to version 14.4.1 to enhance security and stability.'
      }
    },

    fleetInstanceNotResponding: {
      description: 'A http request to the user\'s Fleet instance failed.',
      statusCode: 404,
    },

    invalidToken: {
      description: 'The provided token for the api-only user could not be used to authorize requests from fleetdm.com',
      statusCode: 403,
    },

  },


  fn: async function ({fleetInstanceUrl, fleetApiKey, sql}) {

    if (!sails.config.custom.openAiSecret) {
      throw new Error('sails.config.custom.openAiSecret not set.');
    }//•

    // Check the fleet instance url and API key provided
    let responseFromFleetInstance = await sails.helpers.http.get(fleetInstanceUrl+'/api/v1/fleet/me',{},{'Authorization': 'Bearer ' +fleetApiKey})
    .intercept('requestFailed', 'fleetInstanceNotResponding')
    .intercept('non200Response', 'invalidToken')
    .intercept((error)=>{
      return new Error(`When sending a request to a Fleet instance's /me endpoint to verify the token, an error occurred: ${error}`);
    });

    // Build our prompt
    let prompt = `Given this osquery policy: aka a query which either passes (≥1 row) or fails (0 rows) for a given laptop, what risks might we anticipate from that laptop having failed the policy?

Here is the query:
\`\`\`
${sql}
\`\`\`

Remember to minimize the number of words used!

Please give me all of the above in JSON, with this data shape:

\`\`\`
{
  risks: 'TODO',
  whatWillProbablyHappenDuringMaintenance: 'TODO'
}
\`\`\``;

    let BASE_MODEL = 'gpt-4';// The base model to use.  https://platform.openai.com/docs/models/gpt-4
    // (Max tokens for gpt-3.5 ≈≈ 4000) (Max tokens for gpt-4 ≈≈ 8000)
    // [?] API: https://platform.openai.com/docs/api-reference/chat/create

    let llmReport = await sails.helpers.http.post('https://api.openai.com/v1/chat/completions', {
      model: BASE_MODEL,
      messages: [// https://platform.openai.com/docs/guides/chat/introduction
        {
          role: 'user',
          content: prompt
        }
      ],
      temperature: 0.7,
      max_tokens: 256//eslint-disable-line camelcase
    }, {
      Authorization: `Bearer ${sails.config.custom.openAiSecret}`
    })
    .tolerate((err)=>{
      // FUTURE: Actual negotiate errors instead of just pretending it works but sending back garbage.
      sails.log.warn(failureMessage+'  Error details from LLM: '+err.stack);
      return;
    });

    // Get data into expected formaat
    let report;
    if (!llmReport) {// If LLM could not be reached…
      let failureMessage = 'Failed to generate human interpretation using generative AI.';
      report = {
        risks: failureMessage,
        whatWillProbablyHappenDuringMaintenance: failureMessage
      };
    } else {// Otherwise, descriptions were successfully generated…
      llmMessage = llmReport.choices[0].message.content;
      llmMessage = llmMessage.replace(/\`\`\`/g, '');
      report = JSON.parse(llmMessage);
    }

    return report;

  }


};
