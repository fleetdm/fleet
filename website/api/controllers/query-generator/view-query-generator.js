module.exports = {


  friendlyName: 'View query generator',


  description: 'Display "Query generator" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/query-generator'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.schemaTables)) {
      throw {badConfig: 'builtStaticContent.schemaTables'};
    }
    // Respond with view.
    return {};

  }


};
