module.exports = {


  friendlyName: 'Create issues for todays rituals',


  description: '',


  fn: async function () {

    let path = require('path');
    let YAML = require('yaml');
    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');


    // Load rituals
    let ritualsPath = path.resolve(topLvlRepoPath, 'handbook/company/rituals.yml');
    let rituals = [];
    let ritualsYml = await sails.helpers.fs.read(ritualsPath);
    try {
      rituals = YAML.parse(ritualsYml, { prettyErrors: true });
    } catch (err) {
      throw new Error(`Could not parse the YAMl for rituals at ${ritualsPath} on line ${err.linePos.start.line}. To resolve, make sure the YAML is valid, and try again: ` + err.stack);
    }
    sails.log('rituals', rituals);

    // Validate rituals
    for (let ritual of rituals) {

      let KNOWN_AUTOMATABLE_FREQUENCIES = ['Daily', 'Weekly', 'Triweekly'];//TODO: others
      if (ritual.autoIssue && !KNOWN_AUTOMATABLE_FREQUENCIES.includes(ritual.frequency)) {
        throw new Error(`Invalid ritual: "${ritual.task}" indicates frequency "${ritual.frequency}", but that isn't supported with automations turned on.  Supported frequencies: ${KNOWN_AUTOMATABLE_FREQUENCIES}`);
      }

      // TODO

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
      // TODO

    }//∞


  }


};

