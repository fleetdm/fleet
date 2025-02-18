module.exports = {


  friendlyName: 'Get scripts',


  description: 'Builds and returns an array of deployed scripts on the Fleet instance and undeployed scripts stored in the dashboard\'s datastore',


  exits: {
    success: {
      outputType: [{}],
    }
  },


  fn: async function () {
    // Get all teams on the Fleet instance.
    let teamsResponseData = await sails.helpers.http.get.with({
      url: '/api/v1/fleet/teams',
      baseUrl: sails.config.custom.fleetBaseUrl,
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
      }
    })
    .timeout(120000)
    .retry(['requestFailed', {name: 'TimeoutError'}]);

    let allTeams = teamsResponseData.teams;

    let teamApids = _.pluck(allTeams, 'id');
    let teams = [];
    for(let team of allTeams) {
      teams.push({
        fleetApid: team.id,
        teamName: team.name,
      });
    }
    // Add the "team" for hosts with no team
    teams.push({
      fleetApid: 0,
      teamName: 'No team',
    });


    let allScripts = [];
    // Get all of the scripts on a Fleet instance.
    for(let teamApid of teamApids){
      let scriptsResponseData = await sails.helpers.http.get.with({
        url: `/api/v1/fleet/scripts?team_id=${teamApid}`,
        baseUrl: sails.config.custom.fleetBaseUrl,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      })
      .timeout(120000)
      .retry(['requestFailed', {name: 'TimeoutError'}]);
      let scriptsForThisTeam = scriptsResponseData.scripts;
      if(scriptsForThisTeam !== null) {
        allScripts = allScripts.concat(scriptsForThisTeam);
      }
    }

    // Grab all of the configuration scripts on the Fleet instance.
    let noTeamConfigurationScriptsResponseData = await sails.helpers.http.get.with({
      url: '/api/v1/fleet/scripts',
      baseUrl: sails.config.custom.fleetBaseUrl,
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
      }
    })
    .timeout(120000)
    .retry(['requestFailed', {name: 'TimeoutError'}]);
    let scriptsForThisTeam = noTeamConfigurationScriptsResponseData.scripts;

    if(scriptsForThisTeam !== null){
      allScripts = allScripts.concat(scriptsForThisTeam);
    }

    // If there are no scripts on the Fleet instance, return an empty array and the teams information.
    if(allScripts === [ null ]){
      return {scripts: [], teams};
    }
    let scriptsOnThisFleetInstance = [];

    let allScriptsByIdentifier = _.groupBy(allScripts, 'name');
    for(let scriptIdentifier in allScriptsByIdentifier) {
      if(scriptIdentifier === null){
        continue;
      }
      let teamsForThisProfile = [];
      for(let script of allScriptsByIdentifier[scriptIdentifier]){
        let informationAboutThisScript = {
          scriptFleetApid: script.id,
          fleetApid: script.team_id ? script.team_id : 0,
          teamName: script.team_id ? _.find(teams, {fleetApid: script.team_id}).teamName : 'No team',
        };
        teamsForThisProfile.push(informationAboutThisScript);
      }
      let script = allScriptsByIdentifier[scriptIdentifier][0];// Grab the first script returned in the api repsonse to build our script configuration.
      let scriptInformation = {
        name: script.name,
        identifier: scriptIdentifier,
        platform: _.endsWith(script.name, 'sh') ? 'macOS & Linux' : 'Windows',
        createdAt: new Date(script.created_at).getTime(),
        teams: teamsForThisProfile
      };
      scriptsOnThisFleetInstance.push(scriptInformation);
    }
    // Get the undeployed scripts from the app's database.
    let undeployedScripts = await UndeployedScript.find();
    scriptsOnThisFleetInstance = _.union(scriptsOnThisFleetInstance, undeployedScripts);
    // Sort scripts by their name.
    scriptsOnThisFleetInstance = _.sortByOrder(scriptsOnThisFleetInstance, 'name', 'asc');
    return scriptsOnThisFleetInstance;

  }


};
