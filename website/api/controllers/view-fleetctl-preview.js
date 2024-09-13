module.exports = {


  friendlyName: 'View fleetctl preview',


  description: 'Display "fleetctl preview" page.',

  inputs: {
    start: {
      type: 'boolean',
      description: 'A boolean flag that will hide the "next steps" buttons on the page if set to true',
      defaultsTo: false,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/fleetctl-preview'
    },

    redirect: {
      description: 'The requesting user is not logged in.',
      responseType: 'redirect'
    },

  },


  fn: async function ({start}) {

    let trialLicenseKey;
    // Check to see if this user has a Fleet premium trial license key.
    let userHasTrialLicense = this.req.me.fleetPremiumTrialLicenseKey;
    let userHasExpiredTrialLicense = false;
    if(userHasTrialLicense) {
      if(this.req.me.fleetPremiumTrialLicenseKeyExpiresAt < Date.now()) {
        userHasExpiredTrialLicense = true;
      }
      trialLicenseKey = this.req.me.fleetPremiumTrialLicenseKey;
    } else {
      trialLicenseKey = '';
    }

    // Respond with view.
    return {
      hideNextStepsButtons: start,
      trialLicenseKey,
      userHasExpiredTrialLicense,
    };

  }


};
