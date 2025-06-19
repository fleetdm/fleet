module.exports = {


  friendlyName: 'Migrate users with no trial key',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run migrate-users-with-no-trial-key`)');

    let thirtyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 30);


    await User.stream({}).eachRecord(async (thisUser)=>{
      // Generate a new trial license key for the user.
      let trialLicenseKeyForThisUser = await sails.helpers.createLicenseKey.with({
        numberOfHosts: 10,
        organization: thisUser.organization ? thisUser.organization : 'Fleet Premium trial',
        expiresAt: thirtyDaysFromNowAt,
      });

      // Save the new license key to this user's db record
      await User.updateOne({id: thisUser.id}).set({
        fleetPremiumTrialLicenseKey: trialLicenseKeyForThisUser,
        fleetPremiumTrialLicenseKeyExpiresAt: thirtyDaysFromNowAt,
      });

      // Only continue if this user is not unsubscribed from marketing emails.
      if(thisUser.stageThreeNurtureEmailSentAt !== 1){
        // Send an email informing the user that their new Fleet premium trial is available.
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-fleet-premium-trial',
          layout: 'layout-email',
          templateData: {
            emailAddress: thisUser.emailAddress
          },
          to: thisUser.emailAddress,
          subject: 'Whoops',
          from: sails.config.custom.contactFormEmailAddress,
          fromName: 'Mike McNeil',
          ensureAck: true,
        });
      }



    });



  }


};

