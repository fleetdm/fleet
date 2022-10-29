module.exports = {


  friendlyName: 'Deliver Fleet GitHub repo statistics',


  description: 'Get the average age of open pull requests and issues with the bug label in the fleetdm/fleet GitHub repo.',


  exits: {


  },


  fn: async function () {

    sails.log('Getting average open time for issues with the "bug" label and open pull requests in the fleetdm/fleet Github repo...');

    if (!sails.config.custom.githubAccessToken) {
      throw new Error('No GitHub access token configured!  (Please set `sails.config.custom.githubAccessToken`.)');
    }//•

    let baseHeaders = {
      'User-Agent': 'fleet test',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    const oneDayInMs = (1000 * 60 * 60 * 24);
    const todaysDate = new Date;


    //   ██████╗ ██████╗ ███████╗███╗   ██╗    ██████╗ ██╗   ██╗ ██████╗ ███████╗
    //  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ██╔══██╗██║   ██║██╔════╝ ██╔════╝
    //  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██████╔╝██║   ██║██║  ███╗███████╗
    //  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██╔══██╗██║   ██║██║   ██║╚════██║
    //  ╚██████╔╝██║     ███████╗██║ ╚████║    ██████╔╝╚██████╔╝╚██████╔╝███████║
    //   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═════╝  ╚═════╝  ╚═════╝ ╚══════╝
    //


    let allIssuesWithBugLabel = [];
    let pageNumberForPossiblePaginatedResults = 1;
    let maximumResultsPerRequest = 100;
    let daysSinceBugsWereOpened = [];

    // Fetch all open issues in the fleetdm/fleet repo with the bug label.
    let issuesWithBugLabel = await sails.helpers.http.get(
      `https://api.github.com/repos/fleetdm/fleet/issues`,
      {
        'state': 'open',
        'labels': 'bug',
        'per_page': maximumResultsPerRequest,
        'page': pageNumberForPossiblePaginatedResults,
      },
      baseHeaders
    ).retry();

    allIssuesWithBugLabel = allIssuesWithBugLabel.concat(issuesWithBugLabel);
    // Set a flag that will be true if the number of results we recieved is the same amount of results that are sent at one time, if this is true, we will request the next set of results until this value is false.
    let maybeNeedToCheckForMoreBugs = (issuesWithBugLabel.length === maximumResultsPerRequest);

    while(maybeNeedToCheckForMoreBugs) {
      pageNumberForPossiblePaginatedResults += 1;
      let aditionalResultsToCheck = await sails.helpers.http.get(
        `https://api.github.com/repos/fleetdm/fleet/issues`,
        {
          'state': 'open',
          'labels': 'bug',
          'per_page': maximumResultsPerRequest,
          'page': pageNumberForPossiblePaginatedResults,
        },
        baseHeaders
      ).retry();
      // Add these issues to the allIssuesWithBugLabel array.
      allIssuesWithBugLabel = allIssuesWithBugLabel.concat(aditionalResultsToCheck);
      // Update the resultsMayHaveMultiplePages to use the amount of the last set of results we recieved.
      maybeNeedToCheckForMoreBugs = (aditionalResultsToCheck.length === maximumResultsPerRequest);
    }


    for(let issue of allIssuesWithBugLabel) {
      // Create a date object from the issue's created_at timestamp.
      let issueOpenedOn = new Date(issue.created_at);
      // Get the amount of time this issue has been open in milliseconds.
      let timeOpenInMS = Math.abs(todaysDate - issueOpenedOn);
      // Convert the miliseconds to days and add the value to the daysSinceBugsWereOpened array
      let timeOpenInDaysRoundedDown = Math.floor(timeOpenInMS/oneDayInMs);
      daysSinceBugsWereOpened.push(timeOpenInDaysRoundedDown);
    }

    // Get the average open time for bugs.
    let averageDaysBugsAreOpenFor = Math.floor(_.sum(daysSinceBugsWereOpened)/daysSinceBugsWereOpened.length);




    //   ██████╗ ██████╗ ███████╗███╗   ██╗    ██████╗ ██████╗ ███████╗
    //  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ██╔══██╗██╔══██╗██╔════╝
    //  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██████╔╝██████╔╝███████╗
    //  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██╔═══╝ ██╔══██╗╚════██║
    //  ╚██████╔╝██║     ███████╗██║ ╚████║    ██║     ██║  ██║███████║
    //   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═╝     ╚═╝  ╚═╝╚══════╝
    //

    let pullRequestResultsPageNumber = 1;
    let allOpenPullRequests = [];
    let daysSincePullRequestsWereOpened = [];

    allOpenPullRequests = await sails.helpers.http.get(
      `https://api.github.com/repos/fleetdm/fleet/pulls`,
      {
        'state': 'open',
        'per_page': 100,
        'page': pullRequestResultsPageNumber,
      },
      baseHeaders
    ).retry();

    let maybeNeedToCheckForMorePullRequests = (allOpenPullRequests.length === 100);

    // Request more results until the number of results returned is less than the amount requested.
    while(maybeNeedToCheckForMorePullRequests) {
      pullRequestResultsPageNumber += 1;
      // Grab the next page of results
      let moreResultsToCheck = await sails.helpers.http.get(
        `https://api.github.com/repos/fleetdm/fleet/pulls`,
        {
          'state': 'open',
          'per_page': 100,
          'page': pullRequestResultsPageNumber,
        },
        baseHeaders
      ).retry();
      // Update the maybeNeedToCheckForMorePullRequests variable to use the amount of results from the last request.
      maybeNeedToCheckForMorePullRequests = (moreResultsToCheck.length === 100);
      // Add the results to the array of results.
      allOpenPullRequests.concat(moreResultsToCheck);
    }


    for(let pullRequest of allOpenPullRequests) {
      // Create a date object from the PR's created_at timestamp.
      let pullRequestOpenedOn = new Date(pullRequest.created_at);
      // Get the amount of time this issue has been open in milliseconds.
      let timeOpenInMS = Math.abs(todaysDate - pullRequestOpenedOn);
      // Convert the miliseconds to days and add the value to the daysSincePullRequestsWereOpened array
      let timeOpenInDaysRoundedDown = Math.floor(timeOpenInMS/oneDayInMs);
      daysSincePullRequestsWereOpened.push(timeOpenInDaysRoundedDown);
    }

    let averageDaysPullRequestsAreOpenFor = Math.floor(_.sum(daysSincePullRequestsWereOpened)/daysSincePullRequestsWereOpened.length);



    // Log the results
    sails.log(`Bugs:
       ------------------------------------
       Number of open issues with the "bug" label: ${daysSinceBugsWereOpened.length}
       Average open time: ${averageDaysBugsAreOpenFor} days.
       ------------------------------------

       Pull requests:
       ------------------------------------
       Number of open pull requests: ${daysSincePullRequestsWereOpened.length}
       Average open time: ${averageDaysPullRequestsAreOpenFor} days.
       ------------------------------------`);
  }


};

