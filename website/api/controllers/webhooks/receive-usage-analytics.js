module.exports = {


  friendlyName: 'Receive usage analytics',


  description: 'Receive anonymous usage analytics from deployments of Fleet running in production.  (Not fleetctl preview or dev-mode deployments.)',


  inputs: {
    anonymousIdentifier: { required: true, type: 'string', example: '9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=', description: 'An anonymous identifier telling us which Fleet deployment this is.', },
    fleetVersion: { required: true, type: 'string', example: 'x.x.x' },
    licenseTier: { type: 'string', isIn: ['free', 'premium', 'unknown'], defaultsTo: 'unknown' },
    numHostsEnrolled: { required: true, type: 'number', min: 0, custom: (num) => Math.floor(num) === num },
    numUsers: { type: 'number', defaultsTo: 0 },
    numTeams: { type: 'number', defaultsTo: 0 },
    numPolicies: { type: 'number', defaultsTo: 0 },
    numLabels: { type: 'number', defaultsTo: 0 },
    softwareInventoryEnabled: { type: 'boolean', defaultsTo: false },
    vulnDetectionEnabled: { type: 'boolean', defaultsTo: false },
    systemUsersEnabled: { type: 'boolean', defaultsTo: false },
    hostsStatusWebHookEnabled: { type: 'boolean', defaultsTo: false },
    numWeeklyActiveUsers: { type: 'number', defaultsTo: 0 },
    numWeeklyPolicyViolationDaysActual: { type: 'number', defaultsTo: 0 },
    numWeeklyPolicyViolationDaysPossible: { type: 'number', defaultsTo: 0 },
    hostsEnrolledByOperatingSystem: { type: {}, defaultsTo: {} },
    hostsEnrolledByOrbitVersion: { type: [{orbitVersion: 'string', numHosts: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    hostsEnrolledByOsqueryVersion: { type: [{osqueryVersion: 'string', numHosts: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    storedErrors: { type: [{}], defaultsTo: [] }, // TODO migrate all rows that have "[]" to {}
    numHostsNotResponding: { type: 'number', defaultsTo: 0, description: 'The number of hosts per deployment that have not submitted results for distibuted queries. A host is counted as not responding if Fleet hasn\'t received a distributed write to requested distibuted queries for the host during the 2-hour interval since the host was last seen. Hosts that have not been seen for 7 days or more are not counted.', },
    organization: { type: 'string', defaultsTo: 'unknown', description: 'For Fleet Premium deployments, the organization registered with the license.', },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {

    // Create a database record for these usage statistics
    await HistoricalUsageSnapshot.create(inputs);

    if(!sails.config.custom.datadogApiKey) {
      throw new Error('No Datadog API key configured! (Please set sails.config.custom.datadogApiKey)');
    }

    // Store strings and booleans as tags.
    let baseMetricTags = [
      `fleet_version:${inputs.fleetVersion}`,
      `license_tier:${inputs.licenseTier}`,
      `software_inventory_enabled:${inputs.softwareInventoryEnabled}`,
      `vuln_detection_enabled:${inputs.vulnDetectionEnabled}`,
      `system_users_enabled:${inputs.systemUsersEnabled}`,
      `host_status_webhook_enabled:${inputs.hostsStatusWebHookEnabled}`,
    ];

    // Create a timestamp in seconds for these metrics
    let metricsTimestampInSeconds = Math.floor(Date.now() / 1000);

    // Build metrics for the usagle statistics that are numbers
    let metricsToSendToDatadog = [
      {
        metric: 'usage_statistics.fleet_server_stats',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: 1
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_hosts_enrolled',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numHostsEnrolled
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_users',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numUsers
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_teams',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numTeams
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_policies',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numPolicies
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_labels',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numLabels
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_weekly_active_users',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numWeeklyActiveUsers
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_weekly_policy_violation_days_actual',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numWeeklyPolicyViolationDaysActual
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_weekly_policy_violation_days_possible',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numWeeklyPolicyViolationDaysPossible
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
      {
        metric: 'usage_statistics.num_hosts_not_responding',
        type: 3,
        points: [{
          timestamp: metricsTimestampInSeconds,
          value: inputs.numHostsNotResponding
        }],
        resources: [{
          name: inputs.anonymousIdentifier,
          type: 'fleet_instance'
        }],
        tags: baseMetricTags,
      },
    ];

    // Build metrics for logged errors
    if(inputs.storedErrors.length > 0) {
      // If inputs.storedErrors is not an empty array, we'll iterate through it to build custom metric for each object in the array
      for(let error of inputs.storedErrors) {
        // Make sure every error object in the storedErrors array has a 'loc' array and a count.
        if((error.loc && _.isArray(error.loc)) && error.count) {
          // Create a new array of tags for this error
          let errorTags = _.clone(baseMetricTags);
          let errorLocation = 1;
          // Create a tag for each error location
          for(let location of error.loc) { // iterate throught the location array of this error
            // Add the error's location as a custom tag (SNAKE_CASED)
            errorTags.push(`error_location_${errorLocation}:${location.replace(/\s/gi, '_')}`);
            errorLocation++;
          }
          let metricToAdd = {
            metric: 'usage_statistics.stored_errors',
            type: 3,
            points: [{timestamp: metricsTimestampInSeconds, value: error.count}],
            resources: [{name: inputs.anonymousIdentifier, type: 'fleet_instance'}],
            tags: errorTags,
          };
          // Add the custom metric to the array of metrics to send to Datadog.
          metricsToSendToDatadog.push(metricToAdd);
        }//ﬁ
      }//∞
    }//ﬁ


    // If inputs.hostsEnrolledByOrbitVersion is not an empty array, we'll iterate through it to build custom metric for each object in the array
    if(inputs.hostsEnrolledByOrbitVersion.length > 0) {
      for(let version of inputs.hostsEnrolledByOrbitVersion) {
        let orbitVersionTags = _.clone(baseMetricTags);
        orbitVersionTags.push(`orbit_version:${version.orbitVersion}`);
        let metricToAdd = {
          metric: 'usage_statistics.host_count_by_orbit_version',
          type: 3,
          points: [{timestamp: metricsTimestampInSeconds, value:version.numHosts}],
          resources: [{name: inputs.anonymousIdentifier, type: 'fleet_instance'}],
          tags: orbitVersionTags,
        };
        // Add the custom metric to the array of metrics to send to Datadog.
        metricsToSendToDatadog.push(metricToAdd);
      }//∞
    }//ﬁ

    // If inputs.hostsEnrolledByOsqueryVersion is not an empty array, we'll iterate through it to build custom metric for each object in the array
    if(inputs.hostsEnrolledByOsqueryVersion.length > 0) {
      for(let version of inputs.hostsEnrolledByOsqueryVersion) {
        let osqueryVersionTags = _.clone(baseMetricTags);
        osqueryVersionTags.push(`osquery_version:${version.osqueryVersion}`);
        let metricToAdd = {
          metric: 'usage_statistics.host_count_by_osquery_version',
          type: 3,
          points: [{timestamp: metricsTimestampInSeconds, value:version.numHosts}],
          resources: [{name: inputs.anonymousIdentifier, type: 'fleet_instance'}],
          tags: osqueryVersionTags,
        };
        // Add the custom metric to the array of metrics to send to Datadog.
        metricsToSendToDatadog.push(metricToAdd);
      }//∞
    }//ﬁ

    // If the hostByOperatingSystem is not an empty object, we'll iterate through the object to build metrics for each type of operating system.
    // See https://fleetdm.com/docs/using-fleet/usage-statistics#what-is-included-in-usage-statistics-in-fleet to see an example of a hostByOperatingSystem send by Fleet instances.
    if(_.keys(inputs.hostsEnrolledByOperatingSystem).length > 0) {
      // Iterate through each array of objects
      for(let operatingSystem in inputs.hostsEnrolledByOperatingSystem) {
        // For every object in the array, we'll send a metric to track host count for each operating system version.
        for(let osVersion of inputs.hostsEnrolledByOperatingSystem[operatingSystem]) {
          // Only continue if the object in the array has a numEnrolled and version value.
          if(osVersion.numEnrolled && osVersion.version) {
            // Clone the baseMetricTags array, each metric will have the operating version name added as a `os_version_name` tag
            let osInfoTags = _.clone(baseMetricTags);
            osInfoTags.push(`os_version_name:${osVersion.version}`);
            let metricToAdd = {
              metric: 'usage_statistics.host_count_by_os_version',
              type: 3,
              points: [{timestamp: metricsTimestampInSeconds, value:osVersion.numEnrolled}],
              resources: [{name: operatingSystem, type: 'os_type'}],
              tags: osInfoTags,
            };
            // Add the custom metric to the array of metrics to send to Datadog.
            metricsToSendToDatadog.push(metricToAdd);
          }//ﬁ
        }//∞
      }//∞
    }//ﬁ

    await sails.helpers.http.post.with({
      url: 'https://api.us5.datadoghq.com/api/v2/series',
      data: {
        series: metricsToSendToDatadog,
      },
      headers: {
        'DD-API-KEY': sails.config.custom.datadogApiKey,
        'Content-Type': 'application/json',
      }
    }).tolerate((err)=>{
      // If there was an error sending metrics to Datadog, we'll log the error in a warning, but we won't throw an error.
      // This way, we'll still return a 200 status to the Fleet instance that sent usage analytics.
      sails.log.warn(`When the receive-usage-analytics webhook tried to send metrics to Datadog, an error occured. Raw error: ${require('util').inspect(err)}`);
    });



  }


};
