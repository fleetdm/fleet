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
    }

  },


  fn: async function ({start}) {

    let userHasTrialLicense = false;
    let trialLicenseKey;
    let userHasExpiredTrialLicense = false;

    if(this.req.me) {
      userHasTrialLicense = this.req.me.fleetPremiumTrialLicenseKey;
      // Check to see if this user has a Fleet premium trial license key.
      if(userHasTrialLicense) {
        if(this.req.me.fleetPremiumTrialLicenseKeyExpiresAt < Date.now()) {
          userHasExpiredTrialLicense = true;
        }
        trialLicenseKey = this.req.me.fleetPremiumTrialLicenseKey;
      } else {
        // If this user is logged in and does not have a trial license key, generate a new one for them.
        let thirtyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 30);
        let trialLicenseKeyForThisUser = await sails.helpers.createLicenseKey.with({
          numberOfHosts: 10,
          organization: this.req.me.organization,
          expiresAt: thirtyDaysFromNowAt,
        });
        // Save the trial license key to the DB record for this user.
        await User.updateOne({id: this.req.me.id})
        .set({
          fleetPremiumTrialLicenseKey: trialLicenseKeyForThisUser,
          fleetPremiumTrialLicenseKeyExpiresAt: thirtyDaysFromNowAt,
        });
        trialLicenseKey = trialLicenseKeyForThisUser;
        userHasTrialLicense = true;
      }
    }

    // Respond with view.
    return {
      hideNextStepsButtons: start,
      trialLicenseKey,
      userHasTrialLicense,
      userHasExpiredTrialLicense,
    };

  }


};
