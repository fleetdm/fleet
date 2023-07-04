module.exports = {


  friendlyName: 'View osquery table details',


  description: 'Display "Osquery table details" page.',

  inputs: {
    tableName : {
      description: 'The slug of the osquery table that this user wants to display',
      example: 'account_policy_data',
      type: 'string',
      required: true,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/osquery-table-details'
    },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },

  },


  fn: async function ({tableName}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    let tableToDisplay = _.find(sails.config.builtStaticContent.markdownPages, { url: '/tables/' + tableName });

    if (!tableToDisplay) {// If there's no EXACTLY matching content page, throw a 404.
      throw 'notFound';
    }

    let pageTitleForMeta = '"'+tableToDisplay.title +'" in osquery | Fleet documentation';
    let pageDescriptionForMeta = 'Read about how to use the "'+tableToDisplay.title+'" table with osquery and Fleet.';


    let allTables = sails.config.builtStaticContent.markdownPages.filter((page)=>{
      return !! _.startsWith(page.url, '/tables/');
    });
    // Respond with view.
    return {
      path: require('path'),
      allTables,
      tableToDisplay,
      pageTitleForMeta,
      pageDescriptionForMeta,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};
