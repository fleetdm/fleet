module.exports = {


  friendlyName: 'View login',


  description: 'Display "Login" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/entrance/login',
    },

    redirect: {
      description: 'The requesting user is already logged in.',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    if (this.req.me) {
      if(this.req.me.hasBillingCard){
        throw {redirect: '/customers/new-license'};
      } else {
        throw {redirect: '/try-fleet/sandbox'};
      }
    }

    return {};

  }


};
