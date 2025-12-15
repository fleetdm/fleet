module.exports = {


  friendlyName: 'Deliver expired local trial emails',


  description: 'A script designed to be run daily, that sends emails to Fleetdm.com users whose Fleet premium trial expired in the past 24 hours.',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run deliver-expired-local-trial-emails`)');

    let nowAt = Date.now();
    let oneDayAgoAt = nowAt - (1000 * 60 * 60 * 24);

    // Build a list of users with a local Fleet Premium trial that has expired in the past 24 hours.
    let usersWithRecentlyExpiredLocalTrials = await User.find({
      fleetPremiumTrialType: 'local trial',
      fleetPremiumTrialLicenseKeyExpiresAt: {
        '>=': oneDayAgoAt,
        '<': nowAt,
      }
    });


    for(let expiredTrialUser of usersWithRecentlyExpiredLocalTrials) {
      // Send an "Your fleet trial has ended" email to the user.
      await sails.helpers.sendTemplateEmail.with({
        to: expiredTrialUser.emailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your Fleet trial has ended',
        template: 'email-fleet-premium-local-trial-ended',
        layout: 'layout-nurture-email',
        templateData: {
          firstName: expiredTrialUser.firstName,
        },
        ensureAck: true,
      }).tolerate((err)=>{
        sails.log.warn(`When sending an email to a user with a newly expired Fleet Premium local trial (email: ${expiredTrialUser.emailAddress}) an error occured. Full error: ${require('util').inspect(err)}`);
        return;
      });

    }

    sails.log(`Sent expired trial emails for ${usersWithRecentlyExpiredLocalTrials.length} user(s).`);

  }


};

