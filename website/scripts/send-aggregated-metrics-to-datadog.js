module.exports = {


  friendlyName: 'Send aggregated metrics to datadog',


  description: 'Sends the aggregated metrics for usage statistics reported by Fleet instances in the past week',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run send-metrics-to-datadog`)');

    let nowAt = Date.now();
    let oneWeekAgoAt = nowAt - (1000 * 60 * 60 * 24 * 7);
    // get a timestamp in seconds to use for the metrics we'll send to datadog.
    let timestampForTheseMetrics = Math.floor(nowAt / 1000);
    // Get all the usage snapshots for the past week.
    let usageStatisticsReportedInTheLastWeek = await HistoricalUsageSnapshot.find({
      createdAt: { '>=': oneWeekAgoAt},// Search for records created in the past week.
    })
    .sort('createdAt DESC');// Sort the results by the createdAt timestamp
    // Filter out development premium licenses and loadtests.
    let filteredStatistics = _.filter(usageStatisticsReportedInTheLastWeek, (report)=>{
      return !_.contains(['Fleet Sandbox', 'fleet-loadtest', 'development-only', 'Dev license (expired)', ''], report.organization);
    });

    let statisticsReportedByFleetInstance = _.groupBy(filteredStatistics, 'anonymousIdentifier');

    let metricsToReport = [];
    let latestStatisticsForEachInstance = [];
    for (let id in statisticsReportedByFleetInstance) {
      let lastReportIdForThisInstance = _.max(_.pluck(statisticsReportedByFleetInstance[id], 'id'));
      let latestReportFromThisInstance = _.find(statisticsReportedByFleetInstance[id], {id: lastReportIdForThisInstance});
      latestStatisticsForEachInstance.push(latestReportFromThisInstance);
    }
    // Get a filtered array of metrics reported by Fleet Premium instances
    let latestPremiumUsageStatistics = _.filter(latestStatisticsForEachInstance, {licenseTier: 'premium'});
    // Group reports by organization name.
    let reportsByOrgName = _.groupBy(latestPremiumUsageStatistics, 'organization');
    for(let org in reportsByOrgName) {
      // Sort the results for this array by the createdAt value. This makes sure we're always sending the most recent results.
      let reportsForThisOrg = _.sortByOrder(reportsByOrgName[org], 'createdAt', 'desc');
      let lastReportForThisOrg = reportsForThisOrg[0];
      // Get the metrics we'll report for each org.
      // Combine the numHostsEnrolled values from the last report for each unique Fleet instance that reports this organization.
      let totalNumberOfHostsReportedByThisOrg = _.sum(reportsForThisOrg, (report)=>{
        return report.numHostsEnrolled;
      });
      let lastReportedFleetVersion = lastReportForThisOrg.fleetVersion;
      let hostCountMetricForThisOrg = {
        metric: 'usage_statistics.num_hosts_enrolled_by_org',
        type: 3,
        points: [{
          timestamp: timestampForTheseMetrics,
          value: totalNumberOfHostsReportedByThisOrg
        }],
        resources: [{
          name: reportsByOrgName[org][0].anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: [
          `organization:${org}`,
          `fleet_version:${lastReportedFleetVersion}`,
        ],
      };
      metricsToReport.push(hostCountMetricForThisOrg);
    }
    // Filter the statistics to be only for released versions of Fleet.
    // Note: we're doing this after we've reported the metrics for Fleet Premium instances to make sure
    // that we are reporting metrics sent by customers who may be using a non-4.x.x version of Fleet.
    let latestStatisticsReportedByReleasedFleetVersions = _.filter(latestStatisticsForEachInstance, (statistics)=>{
      return _.startsWith(statistics.fleetVersion, '4.');
    });
    let numberOfInstancesToReport = latestStatisticsReportedByReleasedFleetVersions.length;
    // Build aggregated metrics for JSON attrributes
    // Create an empty object to store combined host counts.
    let combinedHostsEnrolledByOperatingSystem = {};
    // Get an array of the last reported hostsEnrolledByOperatingSystem values.
    let allHostsEnrolledByOsValues = _.pluck(latestStatisticsReportedByReleasedFleetVersions, 'hostsEnrolledByOperatingSystem');
    // Iterate through each reported value, and combine them.
    for(let reportedHostCounts of allHostsEnrolledByOsValues) {
      _.merge(combinedHostsEnrolledByOperatingSystem, reportedHostCounts, (combinedCountsForThisOperatingSystemType, countsForThisOperatingSystemType) => {
        if(Array.isArray(combinedCountsForThisOperatingSystemType) && Array.isArray(countsForThisOperatingSystemType)){
          let mergedArrayOfHostCounts = [];
          // Iterate through the counts in the array we're combining with the aggregator object.
          for (let versionInfo of countsForThisOperatingSystemType) {
            let matchingVersionFromCombinedCounts = _.find(combinedCountsForThisOperatingSystemType, (osType) => osType.version === versionInfo.version);
            if (matchingVersionFromCombinedCounts) {
              mergedArrayOfHostCounts.push({ version: versionInfo.version, numEnrolled: versionInfo.numEnrolled + matchingVersionFromCombinedCounts.numEnrolled });
            } else {
              mergedArrayOfHostCounts.push(versionInfo);
            }
          }
          // Now add the hostCounts from the combined host counts.
          for (let versionInfo of combinedCountsForThisOperatingSystemType) {
            let versionOnlyExistsInCombinedCounts = !_.find(countsForThisOperatingSystemType, (osVersion)=>{ return osVersion.version === versionInfo.version;});
            if (versionOnlyExistsInCombinedCounts) {
              mergedArrayOfHostCounts.push(versionInfo);
            }
          }
          return mergedArrayOfHostCounts;
        }
      });
    }
    for(let operatingSystem in combinedHostsEnrolledByOperatingSystem) {
      // For every object in the array, we'll send a metric to track host count for each operating system version.
      for(let osVersion of combinedHostsEnrolledByOperatingSystem[operatingSystem]) {
        // Only continue if the object in the array has a numEnrolled and version value.
        if(osVersion.numEnrolled && osVersion.version !== '') {
          let metricToAdd = {
            metric: 'usage_statistics_v2.host_count_by_os_version',
            type: 3,
            points: [{timestamp: timestampForTheseMetrics, value:osVersion.numEnrolled}],
            resources: [{name: operatingSystem, type: 'os_type'}],
            tags: [`os_version_name:${osVersion.version}`],
          };
          // Add the custom metric to the array of metrics to send to Datadog.
          metricsToReport.push(metricToAdd);
        }//ﬁ
      }//∞
    }//∞


    let allHostsEnrolledByOsqueryVersion = _.pluck(latestStatisticsReportedByReleasedFleetVersions, 'hostsEnrolledByOsqueryVersion');
    let combinedHostsEnrolledByOsqueryVersion = [];
    let flattenedHostsEnrolledByOsqueryVersions = _.flatten(allHostsEnrolledByOsqueryVersion);
    let groupedHostsEnrolledValuesByOsqueryVersion = _.groupBy(flattenedHostsEnrolledByOsqueryVersions, 'osqueryVersion');
    for(let osqueryVersion in groupedHostsEnrolledValuesByOsqueryVersion) {
      combinedHostsEnrolledByOsqueryVersion.push({
        osqueryVersion: osqueryVersion,
        numHosts: _.sum(groupedHostsEnrolledValuesByOsqueryVersion[osqueryVersion], (version)=>{return version.numHosts;})
      });
    }

    for(let version of combinedHostsEnrolledByOsqueryVersion) {
      if(version.osqueryVersion !== ''){
        let metricToAdd = {
          metric: 'usage_statistics_v2.host_count_by_osquery_version',
          type: 3,
          points: [{timestamp: timestampForTheseMetrics, value:version.numHosts}],
          tags: [`osquery_version:${version.osqueryVersion}`],
        };
        // Add the custom metric to the array of metrics to send to Datadog.
        metricsToReport.push(metricToAdd);
      }
    }//∞


    let combinedHostsEnrolledByOrbitVersion = [];
    let allHostsEnrolledByOrbitVersion = _.pluck(latestStatisticsReportedByReleasedFleetVersions, 'hostsEnrolledByOrbitVersion');
    let flattenedHostsEnrolledByOrbitVersions = _.flatten(allHostsEnrolledByOrbitVersion);
    let groupedHostsEnrolledValuesByOrbitVersion = _.groupBy(flattenedHostsEnrolledByOrbitVersions, 'orbitVersion');
    for(let orbitVersion in groupedHostsEnrolledValuesByOrbitVersion) {
      combinedHostsEnrolledByOrbitVersion.push({
        orbitVersion: orbitVersion,
        numHosts: _.sum(groupedHostsEnrolledValuesByOrbitVersion[orbitVersion], (version)=>{return version.numHosts;})
      });
    }
    for(let version of combinedHostsEnrolledByOrbitVersion) {
      if(version.orbitVersion !== '') {
        let metricToAdd = {
          metric: 'usage_statistics_v2.host_count_by_orbit_version',
          type: 3,
          points: [{timestamp: timestampForTheseMetrics, value:version.numHosts}],
          tags: [`orbit_version:${version.orbitVersion}`],
        };
        // Add the custom metric to the array of metrics to send to Datadog.
        metricsToReport.push(metricToAdd);
      }
    }//∞

    // Merge the arrays of JSON storedErrors
    let allStoredErrors = _.pluck(latestStatisticsReportedByReleasedFleetVersions, 'storedErrors');
    let flattenedStoredErrors = _.flatten(allStoredErrors);
    let groupedStoredErrorsByLocation = _.groupBy(flattenedStoredErrors, 'loc');
    let combinedStoredErrors = [];
    for(let location in groupedStoredErrorsByLocation) {
      combinedStoredErrors.push({
        location: groupedStoredErrorsByLocation[location][0].loc,
        count: _.sum(groupedStoredErrorsByLocation[location], (location)=>{return location.count;}),
        numberOfInstancesReportingThisError: groupedStoredErrorsByLocation[location].length
      });
    }
    for(let error of combinedStoredErrors) {
      // Create a new array of tags for this error
      let errorTags = [];
      let errorLocation = 1;
      // Create a tag for each error location
      for(let location of error.location) { // iterate throught the location array of this error
        // Add the error's location as a custom tag (SNAKE_CASED)
        errorTags.push(`error_location_${errorLocation}:${location.replace(/\s/gi, '_')}`);
        errorLocation++;
      }
      // Add a metric with the combined error count for each unique error location
      metricsToReport.push({
        metric: 'usage_statistics_v2.stored_errors_counts',
        type: 3,
        points: [{timestamp: timestampForTheseMetrics, value: error.count}],
        tags: errorTags,
      });
      // Add a metric to report how many different instances reported errors with the same location.
      metricsToReport.push({
        metric: 'usage_statistics_v2.stored_errors_statistics',
        type: 3,
        points: [{timestamp: timestampForTheseMetrics, value: error.numberOfInstancesReportingThisError}],
        tags: errorTags,
      });
    }//∞


    // Build a metric for each Fleet version reported.
    let statisticsByReportedFleetVersion = _.groupBy(latestStatisticsReportedByReleasedFleetVersions, 'fleetVersion');
    for(let version in statisticsByReportedFleetVersion){
      let numberOfInstancesReportingThisVersion = statisticsByReportedFleetVersion[version].length;
      metricsToReport.push({
        metric: 'usage_statistics.fleet_version',
        type: 3,
        points: [{
          timestamp: timestampForTheseMetrics,
          value: numberOfInstancesReportingThisVersion
        }],
        tags: [`fleet_version:${version}`],
      });
    }
    // Build a metric for each license tier reported.
    let statisticsByReportedFleetLicenseTier = _.groupBy(latestStatisticsReportedByReleasedFleetVersions, 'licenseTier');
    for(let tier in statisticsByReportedFleetLicenseTier){
      let numberOfInstancesReportingThisLicenseTier = statisticsByReportedFleetLicenseTier[tier].length;
      metricsToReport.push({
        metric: 'usage_statistics.fleet_license',
        type: 3,
        points: [{
          timestamp: timestampForTheseMetrics,
          value: numberOfInstancesReportingThisLicenseTier
        }],
        tags: [`license_tier:${tier}`],
      });
    }
    // Build aggregated metrics for boolean variables:
    // Software Inventory
    let numberOfInstancesWithSoftwareInventoryEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {softwareInventoryEnabled: true}).length;
    let numberOfInstancesWithSoftwareInventoryDisabled = numberOfInstancesToReport - numberOfInstancesWithSoftwareInventoryEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.software_inventory',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithSoftwareInventoryEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.software_inventory',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithSoftwareInventoryDisabled
      }],
      tags: [`enabled:false`],
    });
    // vulnDetectionEnabled
    let numberOfInstancesWithVulnDetectionEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {vulnDetectionEnabled: true}).length;
    let numberOfInstancesWithVulnDetectionDisabled = numberOfInstancesToReport - numberOfInstancesWithVulnDetectionEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.vuln_detection',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithVulnDetectionEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.vuln_detection',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithVulnDetectionDisabled
      }],
      tags: [`enabled:false`],
    });
    // SystemUsersEnabled
    let numberOfInstancesWithSystemUsersEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {systemUsersEnabled: true}).length;
    let numberOfInstancesWithSystemUsersDisabled = numberOfInstancesToReport - numberOfInstancesWithSystemUsersEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.system_users',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithSystemUsersEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.system_users',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithSystemUsersDisabled
      }],
      tags: [`enabled:false`],
    });
    // hostsStatusWebHookEnabled
    let numberOfInstancesWithHostsStatusWebHookEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {hostsStatusWebHookEnabled: true}).length;
    let numberOfInstancesWithHostsStatusWebHookDisabled = numberOfInstancesToReport - numberOfInstancesWithHostsStatusWebHookEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.host_status_webhook',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithHostsStatusWebHookEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.host_status_webhook',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithHostsStatusWebHookDisabled
      }],
      tags: [`enabled:false`],
    });
    // mdmMacOsEnabled
    let numberOfInstancesWithMdmMacOsEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {mdmMacOsEnabled: true}).length;
    let numberOfInstancesWithMdmMacOsDisabled = numberOfInstancesToReport - numberOfInstancesWithMdmMacOsEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.macos_mdm',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMdmMacOsEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.macos_mdm',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMdmMacOsDisabled
      }],
      tags: [`enabled:false`],
    });
    // mdmWindowsEnabled
    let numberOfInstancesWithMdmWindowsEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {mdmWindowsEnabled: true}).length;
    let numberOfInstancesWithMdmWindowsDisabled = numberOfInstancesToReport - numberOfInstancesWithMdmWindowsEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.windows_mdm',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMdmWindowsEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.windows_mdm',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMdmWindowsDisabled
      }],
      tags: [`enabled:false`],
    });
    // liveQueryDisabled
    let numberOfInstancesWithLiveQueryDisabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {liveQueryDisabled: true}).length;
    let numberOfInstancesWithLiveQueryEnabled = numberOfInstancesToReport - numberOfInstancesWithLiveQueryDisabled;
    metricsToReport.push({
      metric: 'usage_statistics.live_query',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithLiveQueryDisabled
      }],
      tags: [`enabled:false`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.live_query',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithLiveQueryEnabled
      }],
      tags: [`enabled:true`],
    });
    // hostExpiryEnabled
    let numberOfInstancesWithHostExpiryEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {hostExpiryEnabled: true}).length;
    let numberOfInstancesWithHostExpiryDisabled = numberOfInstancesToReport - numberOfInstancesWithHostExpiryEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.host_expiry',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithHostExpiryEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.host_expiry',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithHostExpiryDisabled
      }],
      tags: [`enabled:false`],
    });
    // aiFeaturesDisabled
    let numberOfInstancesWithAiFeaturesDisabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {aiFeaturesDisabled: true}).length;
    let numberOfInstancesWithAiFeaturesEnabled = numberOfInstancesToReport - numberOfInstancesWithAiFeaturesDisabled;
    metricsToReport.push({
      metric: 'usage_statistics.ai_features',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithAiFeaturesEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.ai_features',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithAiFeaturesDisabled
      }],
      tags: [`enabled:false`],
    });
    // maintenanceWindowsEnabled
    let numberOfInstancesWithMaintenanceWindowsEnabled = _.where(latestStatisticsReportedByReleasedFleetVersions, {maintenanceWindowsEnabled: true}).length;
    let numberOfInstancesWithMaintenanceWindowsDisabled = numberOfInstancesToReport - numberOfInstancesWithMaintenanceWindowsEnabled;
    metricsToReport.push({
      metric: 'usage_statistics.maintenance_windows',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMaintenanceWindowsEnabled
      }],
      tags: [`enabled:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.maintenance_windows',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMaintenanceWindowsDisabled
      }],
      tags: [`enabled:false`],
    });
    // maintenanceWindowsConfigured
    let numberOfInstancesWithMaintenanceWindowsConfigured = _.where(latestStatisticsReportedByReleasedFleetVersions, {maintenanceWindowsEnabled: true}).length;
    let numberOfInstancesWithoutMaintenanceWindowsConfigured = numberOfInstancesToReport - numberOfInstancesWithMaintenanceWindowsConfigured;
    metricsToReport.push({
      metric: 'usage_statistics.maintenance_windows_configured',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithMaintenanceWindowsConfigured
      }],
      tags: [`configured:true`],
    });
    metricsToReport.push({
      metric: 'usage_statistics.maintenance_windows_configured',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: numberOfInstancesWithoutMaintenanceWindowsConfigured
      }],
      tags: [`configured:false`],
    });

    // Create two metrics to track total number of hosts reported in the last week.
    let totalNumberOfHostsReportedByPremiumInstancesInTheLastWeek = _.sum(_.pluck(_.filter(latestStatisticsReportedByReleasedFleetVersions, {licenseTier: 'premium'}), 'numHostsEnrolled'));
    metricsToReport.push({
      metric: 'usage_statistics.total_num_hosts_enrolled',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: totalNumberOfHostsReportedByPremiumInstancesInTheLastWeek
      }],
      tags: [`license_tier:premium`],
    });

    let totalNumberOfHostsReportedByFreeInstancesInTheLastWeek = _.sum(_.pluck(_.filter(latestStatisticsReportedByReleasedFleetVersions, {licenseTier: 'free'}), 'numHostsEnrolled'));
    metricsToReport.push({
      metric: 'usage_statistics.total_num_hosts_enrolled',
      type: 3,
      points: [{
        timestamp: timestampForTheseMetrics,
        value: totalNumberOfHostsReportedByFreeInstancesInTheLastWeek
      }],
      tags: [`license_tier:free`],
    });

    // FUTURE: Uncomment the section below to send metrics about reported number of queries to Datadog.
    // let fleetInstancesThatReportedNumQueries = _.filter(latestStatisticsReportedByReleasedFleetVersions, (statistics)=>{
    //   return statistics.numQueries > 0;
    // });

    // let averageNumberOfQueries = Math.foor(_.sum(_.pluck(fleetInstancesThatReportedNumQueries, 'numQueries')) / fleetInstancesThatReportedNumQueries.length);
    // metricsToReport.push({
    //   metric: 'usage_statistics.avg_num_queries',
    //   type: 3,
    //   points: [{
    //     timestamp: timestampForTheseMetrics,
    //     value: averageNumberOfQueries
    //   }],
    // });

    // let highestNumberOfQueries = _.max(_.pluck(fleetInstancesThatReportedNumQueries, 'numQueries'));
    // metricsToReport.push({
    //   metric: 'usage_statistics.max_num_queries',
    //   type: 3,
    //   points: [{
    //     timestamp: timestampForTheseMetrics,
    //     value: highestNumberOfQueries
    //   }],
    // });

    // Break the metrics into smaller arrays to ensure we don't exceed Datadog's 512 kb request body limit.
    let chunkedMetrics = _.chunk(metricsToReport, 500);// Note: 500 stringified JSON metrics is ~410 kb.
    for(let chunkOfMetrics of chunkedMetrics) {
      await sails.helpers.http.post.with({
        url: 'https://api.us5.datadoghq.com/api/v2/series',
        data: {
          series: chunkOfMetrics,
        },
        headers: {
          'DD-API-KEY': sails.config.custom.datadogApiKey,
          'Content-Type': 'application/json',
        }
      }).intercept((err)=>{
        // If there was an error sending metrics to Datadog, we'll log the error in a warning, but we won't throw an error.
        // This way, we'll still return a 200 status to the Fleet instance that sent usage analytics.
        return new Error(`When the send-metrics-to-datadog script sent a request to send metrics to Datadog, an error occured. Raw error: ${require('util').inspect(err)}`);
      });
    }//∞
    sails.log(`Aggregated metrics for ${numberOfInstancesToReport} Fleet instances from the past week sent to Datadog.`);
  }


};

