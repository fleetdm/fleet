module.exports = {


  friendlyName: 'Get Bug and PR report',


  description: 'Get information about open bugs and closed pull requests in the fleetdm/fleet GitHub repo.',


  exits: {

  },


  fn: async function () {

    sails.log('Getting average open time for issues with the "bug" label and open pull requests in the fleetdm/fleet Github repo...');

    if(!sails.config.custom.githubAccessToken) {
      throw new Error('Missing GitHub access token! To use this script, a GitHub access token is required. To resolve, add a GitHub access token to your local configuration (website/config/local.js) as sails.config.custom.githubAccessToken or provide one when running this script. (ex: "sails_custom__githubAccessToken=YOUR_PERSONAL_ACCESS_TOKEN sails run get-bug-and-pr-report")');
    }

    let baseHeaders = {
      'User-Agent': 'Fleet average open time',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    const ONE_DAY_IN_MILLISECONDS = (1000 * 60 * 60 * 24);
    const todaysDate = new Date;
    const threeWeeksAgo = new Date(Date.now() - (21 * ONE_DAY_IN_MILLISECONDS));
    const NUMBER_OF_RESULTS_REQUESTED = 100;

    let daysSinceBugsWereOpened = [];
    let commitToMergeTimesInDays = [];

    await sails.helpers.flow.simultaneously([

      //   ██████╗ ██████╗ ███████╗███╗   ██╗    ██████╗ ██╗   ██╗ ██████╗ ███████╗
      //  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ██╔══██╗██║   ██║██╔════╝ ██╔════╝
      //  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██████╔╝██║   ██║██║  ███╗███████╗
      //  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██╔══██╗██║   ██║██║   ██║╚════██║
      //  ╚██████╔╝██║     ███████╗██║ ╚████║    ██████╔╝╚██████╔╝╚██████╔╝███████║
      //   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═════╝  ╚═════╝  ╚═════╝ ╚══════╝
      //
      async()=>{

        let pageNumberForPossiblePaginatedResults = 0;
        let allIssuesWithBugLabel = [];

        // Fetch all open issues in the fleetdm/fleet repo with the bug label.
        // Note: This will send requests to GitHub until the number of results is less than the number we requested.
        await sails.helpers.flow.until(async ()=>{
          // Increment the page of results we're requesting.
          pageNumberForPossiblePaginatedResults += 1;
          let issuesWithBugLabel = await sails.helpers.http.get(
            `https://api.github.com/repos/fleetdm/fleet/issues`,
            {
              'state': 'open',
              'labels': 'bug',
              'per_page': NUMBER_OF_RESULTS_REQUESTED,
              'page': pageNumberForPossiblePaginatedResults,
            },
            baseHeaders
          ).retry();
          // Add the results to the allIssuesWithBugLabel array.
          allIssuesWithBugLabel = allIssuesWithBugLabel.concat(issuesWithBugLabel);
          // If we recieved less results than we requested, we've reached the last page of the results.
          return issuesWithBugLabel.length !== NUMBER_OF_RESULTS_REQUESTED;
        }, 10000);

        // iterate through the allIssuesWithBugLabel array, adding the number
        for(let issue of allIssuesWithBugLabel) {
          // Create a date object from the issue's created_at timestamp.
          let issueOpenedOn = new Date(issue.created_at);
          // Get the amount of time this issue has been open in milliseconds.
          let timeOpenInMS = Math.abs(todaysDate - issueOpenedOn);
          // Convert the miliseconds to days and add the value to the daysSinceBugsWereOpened array
          let timeOpenInDays = timeOpenInMS / ONE_DAY_IN_MILLISECONDS;
          daysSinceBugsWereOpened.push(timeOpenInDays);
        }

      },
      //   ██████╗██╗      ██████╗ ███████╗███████╗██████╗     ██████╗ ██████╗ ███████╗
      //  ██╔════╝██║     ██╔═══██╗██╔════╝██╔════╝██╔══██╗    ██╔══██╗██╔══██╗██╔════╝
      //  ██║     ██║     ██║   ██║███████╗█████╗  ██║  ██║    ██████╔╝██████╔╝███████╗
      //  ██║     ██║     ██║   ██║╚════██║██╔══╝  ██║  ██║    ██╔═══╝ ██╔══██╗╚════██║
      //  ╚██████╗███████╗╚██████╔╝███████║███████╗██████╔╝    ██║     ██║  ██║███████║
      //   ╚═════╝╚══════╝ ╚═════╝ ╚══════╝╚══════╝╚═════╝     ╚═╝     ╚═╝  ╚═╝╚══════╝
      //
      async()=>{

        // Fetch the last 100 closed pull requests in the fleetdm/fleet repo.
        let lastHundredClosedPullRequests = await sails.helpers.http.get(
          `https://api.github.com/repos/fleetdm/fleet/pulls`,
          {
            'state': 'closed',
            'sort': 'updated',
            'direction': 'desc',
            'per_page': NUMBER_OF_RESULTS_REQUESTED,
            'page': 1,
          },
          baseHeaders
        ).retry();

        // Filter the results to get pull requests merged in the past three weeks.
        let pullRequestsMergedInThePastThreeWeeks = lastHundredClosedPullRequests.filter((pullRequest)=>{
          return threeWeeksAgo <= new Date(pullRequest.merged_at);
        });

        // To get the timestamp of the first commit for each pull request, we'll need to send a request to the commits API endpoint.
        await sails.helpers.flow.simultaneouslyForEach(pullRequestsMergedInThePastThreeWeeks, async (pullRequest)=>{
          // Create a date object from the PR's merged_at timestamp.
          let pullRequestMergedOn = new Date(pullRequest.merged_at);

          // https://docs.github.com/en/rest/commits/commits#list-commits
          let commitsOnThisPullRequest = await sails.helpers.http.get(pullRequest.commits_url, {}, baseHeaders).retry();

          // Create a new Date from the timestamp of the first commit on this pull request.
          let firstCommitAt = new Date(commitsOnThisPullRequest[0].commit.author.date); // https://docs.github.com/en/rest/commits/commits#list-commits--code-samples
          // Get the amount of time this issue has been open in milliseconds.
          let timeOpenInMS = Math.abs(pullRequestMergedOn - firstCommitAt);
          // Convert the miliseconds to days and add the value to the daysSincePullRequestsWereOpened array.
          let timeFromFirstCommitInDays = Math.round(timeOpenInMS / ONE_DAY_IN_MILLISECONDS);
          commitToMergeTimesInDays.push(timeFromFirstCommitInDays);
        });

      },
    ]);

    // Get the averages from the arrays of results.
    let averageNumberOfDaysBugsAreOpenFor = Math.round(_.sum(daysSinceBugsWereOpened)/daysSinceBugsWereOpened.length);
    let averageNumberOfDaysFromCommitToMerge = Math.round(_.sum(commitToMergeTimesInDays)/commitToMergeTimesInDays.length);

    // Log the results
    sails.log(`Bugs:
       ------------------------------------
       Number of open issues with the "bug" label: ${daysSinceBugsWereOpened.length}
       Average open time: ${averageNumberOfDaysBugsAreOpenFor} days.
       ------------------------------------

       Pull requests:
       ------------------------------------
       Number of pull requests merged in the past three weeks: ${commitToMergeTimesInDays.length}
       Average time from first commit to merge: ${averageNumberOfDaysFromCommitToMerge} days.
       ------------------------------------`);
  }

};

