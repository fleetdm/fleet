module.exports = {


  friendlyName: 'Send usage statistics to Google sheet',


  description: 'Replaces the contents of the usage statistics Google sheet with the latest usage statistics reported by Fleet Premium instances.',


  extendedDescription: 'This script keeps an internal Google sheet up to date with the latest reported usage statistics, replacing a manual monthly export. It is intended to be run daily by the Heroku scheduler.',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run send-usage-statistics-to-google-sheet`)');

    require('assert')(sails.config.custom.usageStatisticsServiceAccountEmailAddress);
    require('assert')(sails.config.custom.usageStatisticsServiceAccountPrivateKey);

    // Hardcoded to a test sheet while the format is fine-tuned. Once the format settles, this will
    // point at the sheet everyone references.
    const SPREADSHEET_ID = '1YVTgjabIHLt0bXAExMuxkOhFm1KPr0LhGHFiCDKmHRI';

    // Organizations reported by internal, development, and load testing instances.
    const ORGANIZATIONS_TO_EXCLUDE = [
      'unknown',
      'development-only',
      'fleet-loadtest',
      'Fleet Sandbox',
      'development',
      'Fleet - QA Load Test',
      'Webhook Tester Pentest Org',
      'QA Render Instance',
      'Fleet (QA Wolf)',
      'Fleet Device Management, Inc. Developer',
      'fleet',
      'Fleet internal premium key',
      'Fleet internal training',
      'Fleet CSE Sandbox',
      'Fleet Demo',
      'Fleet Premium trial',
      'Dev license',
      'DEV License',
      '',
    ];

    let nowAt = Date.now();
    let ninetyDaysAgoAt = nowAt - (1000 * 60 * 60 * 24 * 90);

    let usageStatisticsReportedInTheLastNinetyDays = await HistoricalUsageSnapshot.find({
      createdAt: { '>=': ninetyDaysAgoAt },
      licenseTier: 'premium',
      organization: { nin: ORGANIZATIONS_TO_EXCLUDE },
    });

    // Reduce to the latest report from each Fleet instance.
    let statisticsReportedByFleetInstance = _.groupBy(usageStatisticsReportedInTheLastNinetyDays, 'anonymousIdentifier');
    let latestStatisticsForEachInstance = [];
    for (let id in statisticsReportedByFleetInstance) {
      let lastReportIdForThisInstance = _.max(_.pluck(statisticsReportedByFleetInstance[id], 'id'));
      latestStatisticsForEachInstance.push(_.find(statisticsReportedByFleetInstance[id], {id: lastReportIdForThisInstance}));
    }
    // Show the largest deployments first.
    latestStatisticsForEachInstance = _.sortByOrder(latestStatisticsForEachInstance, 'numHostsEnrolled', 'desc');

    // Platforms reported in hostsEnrolledByOperatingSystem that are not their own column. Anything unrecognized is counted as Linux, since Fleet reports each Linux distribution as its own platform.
    const PLATFORM_COLUMNS = {darwin: 'macOS', windows: 'Windows', ios: 'iOS', ipados: 'iPadOS', android: 'Android', chrome: 'ChromeOS'};

    let rows = [];
    for (let statistics of latestStatisticsForEachInstance) {
      let hostCountsByPlatform = {macOS: 0, Windows: 0, Linux: 0, iOS: 0, iPadOS: 0, Android: 0, ChromeOS: 0};
      for (let reportedPlatform in statistics.hostsEnrolledByOperatingSystem) {
        let columnForThisPlatform = PLATFORM_COLUMNS[reportedPlatform.toLowerCase()] || 'Linux';
        for (let versionInfo of statistics.hostsEnrolledByOperatingSystem[reportedPlatform]) {
          hostCountsByPlatform[columnForThisPlatform] += versionInfo.numEnrolled || 0;
        }
      }
      let readableHostCountsByPlatform = _.map(_.pick(hostCountsByPlatform, (count)=>{ return count > 0; }), (count, platform)=>{
        return `${platform}: ${count.toLocaleString('en-US')}`;
      }).join(' · ');

      rows.push([
        new Date(statistics.updatedAt).toISOString(),
        statistics.anonymousIdentifier,
        statistics.organization,
        statistics.fleetVersion,
        statistics.numHostsEnrolled,
        statistics.numUsers,
        statistics.numLabels,
        statistics.softwareInventoryEnabled,
        statistics.vulnDetectionEnabled,
        statistics.systemUsersEnabled,
        statistics.hostsStatusWebHookEnabled,
        statistics.mdmMacOsEnabled,
        statistics.mdmWindowsEnabled,
        statistics.liveQueryDisabled,
        statistics.hostExpiryEnabled,
        statistics.numSoftwareVersions,
        statistics.numHostSoftwares,
        statistics.numSoftwareTitles,
        statistics.numHostSoftwareInstalledPaths,
        statistics.numSoftwareCPEs,
        statistics.numSoftwareCVEs,
        statistics.aiFeaturesDisabled,
        statistics.maintenanceWindowsEnabled,
        statistics.maintenanceWindowsConfigured,
        statistics.numHostsFleetDesktopEnabled,
        statistics.numQueries,
        statistics.numHostsABMPending,
        readableHostCountsByPlatform,
        (statistics.fleetMaintainedAppsMacOS || []).join(', '),
        (statistics.fleetMaintainedAppsWindows || []).join(', '),
        statistics.oktaConditionalAccessConfigured,
        statistics.conditionalAccessEnabled,
        statistics.conditionalAccessBypassDisabled,
        statistics.entraConditionalAccessConfigured,
        hostCountsByPlatform.macOS,
        hostCountsByPlatform.Windows,
        hostCountsByPlatform.Linux,
        hostCountsByPlatform.iOS,
        hostCountsByPlatform.iPadOS,
        hostCountsByPlatform.Android,
        hostCountsByPlatform.ChromeOS,
      ]);
    }//∞

    if (rows.length === 0) {
      // Leave the existing sheet contents in place rather than wiping them with an empty result.
      sails.log.warn('The send-usage-statistics-to-google-sheet script found no usage statistics reported by Fleet Premium instances in the last 90 days. The Google sheet was not updated.');
      return;
    }

    const HEADER_ROW = [
      'Last updated', 'Anonymous identifier', 'Organization', 'Fleet version', 'Hosts enrolled',
      'Users', 'Labels', 'Software inventory', 'Vulnerability detection', 'System users',
      'Host status webhook', 'macOS MDM', 'Windows MDM', 'Live queries disabled', 'Host expiry',
      'Software versions', 'Host software records', 'Software titles', 'Software installed paths',
      'Software CPEs', 'Software CVEs', 'AI features disabled', 'Maintenance windows',
      'Maintenance windows configured', 'Hosts with Fleet Desktop', 'Reports', 'Hosts pending in ABM',
      'Hosts by platform', 'Fleet-maintained apps (macOS)', 'Fleet-maintained apps (Windows)',
      'Okta conditional access configured', 'Conditional access enabled',
      'Conditional access bypass disabled', 'Entra conditional access configured',
      'macOS hosts', 'Windows hosts', 'Linux hosts', 'iOS hosts', 'iPadOS hosts', 'Android hosts', 'ChromeOS hosts',
    ];

    let { google } = require('googleapis');
    let googleAuth = new google.auth.GoogleAuth({
      scopes: ['https://www.googleapis.com/auth/spreadsheets'],
      credentials: {
        client_email: sails.config.custom.usageStatisticsServiceAccountEmailAddress,// eslint-disable-line camelcase
        private_key: sails.config.custom.usageStatisticsServiceAccountPrivateKey,// eslint-disable-line camelcase
      },
    });
    let sheets = google.sheets({version: 'v4', auth: googleAuth});

    // Clear the entire tab before writing so deployments that stopped reporting don't linger as stale rows.
    await sheets.spreadsheets.values.clear({
      spreadsheetId: SPREADSHEET_ID,
      range: 'Data',
    });
    await sheets.spreadsheets.values.update({
      spreadsheetId: SPREADSHEET_ID,
      range: 'Data!A1',
      valueInputOption: 'RAW',
      requestBody: {
        values: [HEADER_ROW].concat(rows),
      },
    });

    sails.log(`Usage statistics for ${rows.length} Fleet Premium deployments were sent to the usage statistics Google sheet.`);
  }


};
