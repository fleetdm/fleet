module.exports = {


  friendlyName: 'View forgot password',


  description: 'Display "Forgot password" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/entrance/forgot-password',
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      extendedDescription: 'Logged-in users should change their password in "Account settings."',
      responseType: 'redirect',
    }

  },


  fn: async function () {

    let redirectToSandbox = false;
    if(this.req.url = '/try-fleet/forgot-password'){
      redirectToSandbox = true;
    }
    if (this.req.me) {
      throw {redirect: '/'};
    }

    return {
      redirectToSandbox
    };

  }


};
