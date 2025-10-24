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
    }

  },


  fn: async function () {
    let trialIsExpired = false;
    // If this user does not have a Fleet Premium trial, show the expired state for local trials.
    if(!this.req.me.fleetPremiumTrialLicenseKey || !this.req.me.fleetPremiumTrialLicenseKeyExpiresAt) {
      return {
        trialIsExpired: true,
        trialExpiredAt: 0,
        trialType: 'local-trial'
      };
    }

    let trialExpiresAt = this.req.me.fleetPremiumTrialLicenseKeyExpiresAt;
    if(trialExpiresAt < Date.now()) {
      trialIsExpired = true;
    }

    // If this user has an active Fleet premium trial instance on Render, redirect them to it.
    if(this.req.me.fleetPremiumTrialType === 'render-trial' && !trialIsExpired) {
      // Find the associated database record for this user's Render trial.
      let renderTrialRecord = await RenderProofOfValue.findOne({user: this.req.me.id});
      throw { redirectToLiveFleetInstance: renderTrialRecord.instanceUrl };
    }

    // Respond with view.
    return {
      trialIsExpired: trialIsExpired,
      trialExpiredAt: this.req.me.fleetPremiumTrialLicenseKeyExpiresAt,
      trialType: this.req.me.fleetPremiumTrialType,
    };

  }


};
