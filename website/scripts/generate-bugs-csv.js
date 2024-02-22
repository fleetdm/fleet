module.exports = {


  friendlyName: 'Genertae bugs CSV',


  description: 'Generate a categorized bugs CSV.',


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
    const NUMBER_OF_RESULTS_REQUESTED = 100;
    let issueCsv;

    await sails.helpers.flow.simultaneously([
      async()=>{

        let pageNumberForPossiblePaginatedResults = 0;
        let allIssuesWithLabels = [];
        let allIssuesObject = {};

        // Fetch all open issues in the fleetdm/fleet repo with the provided labels.
        // Note: This will send requests to GitHub until the number of results is less than the number we requested.
        await sails.helpers.flow.until(async ()=>{
          // Increment the page of results we're requesting.
          pageNumberForPossiblePaginatedResults += 1;
          let issuesWithLabels = await sails.helpers.http.get(
            `https://api.github.com/repos/fleetdm/fleet/issues`,
            {
              'state': 'open',
              'labels': 'bug',
              'per_page': NUMBER_OF_RESULTS_REQUESTED,
              'page': pageNumberForPossiblePaginatedResults,
            },
            baseHeaders
          ).retry();
          allIssuesWithLabels = allIssuesWithLabels.concat(issuesWithLabels);
          // If we received less results than we requested, we've reached the last page of the results.
          return issuesWithLabels.length !== NUMBER_OF_RESULTS_REQUESTED;
        }, 10000);

        // iterate through the allIssuesWithLabels array
        for (let issue of allIssuesWithLabels) {
          // Look for bugs with the bug- prefix and sort them into the allIssuesObject
          for (let label of issue.labels) {
            if (label.name.startsWith('bug-')) {
              if (!allIssuesObject[label.name]) {
                allIssuesObject[label.name] = [];
              }
              allIssuesObject[label.name].push(issue);
            }
          }
        }

        let issueCsvTitles = 'Bug category,Number of open issues,Average open time\n';
        let issueCsvBody = '';

        for (let category in allIssuesObject) {
          let totalOpenTime = 0;
          let totalOpenIssues = 0;
          for (let issue of allIssuesObject[category]) {
            totalOpenIssues += 1;
            let issueOpenTime = todaysDate - new Date(issue.created_at);
            totalOpenTime += issueOpenTime;
          }
          let averageOpenTime = Math.round((totalOpenTime / totalOpenIssues) / ONE_DAY_IN_MILLISECONDS);
          issueCsvBody += `${category},${totalOpenIssues},${averageOpenTime}\n`;
        }

        issueCsv = issueCsvTitles + issueCsvBody;

      },

    ]);

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // Log the results
    sails.log(`
CSV:
---------------------------
${issueCsv}
    `);

  }

};

