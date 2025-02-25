module.exports = {


  friendlyName: 'Get "is PR preapproved?"',


  description: '',


  inputs: {
    repo: { type: 'string', example: 'fleet', required: true, isIn: ['fleet', 'confidential']},
    prNumber: { type: 'number', example: 382, required: true },
    githubUserToCheck: { type: 'string', example: 'mikermcneil', description: 'If excluded, then this returns `true` if all of the PRs changed paths are preapproved for SOMEONE.' },
    isGithubUserMaintainerOrDoesntMatter: { type: 'boolean', required: true, description: 'Whether (a) the user is a maintainer, or (b) it even matters for this check whether the user is a maintainer.' },// FUTURE: « this could be replaced with an extra GitHub API call herein, but doesn't seem worth it
  },


  exits: {

    success: {
      outputFriendlyName: 'Is PR preapproved?',
      outputDescription: 'Whether the provided GitHub user is a maintainer for all changed paths.',
      outputType: 'boolean',
    },

  },


  fn: async function ({repo, prNumber, githubUserToCheck, isGithubUserMaintainerOrDoesntMatter}) {

    require('assert')(sails.config.custom.githubRepoMaintainersByPath);
    require('assert')(sails.config.custom.confidentialGithubRepoMaintainersByPath);
    require('assert')(sails.config.custom.fleetMdmGitopsGithubRepoMaintainersByPath);
    require('assert')(sails.config.custom.githubAccessToken);

    let MAINTAINERS_BY_PATH = sails.config.custom.githubRepoMaintainersByPath;

    if (repo === 'confidential') {
      MAINTAINERS_BY_PATH = sails.config.custom.confidentialGithubRepoMaintainersByPath;
    }

    if (repo === 'fleet-gitops') {
      MAINTAINERS_BY_PATH = sails.config.custom.fleetMdmGitopsGithubRepoMaintainersByPath;
    }

    let owner = 'fleetdm';
    let baseHeaders = {
      'User-Agent': 'Fleet auto-approve',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    // Check the PR's author versus the intersection of maintainers for all changed files.
    return await sails.helpers.flow.build(async()=>{

      let isMaintainerForAllChangedPathsStill = false;

      // [?] https://docs.github.com/en/rest/reference/pulls#list-pull-requests-files
      let changedPaths = _.pluck(await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/files`, {
        per_page: 100,//eslint-disable-line camelcase
      }, baseHeaders).retry(), 'filename');// (don't worry, it's the whole path, not the filename)

      isMaintainerForAllChangedPathsStill = _.all(changedPaths, (changedPath)=>{
        changedPath = changedPath.replace(/\/+$/,'');// « trim trailing slashes, just in case (b/c otherwise could loop forever)

        // sails.log.verbose(`…checking maintainership of changed path "${changedPath}"`);

        let selfMergers = MAINTAINERS_BY_PATH[changedPath] ? [].concat(MAINTAINERS_BY_PATH[changedPath]) : [];// « ensure array
        if (!githubUserToCheck && selfMergers.length >= 1) {// « not checking a user, so just make sure all these paths are preapproved for SOMEONE
          return true;
        }
        if (githubUserToCheck && (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*')))) {
          return true;
        }//•
        let numRemainingPathsToCheck = changedPath.split('/').length;
        while (numRemainingPathsToCheck > 0) {
          let ancestralPath = changedPath.split('/').slice(0, -1 * numRemainingPathsToCheck).join('/');
          // sails.log.verbose(`…checking maintainers of ancestral path "${ancestralPath}" for changed path`);
          let selfMergers = MAINTAINERS_BY_PATH[ancestralPath] ? [].concat(MAINTAINERS_BY_PATH[ancestralPath]) : [];// « ensure array
          if (!githubUserToCheck && selfMergers.length >= 1) {// « not checking a user, so just make sure all these paths are preapproved for SOMEONE
            return true;
          }
          if (githubUserToCheck && (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*')))) {
            return true;
          }//•
          numRemainingPathsToCheck--;
        }//∞
      });//∞

      if (isMaintainerForAllChangedPathsStill && changedPaths.length < 100) {
        return true;
      } else {
        return false;
      }
    });

  }


};

