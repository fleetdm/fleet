module.exports = {


  friendlyName: 'View terms',


  description: 'Display "Legal terms" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/legal/terms'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }
    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { title: '📜 Fleet subscription terms' });

    if(!thisPage) {
      throw new Error(`When a user visited the /legal/terms page, the "📜 Fleet subscription terms" markdown file was not found in the website's builtStaticContent.markdownPages configuration.`);
    }
    return {
      path: require('path'),
      thisPage,
    };

  }


};
