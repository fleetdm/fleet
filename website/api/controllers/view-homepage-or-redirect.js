module.exports = {


  friendlyName: 'View homepage or redirect',


  description: 'Display or redirect to the appropriate homepage, depending on login status.',


  exits: {

    success: {
      statusCode: 200,
      description: 'Requesting user is a guest, so show the public landing page.',
      viewTemplatePath: 'pages/homepage'
    },

    redirect: {
      responseType: 'redirect',
      description: 'Requesting user is logged in, so redirect to the internal welcome page.'
    },

  },


  fn: async function () {

    return {
      primaryBuyingSituation: this.req.session.primaryBuyingSituation || undefined // if set in the session (e.g. from an ad) use the primary buying situation to personalize that sweet, sweet homepage
    };

  }


};
