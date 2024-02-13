module.exports = {
  friendlyName: 'Get milestone commit hashes',
  description: 'Generate a list of commit hashes in order for a given milestone.',
  fn: async function ({}) {
    if (!sails.config.custom.githubAccessToken) {
      throw new Error('Missing GitHub access token! To use this script, a GitHub access token is required. To resolve, add a GitHub access token to your local configuration (website/config/local.js) as sails.config.custom.githubAccessToken or provide one when running this script. (ex: "sails_custom__githubAccessToken=YOUR_PERSONAL_ACCESS_TOKEN sails run get-milestone-commit-hashes")');
    }

    let baseHeaders = {
      'User-Agent': 'Fleet-Scripts',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    const MILESTONE = '4.44.1';
    const NUMBER_OF_RESULTS_REQUESTED = 100;
    let pageNumberForPossiblePaginatedResults = 0;
    let allPullRequests = [];

    await sails.helpers.flow.until(async ()=>{
      pageNumberForPossiblePaginatedResults++;
      let pullRequests = await sails.helpers.http.get(
            `https://api.github.com/repos/fleetdm/fleet/pulls`,
            {
              'state': 'closed',
              'sort': 'updated',
              'direction': 'desc',
              'per_page': NUMBER_OF_RESULTS_REQUESTED,
              'page': pageNumberForPossiblePaginatedResults,
            },
            baseHeaders
      ).retry();

      allPullRequests = allPullRequests.concat(pullRequests);

      console.log(`Retrieved page ${pageNumberForPossiblePaginatedResults} results`);

      // Determine if we should continue fetching more pull requests
      return pageNumberForPossiblePaginatedResults === 5;
    });

    const cherryPickCommand = allPullRequests
      .filter(pullRequest => pullRequest.milestone && pullRequest.milestone.title === MILESTONE)
      .sort((a, b) => new Date(a.merged_at) - new Date(b.merged_at))
      .map(pullRequest => pullRequest.merge_commit_sha)
      .join(' ');

    console.log(`git cherry-pick ${cherryPickCommand}`);

  }
};
