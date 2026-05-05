module.exports = {


  friendlyName: 'Migrate users with no trial key',


  description: 'Generate qualifying users a new Fleet premium license key and inform them via email.',

  inputs: {
    dry: { type: 'boolean', description: 'Whether to make this a dry run.  (No database changes will be made.  Emails will not be sent.)' },
  },

  fn: async function ({dry}) {

    sails.log('Running custom shell script... (`sails run migrate-users-with-no-trial-key`)');

    let thirtyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 30);
    // Get an array of all Users with
    let idsOfUsersWithSubscriptions = await Subscription.find().select('user');
    let numberOfUsersWhoWouldHaveBeenUpdated = 0;

    await User.stream({
      lastSubmittedGetStartedQuestionnaireStep: {
        'in': [ // Only update users who have made it to the 'Is-it-any-good' step of the start questionnaire.
          'is-it-any-good',
          'what-did-you-think',
          'deploy-fleet-in-your-environment',
          'managed-cloud-for-growing-deployments',
          'self-hosted-deploy',
          'whats-left-to-get-you-set-up'
        ]
      },
      psychologicalStage: {'!=': '2 - Aware'},// Skip users who have told us that they decided not to use Fleet.
      stageThreeNurtureEmailSentAt: {'!=': 1},// Do not update users who have unsubscribed from marketing emails.
    }).eachRecord(async (thisUser)=>{
      // Stop running if this user has purchased a fleet premium license.
      if(idsOfUsersWithSubscriptions.includes(thisUser.id)){
        sails.log.verbose(`Skipping ${thisUser.emailAddress} because they already have a premium subscription.`);
        return;
      }

      if(dry) {
        sails.log.verbose(`Would have generated a new trial license key for ${thisUser.emailAddress} and informed them via email`);
        numberOfUsersWhoWouldHaveBeenUpdated++;
      } else {
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

        // Send an email informing the user that their new Fleet premium trial is available.
        await sails.helpers.sendTemplateEmail.with({
          template: 'email-fleet-premium-trial',
          layout: false,
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

    if(dry){
      sails.log(`Dry run: would have generated trial licenses for and updated ${numberOfUsersWhoWouldHaveBeenUpdated} user records.`);
    }



  }


};

