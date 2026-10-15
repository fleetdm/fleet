module.exports = {


  friendlyName: 'Dispatch Claude workflow',


  description: 'Ask GitHub Actions to have Claude plan or implement an issue in the fleetdm/fleet repo.',


  extendedDescription: 'Starts .github/workflows/claude-plan-and-implement.yml.  The plan/implement work itself (reading the codebase, editing files, opening a PR) runs in that workflow, not here, because it is too long-running for a web request.',


  inputs: {
    issueNumber: { type: 'number', example: 1234, required: true },
    mode: { type: 'string', isIn: ['plan', 'implement'], required: true },
  },


  exits: {
    success: { description: 'The workflow was dispatched.' },
  },


  fn: async function ({issueNumber, mode}) {

    if (!sails.config.custom.githubAccessToken) {
      throw new Error('No GitHub access token configured!  (Please set `sails.config.custom.githubAccessToken`.  It needs permission to run workflows in fleetdm/fleet ("Actions: write").)');
    }//•

    // [?] https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event
    await sails.helpers.http.post('https://api.github.com/repos/fleetdm/fleet/actions/workflows/claude-plan-and-implement.yml/dispatches',
      {
        ref: 'main',
        inputs: { issue_number: String(issueNumber), mode },//eslint-disable-line camelcase
      },
      {
        'User-Agent': 'Fleetie pie',
        'Authorization': `token ${sails.config.custom.githubAccessToken}`
      }
    )
    .timeout(5000);

  }


};
