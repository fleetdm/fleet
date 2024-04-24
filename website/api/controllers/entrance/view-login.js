module.exports = {


  friendlyName: 'View login',


  description: 'Display "Login" page.',

  inputs: {
    purchaseLicense: {
      type: 'boolean',
      description: 'If this query string is provided, this user will be taken directly to the /new-license page after they login.',
      extendedDescription: 'This value is only present when a user navigates to this page from the /register page if they were redirected to that page from the /new-license page.',
      defaultsTo: false,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/entrance/login',
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }

  },


  fn: async function ({purchaseLicense}) {

    if (this.req.me) {
      if(this.req.me.isSuperAdmin){
        throw {redirect: '/admin/generate-license'};
      } else {
        throw {redirect: '/start'};
      }
    }

    return {
      redirectToLicenseDispenser: purchaseLicense,
    };

  }


};
