module.exports = {


  friendlyName: 'Send trial usage information to CRM',


  description: 'Reports recent usage information about Render trial instances.',

  inputs: {
    reportAllHistoricalData: {
      type: 'boolean',
      description: 'Whether or not to report details for all past render trial instnaces or not.',
      extendedDescription: 'This is meant to be used once when the script is first created. Without this flag enabled, this script will only report analytics for active and recently expired Render trial instances.'
    }
  },

  fn: async function ({reportAllHistoricalData}) {

    sails.log('Running custom shell script... (`sails run send-trial-usage-information-to-crm`)');

    let nowAt = Date.now();
    let oneDayAgoAt = nowAt - (1000 * 60 * 60 * 24);


    let renderTrialInstancesToSendAnalyticsFor;
    if(reportAllHistoricalData) {
      // Find all active and expired render instance details.
      renderTrialInstancesToSendAnalyticsFor = await RenderProofOfValue.find({status: {'!=': 'record created'}});
    } else {
      // Find render instance details for active Render trials, and trials that have expired in the past 24 hours.
      let thirtyDaysFromNowAt = nowAt + (1000 * 60 * 60 * 24 * 30);
      renderTrialInstancesToSendAnalyticsFor = await RenderProofOfValue.find({
        renderTrialEndsAt: { '>=': oneDayAgoAt, '<=':  thirtyDaysFromNowAt }
      });
    }


    sails.log(`Reporting Render trial usage information for ${renderTrialInstancesToSendAnalyticsFor.length} trial instances`);
    await sails.helpers.flow.simultaneouslyForEach(renderTrialInstancesToSendAnalyticsFor, async (renderTrial)=>{

      let lastReportedStatisticsForThisTrial = await HistoricalUsageSnapshot.find({
        organization: 'Render-trial-'+renderTrial.slug,
      }).sort('createdAt DESC').limit(1);
      // If no records were found, then search for one that is not prefixed with 'Render-trial-'
      if(lastReportedStatisticsForThisTrial.length < 1) {
        sails.log(`No analytics found for prefixed organization, searching for ${renderTrial.slug}`);
        lastReportedStatisticsForThisTrial = await HistoricalUsageSnapshot.find({
          organization: renderTrial.slug,
        }).sort('createdAt DESC').limit(1);
      }
      if(lastReportedStatisticsForThisTrial.length < 1) {
        // If we didn't find usage statistics reported by a Render trial instance, log a warning and continue.
        sails.log(`Skipping reporting information for a trial instance (slug: ${renderTrial.slug}). No usage analytics were found reported by this Render trial`);
        return;
      }
      let thisRenderTrialsUser = await User.findOne({id: renderTrial.user});
      if(!thisRenderTrialsUser) {
        // If the user record associated with this Render trial is missing, (e.g., if this person requested that we delete their account) log a warning and continue.
        sails.log(`Skipping reporting information for a trial instance (slug: ${renderTrial.slug}). No user could be found that was associated with this Render trial.`);
        return;
      }
      // Create a formatted timestamp of when this Render trial was started (When this user signed up)
      let renderTrialStartedOn = new Date(thisRenderTrialsUser.createdAt);
      let formattedTimestampOfWhenThisRenderTrialStarted = renderTrialStartedOn.toISOString().replace('Z', '+0000');
      // Create a formatted timestamp of when this Render trial ends.
      let renderTrialEndsOn = new Date(thisRenderTrialsUser.fleetPremiumTrialLicenseKeyExpiresAt);
      let formattedTimestampOfWhenThisRenderTrialends = renderTrialEndsOn.toISOString().replace('Z', '+0000');
      // Create a formatted timestamp of when this Render trial last reported usage statistics.
      let trialReportedAnalyticsOn = new Date(lastReportedStatisticsForThisTrial[0].createdAt);
      let formattedTimestampOfWhenThisInstanceReportedAnalytics = trialReportedAnalyticsOn.toISOString().replace('Z', '+0000');

      // Build a trialInstanceUsageDetails to send to CRM helper.
      let trialInstanceUsageDetails = {
        status: renderTrial.status,
        lastUpdatedOn: formattedTimestampOfWhenThisInstanceReportedAnalytics,
        trialStartedOn: formattedTimestampOfWhenThisRenderTrialStarted,
        trialEndsOn: formattedTimestampOfWhenThisRenderTrialends,
        numUsers: lastReportedStatisticsForThisTrial[0].numUsers,
        numHostsEnrolled: lastReportedStatisticsForThisTrial[0].numHostsEnrolled
      };
      // Skip reporting usage information for users with a fleetdm.com email address.
      let emailDomain = thisRenderTrialsUser.emailAddress.split('@')[1];
      if(emailDomain.toLowerCase() === 'fleetdm.com'){
        sails.log(`Skipping reporting usage information for a Render trial instance (slug: ${renderTrial.slug}) because it is used by a fleetdm.com email address`);
        return;
      }

      // Update the contact record that was created for this user when they signed up.
      await sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
        emailAddress: thisRenderTrialsUser.emailAddress,
        firstName: thisRenderTrialsUser.firstName,
        lastName: thisRenderTrialsUser.lastName,
        contactSource: 'Website - Sign up',
        trialInstanceUsageDetails: trialInstanceUsageDetails
      }).tolerate((err)=>{
        sails.log.warn(`When reporting usage information about a Render trial instance (slug: ${renderTrial.slug}), an error occured when updating/creating a Salesforce contact/account. Full error: ${require('util').inspect(err)}`);
      });

    });// After each Render trial Instance


  }


};

