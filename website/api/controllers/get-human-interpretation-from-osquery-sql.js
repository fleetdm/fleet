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

    if (!sails.config.custom.anthropicSecret) {
      throw new Error('sails.config.custom.anthropicSecret not set.');
    }//•

    // Build our prompt
    let prompt = `Given this osquery policy: aka a query which either passes (≥1 row) or fails (0 rows) for a given laptop, what risks might we anticipate from that laptop having failed the policy?

Here is the query:
\`\`\`
${sql}
\`\`\`

Remember to minimize the number of words used!

Please give me all of the above in JSON, with this data shape:

{
  risks: 'TODO',
  whatWillHappenDuringMaintenance: 'TODO'
}

Please do not add any text outside of the JSON report or wrap it in a code fence.`;
    // > Note that this returns `whatWillHappenDuringMaintenance` instead of `whatWillProbablyHappenDuringMaintenance`.
    // > This naming gets a better (more decisive-sounding) result from LLMs. We'll rename it for our final response.

    // Fallback message in case LLM API request fails.
    let failureMessage = 'Failed to generate human interpretation using generative AI.';

    let llmResponse = await sails.helpers.ai.prompt.with({prompt, expectJson: true, baseModel: 'claude-haiku-4-5'})
    .tolerate((err)=>{
      sails.log.warn(failureMessage+'  Error details from LLM: '+err.stack);
      return { risks: failureMessage, whatWillHappenDuringMaintenance: failureMessage};
    });

    let report;
    try {
      report = llmResponse;
      // Change `whatWillHappenDuringMaintenance` to `whatWillProbablyHappenDuringMaintenance` (the naming we want to use in our API response)
      report.whatWillProbablyHappenDuringMaintenance = report.whatWillHappenDuringMaintenance;
      delete report.whatWillHappenDuringMaintenance;
    } catch (err) {
      sails.log.warn('When trying to parse a report returned from the prompt helper, an error occurred. Error details: '+err.stack+'\n Report returned from the prompt helper:'+llmResponse);
      report = {
        risks: failureMessage,
        whatWillProbablyHappenDuringMaintenance: failureMessage
      };
    }

    return report;

  }


};
