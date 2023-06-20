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
      outputDescription: 'Whether the provided GitHub user is the DRI for all changed paths.',
      outputType: 'boolean',
    },

  },


  fn: async function ({repo, prNumber, githubUserToCheck, isGithubUserMaintainerOrDoesntMatter}) {

    require('assert')(sails.config.custom.githubRepoDRIByPath);
    require('assert')(sails.config.custom.confidentialGithubRepoDRIByPath);
    require('assert')(sails.config.custom.fleetMdmGitopsGithubRepoDRIByPath);
    require('assert')(sails.config.custom.githubAccessToken);

    let DRI_BY_PATH = sails.config.custom.githubRepoDRIByPath;

    if (repo === 'confidential') {
      DRI_BY_PATH = sails.config.custom.confidentialGithubRepoDRIByPath;
    }

    if (repo === 'fleet-mdm-gitops') {
      DRI_BY_PATH = sails.config.custom.fleetMdmGitopsGithubRepoDRIByPath;
    }

    let owner = 'fleetdm';
    let baseHeaders = {
      'User-Agent': 'Fleet auto-approve',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    // Check the PR's author versus the intersection of DRIs for all changed files.
    return await sails.helpers.flow.build(async()=>{

      let isDRIForAllChangedPathsStill = false;

      // [?] https://docs.github.com/en/rest/reference/pulls#list-pull-requests-files
      let changedPaths = _.pluck(await sails.helpers.http.get(`https://api.github.com/repos/${owner}/${repo}/pulls/${prNumber}/files`, {
        per_page: 100,//eslint-disable-line camelcase
      }, baseHeaders).retry(), 'filename');// (don't worry, it's the whole path, not the filename)

      isDRIForAllChangedPathsStill = _.all(changedPaths, (changedPath)=>{
        changedPath = changedPath.replace(/\/+$/,'');// « trim trailing slashes, just in case (b/c otherwise could loop forever)

        // sails.log.verbose(`…checking DRI of changed path "${changedPath}"`);

        let selfMergers = DRI_BY_PATH[changedPath] ? [].concat(DRI_BY_PATH[changedPath]) : [];// « ensure array
        if (!githubUserToCheck && selfMergers.length >= 1) {// « not checking a user, so just make sure all these paths are preapproved for SOMEONE
          return true;
        }
        if (githubUserToCheck && (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*')))) {
          return true;
        }//•
        let numRemainingPathsToCheck = changedPath.split('/').length;
        while (numRemainingPathsToCheck > 0) {
          let ancestralPath = changedPath.split('/').slice(0, -1 * numRemainingPathsToCheck).join('/');
          // sails.log.verbose(`…checking DRI of ancestral path "${ancestralPath}" for changed path`);
          let selfMergers = DRI_BY_PATH[ancestralPath] ? [].concat(DRI_BY_PATH[ancestralPath]) : [];// « ensure array
          if (!githubUserToCheck && selfMergers.length >= 1) {// « not checking a user, so just make sure all these paths are preapproved for SOMEONE
            return true;
          }
          if (githubUserToCheck && (selfMergers.includes(githubUserToCheck.toLowerCase()) || (isGithubUserMaintainerOrDoesntMatter && selfMergers.includes('*')))) {
            return true;
          }//•
          numRemainingPathsToCheck--;
        }//∞
      });//∞

      if (isDRIForAllChangedPathsStill && changedPaths.length < 100) {
        return true;
      } else {
        return false;
      }
    });

  }


};

