module.exports = {


  friendlyName: 'Get bug and PR report',


  description: 'Get information about open bugs and pull requests.',


  inputs: {

  },


  fn: async function ({}) {

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
    let daysSinceUnreleasedBugsWereOpened = [];
    let daysSinceReleasedBugsWereOpened = [];
    let allBugsWithUnreleasedLabel = [];
    let allBugsWithReleasedLabel = [];
    let allBugs32DaysOrOlder = [];
    let allBugsCreatedInPastWeek = [];
    let allBugsClosedInPastWeek = [];
    let allBugsReportedByCustomersInPastWeek = [];
    let daysSincePullRequestsWereOpened = [];
    let daysSinceContributorPullRequestsWereOpened = [];
    let commitToMergeTimesInDays = [];

    let allPublicOpenPrs = [];
    let publicPrsMergedInThePastThreeWeeks = [];
    let allNonPublicOpenPrs = [];
    let nonPublicPrsClosedInThePastThreeWeeks = [];



    // Endpoint operations
    let allBugsCreatedInPastWeekEndpointOps = [];
    let allBugsCreatedInPastWeekEndpointOpsUnreleased = [];
    let allBugsCreatedInPastWeekEndpointOpsReleased = [];
    let allBugsCreatedInPastWeekEndpointOpsCustomerImpacting = [];

    // Mobile Device Management
    let allBugsCreatedInPastWeekMobileDeviceManagement = [];

    let allBugsCreatedInPastWeekMobileDeviceManagementUnreleased = [];
    let allBugsCreatedInPastWeekMobileDeviceManagementReleased = [];
    let allBugsCreatedInPastWeekMobileDeviceManagementCustomerImpacting = [];


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
          // If we received less results than we requested, we've reached the last page of the results.
          return issuesWithBugLabel.length !== NUMBER_OF_RESULTS_REQUESTED;
        }, 10000);

        // iterate through the allIssuesWithBugLabel array, adding the number
        for (let issue of allIssuesWithBugLabel) {
          // Create a date object from the issue's created_at timestamp.
          let issueOpenedOn = new Date(issue.created_at);
          // Get the amount of time this issue has been open in milliseconds.
          let timeOpenInMS = Math.abs(todaysDate - issueOpenedOn);
          // Convert the miliseconds to days and add the value to the daysSinceBugsWereOpened array
          let timeOpenInDays = timeOpenInMS / ONE_DAY_IN_MILLISECONDS;
          if (timeOpenInDays >= 32) {
            allBugs32DaysOrOlder.push(issue);
          }
          if (timeOpenInDays <= 7) {
            // All bugs in past week
            allBugsCreatedInPastWeek.push(issue);
            // Customer-reported bugs
            if (issue.labels.some(label => label.name.indexOf('customer-') >= 0)) {
              allBugsReportedByCustomersInPastWeek.push(issue);
            }
            // Get Endpoint Ops KPIs
            if (issue.labels.some(label => label.name === '#g-endpoint-ops')) {
              allBugsCreatedInPastWeekEndpointOps.push(issue);
              if (issue.labels.some(label => label.name === '~unreleased bug')) {
                allBugsCreatedInPastWeekEndpointOpsUnreleased.push(issue);
              }
              else if (issue.labels.some(label => label.name === '~released bug')) {
                allBugsCreatedInPastWeekEndpointOpsReleased.push(issue);
              }
              if (issue.labels.some(label => label.name.indexOf('customer-') >= 0)) {
                allBugsCreatedInPastWeekEndpointOpsCustomerImpacting.push(issue);
              }
            }
            // Get MDM KPIs
            if (issue.labels.some(label => label.name === '#g-mdm')) {
              allBugsCreatedInPastWeekMobileDeviceManagement.push(issue);
              if (issue.labels.some(label => label.name === '~unreleased bug')) {
                allBugsCreatedInPastWeekMobileDeviceManagementUnreleased.push(issue);
              }
              else if (issue.labels.some(label => label.name === '~released bug')) {
                allBugsCreatedInPastWeekMobileDeviceManagementReleased.push(issue);
              }
              if (issue.labels.some(label => label.name.indexOf('customer-') >= 0)) {
                allBugsCreatedInPastWeekMobileDeviceManagementCustomerImpacting.push(issue);
              }
            }
          }

          daysSinceBugsWereOpened.push(timeOpenInDays);
          // Send to released or unreleased bugs array
          if (issue.labels.some(label => label.name === '~unreleased bug')) {
            allBugsWithUnreleasedLabel.push(issue);
            daysSinceUnreleasedBugsWereOpened.push(timeOpenInDays);
          } else if (issue.labels.some(label => label.name === '~released bug')) {
            allBugsWithReleasedLabel.push(issue);
            daysSinceReleasedBugsWereOpened.push(timeOpenInDays);
          } else {
            // If not labeled as a released or unreleased bug, log a warning.
            sails.log.warn('Issue #'+issue.number+' is labeled as a bug but is not labeled as released or unreleased.');
          }

        }

      },

      //   ██████╗██╗      ██████╗ ███████╗███████╗██████╗     ██████╗ ██╗   ██╗ ██████╗ ███████╗
      //  ██╔════╝██║     ██╔═══██╗██╔════╝██╔════╝██╔══██╗    ██╔══██╗██║   ██║██╔════╝ ██╔════╝
      //  ██║     ██║     ██║   ██║███████╗█████╗  ██║  ██║    ██████╔╝██║   ██║██║  ███╗███████╗
      //  ██║     ██║     ██║   ██║╚════██║██╔══╝  ██║  ██║    ██╔══██╗██║   ██║██║   ██║╚════██║
      //  ╚██████╗███████╗╚██████╔╝███████║███████╗██████╔╝    ██████╔╝╚██████╔╝╚██████╔╝███████║
      //   ╚═════╝╚══════╝ ╚═════╝ ╚══════╝╚══════╝╚═════╝     ╚═════╝  ╚═════╝  ╚═════╝ ╚══════╝
      //

      async()=>{

        let pageNumberForPaginatedResults = 0;
        let allIssuesWithBugLabel = [];

        // Fetch all closed issues in the fleetdm/fleet repo with the bug label.
        // Note: This will send requests to GitHub until the number of results is less than the number we requested.
        await sails.helpers.flow.until(async ()=>{
          // Increment the page of results we're requesting.
          pageNumberForPaginatedResults += 1;
          let issuesWithBugLabel = await sails.helpers.http.get(
            `https://api.github.com/repos/fleetdm/fleet/issues`,
            {
              'state': 'closed',
              'labels': 'bug',
              'per_page': NUMBER_OF_RESULTS_REQUESTED,
              'page': pageNumberForPaginatedResults,
            },
            baseHeaders
          ).retry();
          // Add the results to the allIssuesWithBugLabel array.
          allIssuesWithBugLabel = allIssuesWithBugLabel.concat(issuesWithBugLabel);
          // Stop when we've received results from the third page.
          return pageNumberForPaginatedResults === 3;
        }, 10000);

        // iterate through the allIssuesWithBugLabel array, adding the number
        for (let issue of allIssuesWithBugLabel) {
          // Create a date object from the issue's closed_at timestamp.
          let issueClosedOn = new Date(issue.closed_at);
          // Get the amount of time this issue has been closed in milliseconds.
          let timeClosedInMS = Math.abs(todaysDate - issueClosedOn);
          // Convert the miliseconds to days and add the value to the allBugsClosedInPastWeek array
          let timeClosedInDays = timeClosedInMS / ONE_DAY_IN_MILLISECONDS;
          if (timeClosedInDays <= 7) {
            allBugsClosedInPastWeek.push(issue);
          }
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

        let pageNumberForPaginatedResults = 0;

        // Fetch the last 300 closed pull requests from the fleetdm/fleet GitHub Repo
        // [?] https://docs.github.com/en/free-pro-team@latest/rest/pulls/pulls#list-pull-requests
        await sails.helpers.flow.until(async ()=>{
          // Increment the page of results we're requesting.
          pageNumberForPaginatedResults += 1;
          let closedPullRequests = await sails.helpers.http.get(
            `https://api.github.com/repos/fleetdm/fleet/pulls`,
            {
              'state': 'closed',
              'sort': 'updated',
              'direction': 'desc',
              'per_page': NUMBER_OF_RESULTS_REQUESTED,
              'page': pageNumberForPaginatedResults,
            },
            baseHeaders
          ).retry();

          // Exclude draft PRs and filter the PRs we received from Github using the pull request's merged_at date.
          let resultsToAdd = closedPullRequests.filter((pullRequest)=>{
            return !pullRequest.draft && threeWeeksAgo <= new Date(pullRequest.merged_at);
          });

          // Add the filtered array of PRs to the array of all pull requests merged in the past three weeks.
          publicPrsMergedInThePastThreeWeeks = publicPrsMergedInThePastThreeWeeks.concat(resultsToAdd);
          // Stop when we've received results from the third page.
          return pageNumberForPaginatedResults === 3;
        });


        // To get the timestamp of the first commit for each pull request, we'll need to send a request to the commits API endpoint.
        await sails.helpers.flow.simultaneouslyForEach(publicPrsMergedInThePastThreeWeeks, async (pullRequest)=>{
          // Create a date object from the PR's merged_at timestamp.
          let pullRequestMergedOn = new Date(pullRequest.merged_at);

          // Get commits on this PR.
          // [?] https://docs.github.com/en/rest/commits/commits#list-commits
          let commitsOnThisPullRequest = await sails.helpers.http.get(pullRequest.commits_url, {}, baseHeaders).retry();

          // Create a new Date from the timestamp of the first commit on this pull request.
          let firstCommitAt = new Date(commitsOnThisPullRequest[0].commit.author.date); // https://docs.github.com/en/rest/commits/commits#list-commits--code-samples
          // Get the amount of time this issue has been open in milliseconds.
          let timeFromCommitToMergeInMS = pullRequestMergedOn - firstCommitAt;
          // Convert the miliseconds to days and add the value to the daysSincePullRequestsWereOpened array.
          let timeFromFirstCommitInDays = timeFromCommitToMergeInMS / ONE_DAY_IN_MILLISECONDS;
          commitToMergeTimesInDays.push(timeFromFirstCommitInDays);
        });

      },
      //   ██████╗ ██████╗ ███████╗███╗   ██╗    ██████╗ ██████╗ ███████╗
      //  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║    ██╔══██╗██╔══██╗██╔════╝
      //  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║    ██████╔╝██████╔╝███████╗
      //  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║    ██╔═══╝ ██╔══██╗╚════██║
      //  ╚██████╔╝██║     ███████╗██║ ╚████║    ██║     ██║  ██║███████║
      //   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝    ╚═╝     ╚═╝  ╚═╝╚══════╝
      //
      async()=>{
        let pullRequestResultsPageNumber = 0;
        let contributorPullRequests = [];
        // Fetch all open pull requests in the fleetdm/fleet repo.
        // Note: This will send requests to GitHub until the number of results is less than the number we requested.
        // [?] https://docs.github.com/en/free-pro-team@latest/rest/pulls/pulls#list-pull-requests
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
          allPublicOpenPrs = allPublicOpenPrs.concat(pullRequests);
          // If we received less results than we requested, we've reached the last page of the results.
          return pullRequests.length !== NUMBER_OF_RESULTS_REQUESTED;
        }, 10000);

        for(let pullRequest of allPublicOpenPrs) {
          // Create a date object from the PR's created_at timestamp.
          let pullRequestOpenedOn = new Date(pullRequest.created_at);
          // Get the amount of time this issue has been open in milliseconds.
          let timeOpenInMS = Math.abs(todaysDate - pullRequestOpenedOn);
          // Convert the miliseconds to days and add the value to the daysSincePullRequestsWereOpened array
          let timeOpenInDays = timeOpenInMS / ONE_DAY_IN_MILLISECONDS;
          if (!pullRequest.draft) {// Exclude draft PRs
            daysSincePullRequestsWereOpened.push(timeOpenInDays);
          }
          // If not a draft, not a bot, not a PR labeled with #handbook
          // Track as a contributor PR and include in contributor PR KPI
          if (!pullRequest.draft && pullRequest.user.type !== 'Bot' && !pullRequest.labels.some(label => label.name === '#handbook' || label.name === '~ceo' || label.name === ':improve documentation')) {
            daysSinceContributorPullRequestsWereOpened.push(timeOpenInDays);
            contributorPullRequests.push(pullRequest);
          }
        }//∞

      },

      //  ███╗   ██╗ ██████╗ ███╗   ██╗      ██████╗ ██╗   ██╗██████╗ ██╗     ██╗ ██████╗
      //  ████╗  ██║██╔═══██╗████╗  ██║      ██╔══██╗██║   ██║██╔══██╗██║     ██║██╔════╝
      //  ██╔██╗ ██║██║   ██║██╔██╗ ██║█████╗██████╔╝██║   ██║██████╔╝██║     ██║██║
      //  ██║╚██╗██║██║   ██║██║╚██╗██║╚════╝██╔═══╝ ██║   ██║██╔══██╗██║     ██║██║
      //  ██║ ╚████║╚██████╔╝██║ ╚████║      ██║     ╚██████╔╝██████╔╝███████╗██║╚██████╗
      //  ╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═══╝      ╚═╝      ╚═════╝ ╚═════╝ ╚══════╝╚═╝ ╚═════╝
      //
      async()=>{

        // Fetch confidential PRs (current open, and recent closed)
        for (let repoName of ['confidential']) {
          // [?] https://docs.github.com/en/free-pro-team@latest/rest/pulls/pulls#list-pull-requests
          let openPrs = await sails.helpers.http.get(`https://api.github.com/repos/fleetdm/${encodeURIComponent(repoName)}/pulls`, {
            state: 'open',
            'per_page': 100,
            page: 1,
          }, baseHeaders);
          allNonPublicOpenPrs = allNonPublicOpenPrs.concat(openPrs);

          // [?] https://docs.github.com/en/free-pro-team@latest/rest/pulls/pulls#list-pull-requests
          let last100ClosedPrs = await sails.helpers.http.get(`https://api.github.com/repos/fleetdm/${encodeURIComponent(repoName)}/pulls`, {
            state: 'closed',
            sort: 'updated',
            direction: 'desc',
            'per_page': 100,
            page: 1,
          }, baseHeaders);

          // Exclude draft PRs and filter the PRs we received from Github using the pull request's closed_at date.
          nonPublicPrsClosedInThePastThreeWeeks = nonPublicPrsClosedInThePastThreeWeeks.concat(
            last100ClosedPrs.filter((pr)=>{
              return !pr.draft && threeWeeksAgo.getTime() <= (new Date(pr.closed_at)).getTime();
            })
          );
        }//∞
      }

    ]);

    // Get the averages from the arrays of results.
    let averageNumberOfDaysBugsAreOpenFor = Math.round(_.sum(daysSinceBugsWereOpened) / daysSinceBugsWereOpened.length);
    let averageNumberOfDaysUnreleasedBugsAreOpenFor = Math.round(_.sum(daysSinceUnreleasedBugsWereOpened) / daysSinceUnreleasedBugsWereOpened.length);
    let averageNumberOfDaysReleasedBugsAreOpenFor = Math.round(_.sum(daysSinceReleasedBugsWereOpened)/daysSinceReleasedBugsWereOpened.length);
    let averageDaysPullRequestsAreOpenFor = Math.round(_.sum(daysSincePullRequestsWereOpened)/daysSincePullRequestsWereOpened.length);
    let averageDaysContributorPullRequestsAreOpenFor = Math.round(_.sum(daysSinceContributorPullRequestsWereOpened)/daysSinceContributorPullRequestsWereOpened.length);


    // Compute Handbook PR KPIs, which are slightly simpler.
    // FUTURE: Refactor this to be less messy.
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    let handbookOpenPrs = [];
    handbookOpenPrs = handbookOpenPrs.concat(allPublicOpenPrs.filter((pr) => !pr.draft && _.pluck(pr.labels, 'name').includes('#handbook')));
    handbookOpenPrs = handbookOpenPrs.concat(allNonPublicOpenPrs.filter((pr) => !pr.draft && _.pluck(pr.labels, 'name').includes('#handbook')));

    let handbookPrsMergedRecently = [];
    handbookPrsMergedRecently = handbookPrsMergedRecently.concat(publicPrsMergedInThePastThreeWeeks.filter((pr) => !pr.draft && _.pluck(pr.labels, 'name').includes('#handbook')));
    handbookPrsMergedRecently = handbookPrsMergedRecently.concat(nonPublicPrsClosedInThePastThreeWeeks.filter((pr) => !pr.draft && _.pluck(pr.labels, 'name').includes('#handbook')));

    let handbookPrOpenTime = handbookPrsMergedRecently.reduce((avgDaysOpen, pr)=>{
      let openedAt = new Date(pr.created_at).getTime();
      let closedAt = new Date(pr.closed_at).getTime();
      let daysOpen = Math.abs(closedAt - openedAt) / ONE_DAY_IN_MILLISECONDS;
      avgDaysOpen = avgDaysOpen + (daysOpen / handbookPrsMergedRecently.length);
      sails.log.verbose('Processing',pr.head.repo.name,':: #'+pr.number,'open '+daysOpen+' days', 'rolling avg now '+avgDaysOpen);
      return avgDaysOpen;
    }, 0);
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    const kpiResults = [];

    // NOTE: If order of the KPI sheets columns changes, the order values are pushed into this array needs to change, as well.
    kpiResults.push(
      averageDaysContributorPullRequestsAreOpenFor,
      allBugs32DaysOrOlder.length,
      allBugsReportedByCustomersInPastWeek.length,
      averageNumberOfDaysReleasedBugsAreOpenFor,
      averageNumberOfDaysUnreleasedBugsAreOpenFor,
      allBugsCreatedInPastWeek.length,
      allBugsClosedInPastWeek.length,);

    // Log the results
    sails.log(`

    CSV for copy-pasting into KPI spreadsheet:
    ---------------------------
    ${kpiResults.join(',')}

    Note: Copy the values above, then paste into Google KPI sheet and select "Split text to columns" to split the values into separate columns.

    Pull requests:
    ---------------------------
    Average open time (no bots, no handbook, no ceo): ${averageDaysContributorPullRequestsAreOpenFor} days.

    Number of open pull requests in the fleetdm/fleet Github repo (no bots, no handbook, no ceo): ${daysSinceContributorPullRequestsWereOpened.length}

    Average open time (all PRs): ${averageDaysPullRequestsAreOpenFor} days.

    Number of open pull requests in the fleetdm/fleet Github repo: ${daysSincePullRequestsWereOpened.length}

    Bugs:
    ---------------------------
    Average open time (released bugs): ${averageNumberOfDaysReleasedBugsAreOpenFor} days.

    Average open time (unreleased bugs): ${averageNumberOfDaysUnreleasedBugsAreOpenFor} days.

    Number of issues with the "bug" label closed in the past week: ${allBugsClosedInPastWeek.length}

    Average open time (all bugs): ${averageNumberOfDaysBugsAreOpenFor} days.

    Number of issues with the "bug" label opened in the past week: ${allBugsCreatedInPastWeek.length}

    Number of open issues with the "bug" label in fleetdm/fleet: ${daysSinceBugsWereOpened.length}

    Number of open issues with the "~released bug" label in fleetdm/fleet: ${allBugsWithReleasedLabel.length}

    Number of open issues with the "~unreleased bug" label in fleetdm/fleet: ${allBugsWithUnreleasedLabel.length}

    Endpoint Operations:
    ---------------------------
    Number of issues with the "#g-endpoint-ops" and "bug" labels opened in the past week: ${allBugsCreatedInPastWeekEndpointOps.length}

    Number of issues with the "#g-endpoint-ops", "bug", and "customer-" labels opened in the past week: ${allBugsCreatedInPastWeekEndpointOpsCustomerImpacting.length}

    Number of issues with the "#g-endpoint-ops", "bug", and "~released bug" labels opened in the past week: ${allBugsCreatedInPastWeekEndpointOpsReleased.length}

    Number of issues with the "#g-endpoint-ops", "bug", and "~unreleased bug" labels opened in the past week: ${allBugsCreatedInPastWeekEndpointOpsUnreleased.length}

    MDM:
    ---------------------------
    Number of issues with the "#g-mdm" and "bug" labels opened in the past week: ${allBugsCreatedInPastWeekMobileDeviceManagement.length}

    Number of issues with the "#g-mdm", "bug", and "customer-" labels opened in the past week: ${allBugsCreatedInPastWeekMobileDeviceManagementCustomerImpacting.length}

    Number of issues with the "#g-mdm", "bug", and "~released bug" labels opened in the past week: ${allBugsCreatedInPastWeekMobileDeviceManagementReleased.length}

    Number of issues with the "#g-mdm", "bug", and "~unreleased bug" labels opened in the past week: ${allBugsCreatedInPastWeekMobileDeviceManagementUnreleased.length}

    Handbook Pull requests
    ---------------------------------------
    Number of open #handbook pull requests in the fleetdm Github org: ${handbookOpenPrs.length}

    Average open time (#handbook PRs): ${Math.round(handbookPrOpenTime*100)/100} days.
    `);

  }

};

