module.exports = {


  friendlyName: 'View fleet premium trial',


  description: 'Display "Fleet premium trial" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleet-premium-trial'
    },

    redirectToLiveFleetInstance: {
      description: 'The user was redirected to their Premium Fleet isntance trial.',
      responseType: 'redirect',
    }

  },


  fn: async function () {

    let trialType;
    let trialIsExpired = false;
    // If this user does not have a Fleet premium trial license, show the expired trial state.
    if(!this.req.me.fleetPremiumTrialLicenseKey || !this.req.me.fleetPremiumTrialLicenseKeyExpiresAt){
      return {
        trialIsExpired: true,
        trialType: 'local-trial'
      };
    }
    let trialExpiresAt = this.req.me.fleetPremiumTrialLicenseKeyExpiresAt;
    if(trialExpiresAt < Date.now()) {

    }

    if(this.req.me.fleetPremiumTrialType === 'render-trial') {
      // Find the associated database record for this user's Render trial.
      let renderTrialRecord = await RenderProofOfValue.findOne({user: this.req.me.id});
      // Check to see if the trial has ended.
      if(renderTrialRecord.renderTrialEndsAt > Date.now()){
        // If the trial is still active, redirect the user to their trial instance.
        return { redirectToLiveFleetInstance: renderTrialRecord.instanceUrl }
      }
    } else {
      let trialLicenseKey = this.req.me.fleetPremiumTrialLicenseKey;
    }


    // Respond with view.
    return {
      trialIsExpired,
      trialType: this.req.me.fleetPremiumTrialType,
    };

  }


};
