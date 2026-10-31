module.exports = {


  friendlyName: 'View hire',


  description: 'Display "Hire" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/hire'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.jobBoard) || !_.isArray(sails.config.builtStaticContent.openPositions)) {
      throw {badConfig: 'builtStaticContent.jobBoard'};
    }

    // Also list Fleet's own open roles that job board candidates are a good fit for.
    let FLEET_JOB_TITLES_TO_INCLUDE = ['Solutions Consultant', 'Solutions Specialist', 'Customer Support Engineer', 'Customer Solutions Architect'];
    let fleetJobs = sails.config.builtStaticContent.openPositions
    .filter((openPosition)=>{
      return _.any(FLEET_JOB_TITLES_TO_INCLUDE, (title) => openPosition.jobTitle.includes(title));
    })
    .map((openPosition) => ({
      // Strip the emoji prefix used in open-positions.yml.
      jobTitle: openPosition.jobTitle.replace(/^[^A-Za-z]+/, ''),
      company: 'Fleet',
      url: openPosition.url,
    }));

    return {
      jobs: [...sails.config.builtStaticContent.jobBoard, ...fleetJobs],
      jobBoardRoles: sails.config.custom.jobBoardRoles,
    };

  }


};
