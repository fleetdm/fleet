module.exports = {


  friendlyName: 'View software',


  description: 'Display "Software" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/software/software'
    }

  },


  fn: async function () {
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
    let undeployedSoftware = await UndeployedSoftware.find().omit(['softwareContents']);

    return {software: undeployedSoftware, teams};

  }


};
