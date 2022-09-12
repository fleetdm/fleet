module.exports = {


  friendlyName: 'View osquery tables',


  description: 'Display "Osquery tables" page.',

  urlWildcardSuffix: 'selectedTableSlug',

  inputs: {
    selectedTableSlug : {
      description: 'The name of the osquery table that this user wants to display',
      example: 'account_policy_data',
      type: 'string',
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/osquery-tables'
    },
    badConfig: { responseType: 'badConfig' },
    notFound: { responseType: 'notFound' },

  },


  fn: async function ({selectedTableSlug}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.schemaTables) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.schemaTables'};
    }
    let tableToDisplay = _.find(sails.config.builtStaticContent.schemaTables, { url: '/tables/' + selectedTableSlug });

    if (!tableToDisplay) {// If there's no EXACTLY matching content page, throw a 404.
      throw 'notFound';
    }

    let pageTitleForMeta = tableToDisplay.title + ' table | Fleet schema';
    let pageDescriptionForMeta = 'View information about the '+tableToDisplay.title+' table on Fleets Schema tables';


    let allTables = sails.config.builtStaticContent.schemaTables.filter((page)=>{
      return !! _.startsWith(page.url, '/tables/');
    });
    // Respond with view.
    return {
      path: require('path'),
      allTables,
      compiledPagePartialsAppPath: sails.config.builtStaticContent.compiledPagePartialsAppPath +'/tables',
      tableToDisplay,
      pageTitleForMeta,
      pageDescriptionForMeta,
    };

  }


};
