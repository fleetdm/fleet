module.exports = {


  friendlyName: 'Reset one fleet premium local trial',


  description: '',


  inputs: {
    emailAddress: { type: 'string', required: true, },
  },


  exits: {
    success: {description: 'A user\'s Fleet premium trial was successfully reset.'},
    userNotFound: {description: 'No user account was found using the provided email address.', responseType: 'notFound'},
    trialTypeUnsupported: {description: 'This user has a Render trial.', responseType: 'badRequest'}
  },


  fn: async function ({emailAddress}) {

    let thisUser = await User.findOne({emailAddress});

    if(!thisUser) {
      throw 'userNotFound';
    }

    if(thisUser.fleetPremiumTrialType === 'render trial') {
      throw 'trialTypeUnsupported';
    }

    let thirtyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 30);

    let newTrialLicenseKeyForThisUser = await sails.helpers.createLicenseKey.with({
      numberOfHosts: 10,
      organization: thisUser.organization,
      expiresAt: thirtyDaysFromNowAt,
    });

    await User.updateOne({id: thisUser.id}).set({
      fleetPremiumTrialLicenseKeyExpiresAt: thirtyDaysFromNowAt,
      fleetPremiumTrialLicenseKey: newTrialLicenseKeyForThisUser,
    });

    // All done.
    return;

  }


};
