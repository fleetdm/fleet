module.exports = {


  friendlyName: 'Get human interpretation from osquery sql',


  description: 'Infer policy information from osquery SQL.',


  inputs: {

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

  },


  fn: async function ({sql}) {

    if (!sails.config.custom.openAiSecret) {
      throw new Error('sails.config.custom.openAiSecret not set.');
    }//•

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
    // Fallback message in case LLM API request fails.
    let failureMessage = 'Failed to generate human interpretation using generative AI.';

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
      sails.log.warn(failureMessage+'  Error details from LLM: '+err.stack);
      return;
    });

    // Get data into expected formaat
    let report;
    if (!llmReport) {// If LLM could not be reached…
      // FUTURE: Actually negotiate errors instead of just pretending it works but sending back garbage.
      report = {
        risks: failureMessage,
        whatWillProbablyHappenDuringMaintenance: failureMessage
      };
    } else {// Otherwise, descriptions were successfully generated…
      let llmMessage = llmReport.choices[0].message.content;
      llmMessage = llmMessage.replace(/\`\`\`/g, '');
      report = JSON.parse(llmMessage);
    }

    return report;

  }


};
