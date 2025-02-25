module.exports = {


  friendlyName: 'View transparency',


  description: 'Display "Transparency" page.',

  exits: {

    success: {
      viewTemplatePath: 'pages/transparency'
    }

  },


  fn: async function () {

    let showSwagForm = false;
    // Due to shipping costs, we'll check the requesting user's cf-ipcountry to see if they're in the US, and their cf-iplongitude header to see if they're in the contiguous US.
    if(sails.config.environment === 'production') {
      // Log a warning if the cloudflare headers we use are missing in production.
      if(!this.req.get('cf-ipcountry') || !this.req.get('cf-iplongitude')) {
        sails.log.warn('When a user visted the transparency page, the Cloudflare header we use to determine if they are visiting from the contiguous United States is missing.');
      }
    }
    if(this.req.get('cf-ipcountry') === 'US' && this.req.get('cf-iplongitude') > -125) {
      showSwagForm = true;
    }
    // Respond with view.
    return {
      showSecureframeBanner: this.req.param('utm_content') === 'secureframe',
      showSwagForm,
    };

  }


};
