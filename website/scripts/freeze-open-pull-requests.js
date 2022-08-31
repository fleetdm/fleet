module.exports = {

  friendlyName: 'Freeze open pull requests',


  description: 'Freeze existing pull requests open on https://github.com/fleetdm/fleet, except those that consist exclusively of changes to files where the author is the DRI, according to auto-approval rules.',


  extendedDescription: `# Usage

  ## Dry run
  sails_custom__githubAccessToken=YOUR_TOKEN_HERE sails run scripts/freeze-open-pull-requests.js --dry

  ## The real deal
  sails_custom__githubAccessToken=YOUR_TOKEN_HERE sails run scripts/freeze-open-pull-requests.js --limit=100`,


  inputs: {
    dry: { type: 'boolean', defaultsTo: false, description: 'Whether to do a dry run, and not actually freeze anything.' },
    limit: { type: 'number', defaultsTo: 100, description: 'The max number of PRs to examine and potentially freeze. (Useful for testing.)' },
  },


  fn: async function () {
    // Is this in use?
    // > For context on the history of this bit of code, which has gone been
    // > implemented a couple of different ways, and gone back and forth, check out:
    // > https://github.com/fleetdm/fleet/pull/5628#issuecomment-1196175485
    throw new Error('Not currently in use.  See comments in this script for more information.');
  }

  // fn: async function ({dry: isDryRun, limit: maxNumPullRequestsToCheck }) {

  //   sails.log('Running custom shell script... (`sails run freeze-open-pull-requests`)');

  //   let owner = 'fleetdm';
  //   let repo = 'fleet';
  //   let baseHeaders = {
  //     'User-Agent': 'sails run freeze-open-pull-requests',
  //     'Authorization': `token ${sails.config.custom.githubAccessToken}`
  //   };

  //   // Fetch open pull requests
  //   // [?] https://docs.github.com/en/rest/pulls/pulls#list-pull-requests
  //   let openPullRequests = await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls`, {
  //     state: 'open',
  //     per_page: maxNumPullRequestsToCheck,//eslint-disable-line camelcase
  //   }, baseHeaders);

  //   if (openPullRequests.length > maxNumPullRequestsToCheck) {
  //     openPullRequests = openPullRequests.slice(0,maxNumPullRequestsToCheck);
  //   }

  //   let SECONDS_TO_WAIT = 5;
  //   sails.log(`Examining and potentially freezing ${openPullRequests.length} PRs very soon…  (To cancel, press CTRL+C quickly within ${SECONDS_TO_WAIT}s!)`);
  //   await sails.helpers.flow.pause(SECONDS_TO_WAIT*1000);

  //   // For all open pull requests…
  //   await sails.helpers.flow.simultaneouslyForEach(openPullRequests, async(pullRequest)=>{

  //     let prNumber = pullRequest.number;
  //     let prAuthor = pullRequest.user.login;
  //     require('assert')(prAuthor !== undefined);

  //     // Freeze, if appropriate.
  //     // (Check versus the intersection of DRIs for all changed files to make sure SOMEONE is preapproved for all of them.)
  //     let isAuthorPreapproved = await sails.helpers.githubAutomations.getIsPrPreapproved.with({
  //       prNumber: prNumber,
  //       isGithubUserMaintainerOrDoesntMatter: true// « doesn't matter here because no auto-approval is happening.  Worst case, a community PR to an area with a "*" in the DRI mapping remains unfrozen.
  //     });

  //     if (isDryRun) {
  //       sails.log(`#${prNumber} by @${prAuthor}:`, isAuthorPreapproved ? 'Would have skipped freeze…' : 'Would have frozen…');
  //     } else {
  //       sails.log(`#${prNumber} by @${prAuthor}:`, isAuthorPreapproved ? 'Skipping freeze…' : 'Freezing…');
  //       if (!isAuthorPreapproved) {
  //         // [?] https://docs.github.com/en/rest/reference/pulls#create-a-review-for-a-pull-request
  //         await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/reviews`, {
  //           event: 'REQUEST_CHANGES',
  //           body: 'The repository has been frozen for an upcoming release.  In case of emergency, you can dismiss this review and merge.'
  //         }, baseHeaders);
  //       }//ﬁ
  //     }
  //   });

  // }

};

