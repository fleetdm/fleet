module.exports = {


  friendlyName: 'View privacy',


  description: 'Display "Privacy policy" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/legal/privacy'
    },

    badConfig: {
      responseType: 'badConfig'
    },


  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { title: '📜 Fleet privacy policy' });

    if(!thisPage) {
      throw new Error(`When a user visited the /legal/privacy page, the "📜 Fleet privacy policy" markdown file was not found in the website's builtStaticContent.markdownPages configuration.`);
    }
    // All done.
    return {
      path: require('path'),
      thisPage,
    };

  }


};
