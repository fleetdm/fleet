module.exports = {


  friendlyName: 'Create issues for todays rituals',


  description: '',


  inputs: {
    dry: { type: 'boolean', defaultsTo: false, description: 'Whether to do a dry run instead.' },
  },

  fn: async function ({ dry }) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isObject(sails.config.builtStaticContent.rituals)) {
      throw new Error('Missing, incomplete, or invalid configuration. Could not create issues for todays rituals, please try running `sails run build-static-content` and try running this script again.');
    }

    let baseHeaders = {// (for github api)
      'User-Agent': 'Fleetie pie',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`
    };

    for (let ritualSource in sails.config.builtStaticContent.rituals) {
      let rituals = sails.config.builtStaticContent.rituals[ritualSource];
      for (let ritual of rituals) {
        // For each ritual, we'll:
        //  - Convert the ritual's frequency into milliseconds.
        //  - Find out when we will be creating the next issue for the ritual.
        //  - Create an issue for the ritual if the ritual takes place in the next 24 hours.

        if (!ritual.autoIssue) {// « Skip to the next ritual if automations aren't enabled.
          continue;
        }
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
        } else if (ritual.frequency === 'Monthly') {
          // For monthly rituals, we will get the number of days in the previous month, and create a timestamp of the next time this ritual should be run.
          let todaysDate = new Date();
          let numberOfDaysInLastMonth = new Date(todaysDate.getFullYear(), todaysDate.getMonth(), 0).getDate();
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * numberOfDaysInLastMonth;
        }
        // Get a JS timestamp representing 12 PM UTC of the day this script is running.
        let twelveHoursInMs = 1000 * 60 * 60 * 12;
        let today = new Date();
        let lastUTCNoonAt = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate(), 12, 0, 0, 0)).getTime();

        // Get a JS timestamp representing 12:00 PM UTC of the day this ritual started.
        let ritualStartedAt = new Date(ritual.startedOn).getTime() + twelveHoursInMs;
        // Find out how many times this ritual has occurred.
        let howManyRitualsCycles = (lastUTCNoonAt - ritualStartedAt ) / ritualsFrequencyInMs;
        // Find out when the next issue will be created at
        let nextIssueShouldBeCreatedAt = ritualStartedAt + ((Math.floor(howManyRitualsCycles) + 1) * ritualsFrequencyInMs);
        // Get the amount of this ritual's cycle remaining.
        let amountOfCycleRemainingTillNextRitual = (Math.floor(howManyRitualsCycles) - howManyRitualsCycles) + 1;
        // Get the number of milliseconds until the next issue for this ritual will be created.
        let timeToNextRitualInMs = amountOfCycleRemainingTillNextRitual * ritualsFrequencyInMs;
        if(_.startsWith(ritual.frequency, 'Daily')) {// Using _.startsWith() to handle frequencies with emoji ("Daily ⏰") and with out ("Daily")
          // Since this script runs once a day, we'll always create issues for daily rituals.
          isItTimeToCreateANewIssue = true;
        } else if(timeToNextRitualInMs === ritualsFrequencyInMs) {
          // For any other frequency, we'll check to see if the calculated timeToNextRitualInMs is the same as the rituals frequency.
          isItTimeToCreateANewIssue = true;
        }
        // Skip to the next ritual if it isn't time yet.
        if (!isItTimeToCreateANewIssue) {
          sails.log.verbose(`Next issue for ${ritual.task} (${ritual.autoIssue.labels.join(',')}) will be created on ${new Date(nextIssueShouldBeCreatedAt)} (Started on: ${ritual.startedOn}, frequency: ${ritual.frequency})`);
          continue;
        }

        // Create an issue with right labels and assignee, in the right repo.
        if (!dry) {
          let owner = 'fleetdm';
          // [?] https://docs.github.com/en/rest/issues/issues#create-an-issue
          await sails.helpers.http.post(`https://api.github.com/repos/${owner}/${ritual.autoIssue.repo}/issues`, {
            title: ritual.task,
            body: ritual.description + (ritual.moreInfoUrl ? ('\n\n> Read more at '+ritual.moreInfoUrl) : ''),
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

