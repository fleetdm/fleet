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

    let ritualSources = [
      { path: 'handbook/company/rituals.yml', },
    ];
    for (let ritualSource of ritualSources) {

      // Load rituals
      let pathToRituals = path.resolve(topLvlRepoPath, ritualSource.path);
      let rituals = [];
      let ritualsYml = await sails.helpers.fs.read(pathToRituals);
      try {
        rituals = YAML.parse(ritualsYml, { prettyErrors: true });
      } catch (err) {
        throw new Error(`Could not parse the YAMl for rituals at ${pathToRituals} on line ${err.linePos.start.line}. To resolve, make sure the YAML is valid, and try again: ` + err.stack);
      }


      // Validate rituals
      for (let ritual of rituals) {

        let KNOWN_AUTOMATABLE_FREQUENCIES = ['Daily', 'Weekly', 'Triweekly'];//TODO: others
        if (ritual.autoIssue && !KNOWN_AUTOMATABLE_FREQUENCIES.includes(ritual.frequency)) {
          throw new Error(`Invalid ritual: "${ritual.task}" indicates frequency "${ritual.frequency}", but that isn't supported with automations turned on.  Supported frequencies: ${KNOWN_AUTOMATABLE_FREQUENCIES}`);
        }

        // TODO: validate task
        // TODO: validate description
        // TODO: validate DRI (github username)
        // TODO: validate ritual.autoIssue
        // TODO: validate ritual.autoIssue.labels

        // TODO: other validations

      }//∞

      for (let ritual of rituals) {

        if (!ritual.autoIssue) {// « Skip to the next ritual if automations aren't enabled.
          continue;
        }

        // Skip to the next ritual if it isn't time yet.
        if (false) {// TODO
          continue;
        }

        // Create an issue with right labels and assignee, in the right repo.
        if (!dry) {
          let owner = 'fleetdm';
          let repo = 'fleet';// TODO: support confidential and classified also
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

