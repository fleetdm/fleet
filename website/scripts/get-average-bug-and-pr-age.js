module.exports = {


  friendlyName: 'Deliver Fleet GitHub repo statistics',


  description: 'Get the average age of open pull requests and issues with the bug label in the fleetdm/fleet GitHub repo.',


  exits: {


  },


  fn: async function () {

    sails.log('Getting average open time for issues with the "bug" label and open pull requests in the fleetdm/fleet Github repo...');


    let baseHeaders = {
      'User-Agent': 'Fleet average open time',
    };

    const ONE_DAY_IN_MILLISECONDS = (1000 * 60 * 60 * 24);
    const todaysDate = new Date;
    const NUMBER_OF_RESULTS_REQUESTED = 100;


    //   ██████╗ ██████╗ ███████╗███╗   ██╗    ██████╗ ██╗   ██╗ ██████╗ ███████╗
    //  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ██╔══██╗██║   ██║██╔════╝ ██╔════╝
    //  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██████╔╝██║   ██║██║  ███╗███████╗
    //  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██╔══██╗██║   ██║██║   ██║╚════██║
    //  ╚██████╔╝██║     ███████╗██║ ╚████║    ██████╔╝╚██████╔╝╚██████╔╝███████║
    //   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═════╝  ╚═════╝  ╚═════╝ ╚══════╝
    //

    let pageNumberForPossiblePaginatedResults = 0;
    let allIssuesWithBugLabel = [];
    let daysSinceBugsWereOpened = [];

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


    for(let issue of allIssuesWithBugLabel) {
      // Create a date object from the issue's created_at timestamp.
      let issueOpenedOn = new Date(issue.created_at);
      // Get the amount of time this issue has been open in milliseconds.
      let timeOpenInMS = Math.abs(todaysDate - issueOpenedOn);
      // Convert the miliseconds to days and add the value to the daysSinceBugsWereOpened array
      let timeOpenInDaysRoundedDown = Math.floor(timeOpenInMS / ONE_DAY_IN_MILLISECONDS);
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

    let pullRequestResultsPageNumber = 0;
    let allOpenPullRequests = [];
    let daysSincePullRequestsWereOpened = [];

    // Fetch all open pull requests in the fleetdm/fleet repo.
    // Note: This will send requests to GitHub until the number of results is less than the number we requested.
    await sails.helpers.flow.until(async ()=>{
      // Increment the page of results we're requesting.
      pullRequestResultsPageNumber += 1;
      let pullRequests = await sails.helpers.http.get(
        `https://api.github.com/repos/fleetdm/fleet/pulls`,
        {
          'state': 'open',
          'per_page': NUMBER_OF_RESULTS_REQUESTED,
          'page': pullRequestResultsPageNumber,
        },
        baseHeaders
      ).retry();
      // Add the results to the array of results.
      allOpenPullRequests = allOpenPullRequests.concat(pullRequests);
      // If we recieved less results than we requested, we've reached the last page of the results.
      return pullRequests.length !== NUMBER_OF_RESULTS_REQUESTED;
    }, 10000);

    for(let pullRequest of allOpenPullRequests) {
      // Create a date object from the PR's created_at timestamp.
      let pullRequestOpenedOn = new Date(pullRequest.created_at);
      // Get the amount of time this issue has been open in milliseconds.
      let timeOpenInMS = Math.abs(todaysDate - pullRequestOpenedOn);
      // Convert the miliseconds to days and add the value to the daysSincePullRequestsWereOpened array
      let timeOpenInDaysRoundedDown = Math.floor(timeOpenInMS / ONE_DAY_IN_MILLISECONDS);
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

