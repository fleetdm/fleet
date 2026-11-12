module.exports = {


  friendlyName: 'View superadmins',


  description: 'Display "Superadmins" page, an admin tool for viewing and managing which users have superadmin access to the Fleet website.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/superadmins'
    }

  },


  fn: async function () {

    // Only select the attributes that the page needs, so that password hashes, tokens, etc. are never sent to the browser.
    let superAdmins = await User.find({
      where: {isSuperAdmin: true},
      select: ['id', 'firstName', 'lastName', 'emailAddress', 'lastSeenAt', 'createdAt'],
    })
    .sort('emailAddress ASC');

    // Get the number of AI-generated configuration profile (autofill) requests.
    // Note: Only successful generations are counted.  Neither the website nor the Fleet server records failed
    // AI autofill requests, so there is no error rate to show yet.
    //
    // Lifetime count: Fleet servers report a cumulative count of the profiles they have generated in their usage
    // statistics, so we add up the latest report from each Fleet deployment that reported in the past week (the
    // same way the send-aggregated-metrics-to-datadog script picks the latest reports.)
    let oneWeekAgoAt = Date.now() - (1000 * 60 * 60 * 24 * 7);
    let recentUsageSnapshots = await HistoricalUsageSnapshot.find({
      where: {createdAt: {'>=': oneWeekAgoAt}},
      select: ['id', 'anonymousIdentifier', 'organization', 'numAppleAiGeneratedProfiles', 'numWindowsAiGeneratedProfiles', 'numAndroidAiGeneratedProfiles'],
    });
    let latestSnapshotsByDeployment = _.map(_.groupBy(_.filter(recentUsageSnapshots, (snapshot)=>{
      // Ignore development licenses and loadtests.
      return !_.contains(['Fleet Sandbox', 'fleet-loadtest', 'development-only', 'Dev license (expired)', ''], snapshot.organization);
    }), 'anonymousIdentifier'), (snapshots)=>{
      return _.max(snapshots, 'id');
    });
    let numAiProfilesGeneratedAllTime = _.sum(latestSnapshotsByDeployment, (snapshot)=>{
      // Snapshots saved before Fleet servers reported these counts will not have them.
      return (snapshot.numAppleAiGeneratedProfiles || 0) + (snapshot.numWindowsAiGeneratedProfiles || 0) + (snapshot.numAndroidAiGeneratedProfiles || 0);
    });

    // Requests this month: Fleet servers on the free tier register with the profile generator, which counts
    // their requests per UTC month.  (Fleet Premium requests are not limited and not counted here.)
    let currentMonth = new Date().toISOString().slice(0, 7);
    let registrationsActiveThisMonth = await ProfileGeneratorRegistration.find({
      where: {usageMonth: currentMonth},
      select: ['profileGenerationsThisMonth', 'nameAndDescriptionRequestsThisMonth'],
    });
    let numFreeProfileRequestsThisMonth = _.sum(registrationsActiveThisMonth, 'profileGenerationsThisMonth');
    let numFreeNameAndDescriptionRequestsThisMonth = _.sum(registrationsActiveThisMonth, 'nameAndDescriptionRequestsThisMonth');

    return {
      superAdmins,
      currentUserId: this.req.me.id,
      aiAutofillStats: {
        numAiProfilesGeneratedAllTime,
        numFreeProfileRequestsThisMonth,
        numFreeNameAndDescriptionRequestsThisMonth,
      },
    };

  }


};
