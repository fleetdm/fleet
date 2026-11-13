module.exports = {


  friendlyName: 'View profile generator usage',


  description: 'Display "Profile generator usage" page, an admin tool for viewing how much the AI configuration profile generator (autofill) is used.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/profile-generator-usage'
    }

  },


  fn: async function () {

    // Note: Only successful generations are counted.  Neither the website nor the Fleet server records failed
    // AI autofill requests, so there is no error rate to show yet.
    //
    // Lifetime count: Adding up the usage statistics takes too long to do while this page loads, so the
    // send-aggregated-metrics-to-datadog script calculates it and saves it to the platform record.
    let platformRecords = await Platform.find();
    let platformRecord = platformRecords[0];
    if (!platformRecord) {
      throw new Error(`Consistency violation: when a user visited the profile generator usage page, no platform record was found.`);
    }

    // Requests this month: Fleet servers on the free tier register with the profile generator, which counts
    // their requests per UTC month.  (Fleet Premium requests are not limited and not counted here.)
    let currentMonth = new Date().toISOString().slice(0, 7);
    let registrationsActiveThisMonth = await ProfileGeneratorRegistration.find({
      where: {usageMonth: currentMonth},
      select: ['profileGenerationsThisMonth', 'nameAndDescriptionRequestsThisMonth'],
    });

    return {
      numAiProfilesGeneratedAllTime: platformRecord.numAiGeneratedProfilesAllTime,
      allTimeStatsLastUpdatedAt: platformRecord.aiGeneratedProfileStatsLastUpdatedAt,
      numFreeProfileRequestsThisMonth: _.sum(registrationsActiveThisMonth, 'profileGenerationsThisMonth'),
      numFreeNameAndDescriptionRequestsThisMonth: _.sum(registrationsActiveThisMonth, 'nameAndDescriptionRequestsThisMonth'),
    };

  }


};
