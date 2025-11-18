module.exports = {


  friendlyName: 'View fleet premium trial',


  description: 'Display "Fleet premium trial" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleet-premium-trial'
    },

    redirectToLiveFleetInstance: {
      description: 'The user was redirected to their Fleet Premium trial instance.',
      responseType: 'redirect',
    },
    redirectToCustomerDashboard: {
      description: 'This user was redirected to their customer dashboard',
      responseType: 'redirect'
    }

  },


  fn: async function () {

    // If the user has a license key, we'll redirect them to the customer dashboard.
    let userHasExistingSubscription = await Subscription.findOne({user: this.req.me.id});
    if (userHasExistingSubscription) {
      throw {redirectToCustomerDashboard: '/customers/dashboard'};
    }

    let trialIsExpired = false;
    let trialLicenseKey;

    if(this.req.me.fleetPremiumTrialLicenseKey) {
      if(this.req.me.fleetPremiumTrialLicenseKeyExpiresAt < Date.now()) {
        trialIsExpired = true;
      }
      trialLicenseKey = this.req.me.fleetPremiumTrialLicenseKey;
      // If this user has an active Fleet premium trial instance on Render, redirect them to it.
      if(this.req.me.fleetPremiumTrialType === 'render trial' && !trialIsExpired) {
        // Find the associated database record for this user's Render trial.
        let renderTrialRecord = await RenderProofOfValue.findOne({user: this.req.me.id});
        throw { redirectToLiveFleetInstance: renderTrialRecord.instanceUrl };
      }
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
    }

    // Respond with view.
    return {
      trialIsExpired,
      trialType: this.req.me.fleetPremiumTrialType,
      trialLicenseKey,
    };

  }


};
