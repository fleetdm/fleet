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
        let nextIssueShouldBeCreatedAt;
        let ritualsFrequencyInMs = 0;
        let now = new Date();

        if(_.startsWith(ritual.frequency, 'Daily')){// Using _.startsWith() to handle frequencies with emoji ("Daily ⏰") and with out ("Daily")
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24;
        } else if(ritual.frequency === 'Weekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7;
        } else if(ritual.frequency === 'Biweekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7 * 2;
        } else if(ritual.frequency === 'Triweekly'){
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 7 * 3;
        } else if (ritual.frequency === 'Annually') {
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * 365;
        } else if (ritual.frequency === 'Monthly') {
          // For monthly rituals, we will create issues on the day of the month that the ritual was started on, or the last day of the month if the ritual was started on a day that doesn't exist in the current month
          // (e.g, the next issue for a monthly ritual started on 2024-01-31 would be created for on 2024-02-29)
          let ritualStartedOn = new Date(ritual.startedOn);
          // Get the day in the month that we'll create the issue for this ritual on.
          let dayToCreateIssueOn = ritualStartedOn.getUTCDate();
          // Get the number of days in the current month.
          let numberOfDaysInThisMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getUTCDate();
          if(dayToCreateIssueOn === now.getUTCDate()){
            // If this is the day of the month this ritual was started on, create an issue.
            isItTimeToCreateANewIssue = true;
          } else if(numberOfDaysInThisMonth < dayToCreateIssueOn && numberOfDaysInThisMonth === now.getUTCDate()){
            // If this ritual was started on a date that does not exist in the current month, and this is the last day of the month, create an issue.
            isItTimeToCreateANewIssue = true;
          }
          nextIssueShouldBeCreatedAt = new Date(now.getFullYear(), now.getMonth() + 1, dayToCreateIssueOn);
          ritualsFrequencyInMs = 1000 * 60 * 60 * 24 * numberOfDaysInThisMonth;
        }//ﬁ

        // Determine if we should create an issue for non-monthly rituals.
        if(ritual.frequency === 'Annually') {
          // Create a date of when the ritual started
          let ritualStartedOn = new Date(ritual.startedOn);
          let dayToCreateIssueOn = ritualStartedOn.getUTCDate();
          let monthToCreateIssueOn = ritualStartedOn.getUTCMonth();

          // Check if today's month and day match the ritual's start date
          if (now.getUTCDate() === dayToCreateIssueOn && now.getUTCMonth() === monthToCreateIssueOn) {
            isItTimeToCreateANewIssue = true;
          }
          nextIssueShouldBeCreatedAt = new Date(now.getUTCFullYear() + 1, monthToCreateIssueOn, dayToCreateIssueOn);
        } else if(ritual.frequency !== 'Monthly') {
          // Get a JS timestamp representing 12 PM UTC of the day this script is running.
          let twelveHoursInMs = 1000 * 60 * 60 * 12;
          let lastUTCNoonAt = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 12, 0, 0, 0)).getTime();

          // Get a JS timestamp representing 12:00 PM UTC of the day this ritual started.
          let ritualStartedAt = new Date(ritual.startedOn).getTime() + twelveHoursInMs;
          // Find out how many times this ritual has occurred.
          let howManyRitualsCycles = (lastUTCNoonAt - ritualStartedAt ) / ritualsFrequencyInMs;
          // Find out when the next issue will be created at
          nextIssueShouldBeCreatedAt = ritualStartedAt + ((Math.floor(howManyRitualsCycles) + 1) * ritualsFrequencyInMs);
          // Get the amount of this ritual's cycle remaining.
          let amountOfCycleRemainingTillNextRitual = (Math.floor(howManyRitualsCycles) - howManyRitualsCycles) + 1;
          // Get the number of milliseconds until the next issue for this ritual will be created.
          let timeToNextRitualInMs = amountOfCycleRemainingTillNextRitual * ritualsFrequencyInMs;

          if(_.startsWith(ritual.frequency, 'Daily')) {// Using _.startsWith() to handle frequencies with emoji ("Daily ⏰") and with out ("Daily")
            // Since this script runs once a day, we'll always create issues for daily rituals.
            isItTimeToCreateANewIssue = true;
          } else if(timeToNextRitualInMs === ritualsFrequencyInMs) {
            // For weekly, biweekly, and triweekly frequencies, we'll check to see if the calculated timeToNextRitualInMs is the same as the rituals frequency.
            isItTimeToCreateANewIssue = true;
          }
        }//ﬁ

        // Skip to the next ritual if it isn't time yet.
        if(!isItTimeToCreateANewIssue) {
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

