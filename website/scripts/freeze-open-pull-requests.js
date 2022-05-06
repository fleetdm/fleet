module.exports = {


  friendlyName: 'Freeze open pull requests',


  description: 'Freeze existing pull requests open on https://github.com/fleetdm/fleet, except those that consist exclusively of changes to files where the author is the DRI, according to auto-approval rules.',


  inputs: {
    dry: { type: 'boolean', defaultsTo: false, description: 'Whether to do a dry run, and not actually freeze anything.' },
  },


  fn: async function ({dry: isDryRun}) {

    sails.log('Running custom shell script... (`sails run freeze-open-pull-requests`)');

    let owner = 'fleetdm';
    let repo = 'fleet';
    let baseHeaders = {
      'User-Agent': 'sails run freeze-open-pull-requests',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    // Fetch open pull requests
    // [?] https://docs.github.com/en/rest/pulls/pulls#list-pull-requests
    let openPullRequests = await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls`, {
      state: 'open',
      per_page: 100,//eslint-disable-line camelcase
    }, baseHeaders);
    // console.log(openPullRequests.length,'PRs returned!');


    // For all open pull requests…
    await sails.helpers.flow.simultaneouslyForEach(openPullRequests, async(pullRequest)=>{

      let prNumber = pullRequest.number;
      let prAuthor = pullRequest.user.login;

      // Freeze, if appropriate.
      // (Check the PR's author versus the intersection of DRIs for all changed files.)
      let isAuthorPreapproved = await sails.helpers.githubAutomations.getIsPrPreapproved.with({
        prNumber: prNumber,
        githubUserToCheck: prAuthor,
        isGithubUserMaintainerOrDoesntMatter: true// « doesn't matter here because no auto-approval is happening.  Worst case, a community PR to an area with a "*" in the DRI mapping remains unfrozen.
      });
      // let isAuthorPreapproved = await sails.helpers.flow.build(async()=>{

      //   let isDRIForAllChangedPathsStill = false;

      //   // [?] https://docs.github.com/en/rest/reference/pulls#list-pull-requests-files
      //   let changedPaths = _.pluck(await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/files`, {
      //     per_page: 100,//eslint-disable-line camelcase
      //   }, baseHeaders).retry(), 'filename');// (don't worry, it's the whole path, not the filename)

      //   isDRIForAllChangedPathsStill = _.all(changedPaths, (changedPath)=>{
      //     changedPath = changedPath.replace(/\/+$/,'');// « trim trailing slashes, just in case (b/c otherwise could loop forever)

      //     require('assert')(githubUserToCheck !== undefined);
      //     // sails.log.verbose(`…checking DRI of changed path "${changedPath}"`);

      //     let selfMergers = DRI_BY_PATH[changedPath] ? [].concat(DRI_BY_PATH[changedPath]) : [];// « ensure array
      //     if (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*'))) {
      //       return true;
      //     }//•
      //     let numRemainingPathsToCheck = changedPath.split('/').length;
      //     while (numRemainingPathsToCheck > 0) {
      //       let ancestralPath = changedPath.split('/').slice(0, -1 * numRemainingPathsToCheck).join('/');
      //       // sails.log.verbose(`…checking DRI of ancestral path "${ancestralPath}" for changed path`);
      //       let selfMergers = DRI_BY_PATH[ancestralPath] ? [].concat(DRI_BY_PATH[ancestralPath]) : [];// « ensure array
      //       if (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*'))) {
      //         return true;
      //       }//•
      //       numRemainingPathsToCheck--;
      //     }//∞
      //   });//∞

      //   if (isDRIForAllChangedPathsStill && changedPaths.length < 100) {
      //     return true;
      //   } else {
      //     return false;
      //   }
      // });

      sails.log(`#${prNumber} by @${prAuthor}:`, isAuthorPreapproved ? 'Skipping freeze…' : 'Freezing…');

      if (!isDryRun) {
        console.log('NOT A DRY RUN!');
        // [?] https://docs.github.com/en/rest/reference/pulls#create-a-review-for-a-pull-request
        // await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/reviews`, {
        //   event: 'REQUEST_CHANGES',
        //   body: 'The repository has been frozen for an upcoming release.  In case of emergency, you can dismiss this review and merge.'
        // }, baseHeaders);
      }//
    });

  }


};

