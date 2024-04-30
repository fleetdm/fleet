module.exports = {


  friendlyName: 'View signup',


  description: 'Display "Signup" page.',

  inputs: {
    purchaseLicense: {
      type: 'boolean',
      description: 'If this query string is provided, this user will be taken directly to the /new-license page after they signup.',
      extendedDescription: 'This value will only be present if the user is redirected to this page from the customers/view-new-license',
      defaultsTo: false,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/entrance/signup',
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }

  },


  fn: async function ({purchaseLicense}) {

    if (this.req.me) {
      throw {redirect: '/start'};
    }

    return {
      redirectToLicenseDispenser: purchaseLicense,
    };

  }


};
