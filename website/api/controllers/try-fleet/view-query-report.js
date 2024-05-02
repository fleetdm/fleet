module.exports = {


  friendlyName: 'View query report',


  description: 'Display "Query report" page.',

  inputs: {

    hostPlatform: {
      type: 'string',
      required: true,
      description: 'The platform of the host to display results for',
      extendedDescription: '',
      isIn: ['macos', 'linux', 'windows']
    },

    tableName: {
      type: 'string',
      required: true,
      description: 'The name of the osquery table to show results for.',
    },

  },


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/query-report'
    },

    badConfig: {
      responseType: 'badConfig'
    },

    redirect: {
      description: 'The requesting user is not logged in.',
      responseType: 'redirect'
    },

    invalidTable: {
      responseType: 'notFound',
      description: 'No osquery table with the specified name could be found.'
    },

  },


  fn: async function ({hostPlatform, tableName}) {

    if(!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.osqueryTables)){
      throw {badConfig: 'builtStaticContent.osqueryTables'};
    }

    // If the requesting user is not logged in, redirect them to the /try-fleet/register page with the specified hostPlatform added as a query parameter.
    if(!this.req.me){
      throw {redirect: `/register?targetPlatform=${encodeURIComponent(hostPlatform)}` };
    }

    if(!sails.config.custom.queryIdsByTableName){
      throw new Error('Missing config variable: The dictionary of query ids required to use the query-report page is missing! (sails.config.custom.queryIdsByTableName)');
    }

    if(!sails.config.custom.hostIdsByHostPlatform){
      throw new Error('Missing config variable: The dictionary of host ids required to use the query-report page is missing! (sails.config.custom.hostIdsByHostPlatform)');
    }

    if(!sails.config.custom.teamApidForQueryReports){
      throw new Error('Missing config variable: The id of the team the query report page gets results for is missing! (sails.config.custom.teamApidForQueryReports)');
    }

    if(!sails.config.custom.fleetBaseUrlForQueryReports){
      throw new Error('Missing config variable: The URL of the fleet instance used for query reports is missing! (sails.config.custom.fleetBaseUrlForQueryReports)');
    }

    if(!sails.config.custom.fleetTokenForQueryReports){
      throw new Error('Missing config variable: The API token for requests to the Fleet instance used for queyr reports is missing! (sails.config.custom.fleetTokenForQueryReports)');
    }

    //  ┬ ┬┌─┐┌─┐┌┬┐  ┬┌┐┌┌─┐┌─┐┬─┐┌┬┐┌─┐┌┬┐┬┌─┐┌┐┌
    //  ├─┤│ │└─┐ │   ││││├┤ │ │├┬┘│││├─┤ │ ││ ││││
    //  ┴ ┴└─┘└─┘ ┴   ┴┘└┘└  └─┘┴└─┴ ┴┴ ┴ ┴ ┴└─┘┘└┘
    let hostIdsByHostPlatform = sails.config.custom.hostIdsByHostPlatform;
    // Get the ID of the host we'll be showing results for.
    let selectedHostId = hostIdsByHostPlatform[hostPlatform];

    // Send an HTTP request to get the host details for hosts on the query report team.
    let hostsOnQueryReportTeamApiResponse = await sails.helpers.http.get.with({
      url: sails.config.custom.fleetBaseUrlForQueryReports+'/api/v1/fleet/hosts?team_id='+encodeURIComponent(sails.config.custom.teamApidForQueryReports),
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetTokenForQueryReports}`
      }
    })
    .intercept((error)=>{
      return new Error(`When sending an API request to ${sails.config.custom.fleetBaseUrlForQueryReports}/api/v1/fleet/hosts?team_id=${sails.config.custom.teamApidForQueryReports} to get information about hosts on the query report team, an error occured: ${error.stack}`);
    });
    if(hostsOnQueryReportTeamApiResponse.hosts.length < 1) {
      throw new Error(`Error! When view-query-report sent a request to ${sails.config.custom.fleetBaseUrlForQueryReports} to get information about the hosts on the query reports team, the API response contained no hosts.`);
    }

    let hostsOnTheQueryReportTeam = hostsOnQueryReportTeamApiResponse.hosts;
    let hostsAvailableToQuery = [];

    // Get information about these hosts for the host selector dropdown.
    for(let host of hostsOnTheQueryReportTeam) {
      let hostInfoForDropdownSelector = {
        name: host.hostname,
        platform: undefined,
      };
      if(host.platform === 'windows'){
        hostInfoForDropdownSelector.platform = 'Windows';
      } else if(host.platform === 'darwin'){
        hostInfoForDropdownSelector.platform = 'macOS';
      } else {
        hostInfoForDropdownSelector.platform = 'Linux';
      }
      hostsAvailableToQuery.push(hostInfoForDropdownSelector);
    }

    // Get the host from the host response
    let hostToGetReportFor = _.find(hostsOnTheQueryReportTeam, {'id': selectedHostId});
    // Convert the host's memory from bytes into GB.
    let hostsMemoryInGb = hostToGetReportFor.memory / (1024 * 1024 * 1024);

    // If the host's memory is not a whole number of GB, we'll show the first two decimal places.
    if(Math.floor(hostsMemoryInGb) !== hostsMemoryInGb){
      hostsMemoryInGb = hostsMemoryInGb.toFixed(2);
    }
    // Build a dictionary containing information about this host.
    let hostDetails = {
      os: hostToGetReportFor.os_version,
      hardwareType: hostToGetReportFor.hardware_model,
      memory: hostsMemoryInGb+'GB',
      processor: hostToGetReportFor.cpu_type,
      osqueryVersion: hostToGetReportFor.osquery_version,
      name: hostToGetReportFor.hostname,
    };

    //  ┌─┐┌─┐┌─┐ ┬ ┬┌─┐┬─┐┬ ┬  ┌┬┐┌─┐┌┐ ┬  ┌─┐┌─┐
    //  │ │└─┐│─┼┐│ │├┤ ├┬┘└┬┘   │ ├─┤├┴┐│  ├┤ └─┐
    //  └─┘└─┘└─┘└└─┘└─┘┴└─ ┴    ┴ ┴ ┴└─┘┴─┘└─┘└─┘

    // Get the IDs of the queries for this team.
    let queryIdsByTableName = sails.config.custom.queryIdsByTableName;

    // Build an array of osquery tables to display,
    let osqueryTablesToDisplay = [];
    // Only show tables that are compatible with the hosts platform, and that have query ids associated with them in the queryIdsByTableName dictionary.
    // This is so when new tables are added, they will only be displayed if they have a query associated with them.
    if(hostPlatform === 'macos'){
      osqueryTablesToDisplay = _.filter(sails.config.builtStaticContent.osqueryTables, (table)=>{
        return _.contains(table.platforms, 'darwin') && queryIdsByTableName[`${table.name}`] !== undefined;
      });
    } else if(hostPlatform === 'linux'){
      osqueryTablesToDisplay = _.filter(sails.config.builtStaticContent.osqueryTables, (table)=>{
        return _.contains(table.platforms, 'linux') && queryIdsByTableName[`${table.name}`] !== undefined;
      });
    } else if(hostPlatform === 'windows'){
      osqueryTablesToDisplay = _.filter(sails.config.builtStaticContent.osqueryTables, (table)=>{
        return _.contains(table.platforms, 'windows') && queryIdsByTableName[`${table.name}`] !== undefined;
      });
    }

    // If the specified table does not exist, or is not compatible with the selected host
    if(!_.contains(_.pluck(osqueryTablesToDisplay, 'name'), tableName)){
      throw 'invalidTable';
    }
    let specifiedOsqueryTable = _.find(osqueryTablesToDisplay, {'name': tableName});

    //  ┌─┐ ┬ ┬┌─┐┬─┐┬ ┬  ┬─┐┌─┐┌─┐┬ ┬┬ ┌┬┐┌─┐
    //  │─┼┐│ │├┤ ├┬┘└┬┘  ├┬┘├┤ └─┐│ ││  │ └─┐
    //  └─┘└└─┘└─┘┴└─ ┴   ┴└─└─┘└─┘└─┘┴─┘┴ └─┘

    let queryIdToGetReportFor = queryIdsByTableName[`${tableName}`];
    // Send an HTTP request to get the query report for the query for this table.
    let queryReportResponse = await sails.helpers.http.get.with({
      url: sails.config.custom.fleetBaseUrlForQueryReports+'/api/v1/fleet/queries/'+encodeURIComponent(queryIdToGetReportFor)+'/report',
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetTokenForQueryReports}`
      }
    })
    .intercept((error)=>{
      return new Error(`When sending an API request to ${sails.config.custom.fleetBaseUrlForQueryReports}/api/v1/fleet/queries/${queryIdToGetReportFor}/report to get the latest query report for the ${tableName} table, an error occured: ${error.stack}`);
    });

    let queryResults = queryReportResponse.results;
    // Group the query results by the host that reported them.
    let queryResultsByHostIds = _.groupBy(queryResults, 'host_id');
    // Default these to empty arrays, if there are no results for this host, we'll send an empty array of results array to the page and show the user an empty state.
    let reportForThisHost = [];
    let reportWithSortedColumns = [];
    let topResultLastFetchedAt = 0;

    // Process the query results for this host (If there are any)
    if(queryResultsByHostIds[selectedHostId]) {
      let resultsForThisHost = queryResultsByHostIds[selectedHostId];
      // Sort the results by their last fetched value.
      let resultsOrderedByLastFetched = _.sortByOrder(resultsForThisHost, 'last_fetched');
      // Get a timestamp of when the last result was fetched from the host.
      topResultLastFetchedAt = Date.parse(resultsOrderedByLastFetched[0].last_fetched);
      // Get an array of the every columns dictionary in the results of this host.
      let unsortedReportForThisHost = _.pluck(resultsOrderedByLastFetched, 'columns');
      // Iterate through the results to sort the columns by their order in the osquery schema.
      for(let result of unsortedReportForThisHost) {
        let sortedColumns = {};
        // Reorder the results by the order of the columns in hte osquery schema, and add the new sorted dictionary to the reportWithSortedColumns array.
        specifiedOsqueryTable.columns.forEach(column => {

          if (result[column.name] !== undefined) {
            sortedColumns[column.name] = result[column.name];
          }
        });
        reportWithSortedColumns.push(sortedColumns);
      }
      // Break the results into smaller arrays with 20 values each for table pagination
      reportForThisHost = _.chunk(reportWithSortedColumns, 20);
    }


    // Respond with view.
    return {
      lastFetchedAt: topResultLastFetchedAt,
      queryReportPages: reportForThisHost,
      osqueryTables: osqueryTablesToDisplay,
      hostPlatform,
      tableName,
      osqueryTableInfo: specifiedOsqueryTable,
      hostDetails, hostsAvailableToQuery,
    };

  }


};
