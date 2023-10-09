module.exports = {


  friendlyName: 'Create issues for todays rituals',


  description: '',


  inputs: {
    dry: { type: 'boolean', defaultsTo: false, description: 'Whether to do a dry run instead.' },
  },


  fn: async function ({ dry }) {

    let path = require('path');
    let YAML = require('yaml');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');
    let baseHeaders = {// (for github api)
      'User-Agent': 'Fleetie pie',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    let owner = 'fleetdm';
    let repo = 'fleet';// TODO: support confidential and classified also

    // Find all the files in the top level /handbook folder and it's sub-folders
    let FILES_IN_HANDBOOK_FOLDER = await sails.helpers.fs.ls.with({
      dir: path.join(topLvlRepoPath, '/handbook'),
      depth: 3
    });
    // Filter the list of filenames to get the rituals YAML files.
    let ritualYamlPaths = FILES_IN_HANDBOOK_FOLDER.filter((filePath)=>{
      return _.endsWith(filePath, 'rituals.yml');
    });
    let pageNumberForPossiblePaginatedResults = 0;
    let currentIssuesInGithubRepo = [];
    const NUMBER_OF_RESULTS_REQUESTED = 100;
    // Fetch all open issues in the fleetdm/fleet repo.
    // Note: This will send requests to GitHub until the number of results is less than the number we requested.
    await sails.helpers.flow.until(async ()=>{
    let githubIssues = await sails.helpers.http.get(
        `https://api.github.com/repos/${owner}/${repo}/issues`,
        {
          'per_page': NUMBER_OF_RESULTS_REQUESTED,
          'page': pageNumberForPossiblePaginatedResults,
        },
        baseHeaders
      ).retry();
      currentIssuesInGithubRepo = currentIssuesInGithubRepo.concat(githubIssues);
      // If we received less results than we requested, we've reached the last page of the results.
      return currentIssuesInGithubRepo.length !== NUMBER_OF_RESULTS_REQUESTED;
    }, 10000);


    for (let ritualSource of ritualYamlPaths) {

      // Load rituals
      let pathToRituals = path.resolve(topLvlRepoPath, ritualSource);
      let rituals = [];
      let ritualsYml = await sails.helpers.fs.read(pathToRituals);
      try {
        rituals = YAML.parse(ritualsYml, { prettyErrors: true });
      } catch (err) {
        throw new Error(`Could not parse the YAMl for rituals at ${pathToRituals} on line ${err.linePos.start.line}. To resolve, make sure the YAML is valid, and try again: ` + err.stack);
      }



      for (let ritual of rituals) {

        if (!ritual.autoIssue) {// « Skip to the next ritual if automations aren't enabled.
          continue;
        }
        // An issue should be created for a ritual if:
        // - ritual.autoissue contains a labels array (an array of strings of the GH labels that will be put on the issue)
        // - the last issue created for a ritual is older than the rituals frequency.
        // - No issue has been created for this ritual.
        // - TODO: anything else?

        // An issue should not be created if:
        // - It has been less than the frequency since the last issue was created
        // -

        let isItTimeToCreateANewIssue = false;// Default this value to false.


        let ritualsFrequencyInMs = 0;

        if(_.startsWith(ritual.frequency, 'Daily')){// Using _.startsWith() to handle frequencies with emoji ("Daily ⏰") and with out ("Daily")
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24;
        } else if(ritual.frequency === 'Weekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7;
        } else if(ritual.frequency === 'Biweekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7 * 2;
        } else if(ritual.frequency === 'Triweekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7 * 3;
        }// TODO: Monthly

        let previousIssuesCreatedForThisRitual = _.filter(currentIssuesInGithubRepo, {title: ritual.task});

        // If we found issues previously created of this ritual, we'll check the created at timestamp of the most recent one.
        if(previousIssuesCreatedForThisRitual.length > 0){
          // Sort the previous issues for the ritual by their issue number, and get the last (most recent) item in the array.
          let lastIssueCreatedForThisRitual = _.sortBy(previousIssuesCreatedForThisRitual, 'number')[previousIssuesCreatedForThisRitual.length - 1];
          // Create a JS timestamp of when the last issue for this ritual was created.
          let lastIssueWasCreatedAt = Date.parse(lastIssueCreatedForThisRitual.created_at);
          // An a JS timestamp of when the next issue for this ritual should be created.
          let nextIssueShouldBeCreatedAt = (lastIssueWasCreatedAt + ritualsFrequencyInMs);
          // Set the flag to be true if it the issue's created at timestamp + the rituals frequency is after an hour ago.
          isItTimeToCreateANewIssue = (Date.now() - (1000 * 60 * 60)) >= nextIssueShouldBeCreatedAt;
        } else {
          // If no GH issue exists that matches the "task" of the ritual, well set the isItTimeToCreateANewIssue flag to be true.
          isItTimeToCreateANewIssue = true;
        }

        // Skip to the next ritual if it isn't time yet.
        if (isItTimeToCreateANewIssue) {
          continue;
        }

        // Create an issue with right labels and assignee, in the right repo.
        if (!dry) {
          // [?] https://docs.github.com/en/rest/issues/issues#create-an-issue
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${repo}/issues`, {
            title: ritual.task,
            body: ritual.description,
            labels: ritual.autoIssue.labels,
            assignees: [ ritual.dri ]
          }, baseHeaders);
        } else {
          sails.log('Dry run: Would have created an issue for ritual:', ritual);
        }

      }//∞
    }//∞

  }


};

