module.exports = {


  friendlyName: 'View profiles',


  description: 'Display "Profiles" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/profiles'
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


    let allProfiles = [];
    let teamApids = _.pluck(allTeams, 'id');
    // Get all of the configuration profiles on the Fleet instance.
    for(let teamApid of teamApids){
      let configurationProfilesResponseData = await sails.helpers.http.get.with({
        url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
        baseUrl: sails.config.custom.fleetBaseUrl,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      })
      .timeout(120000)
      .retry(['requestFailed', {name: 'TimeoutError'}]);
      let profilesForThisTeam = configurationProfilesResponseData.profiles;
      allProfiles = allProfiles.concat(profilesForThisTeam);
    }
    // Add the configurations profiles that are assigned to the "no team" team.
    let noTeamConfigurationProfilesResponseData = await sails.helpers.http.get.with({
      url: '/api/v1/fleet/configuration_profiles',
      baseUrl: sails.config.custom.fleetBaseUrl,
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
      }
    })
    .timeout(120000)
    .retry(['requestFailed', {name: 'TimeoutError'}]);
    let profilesForThisTeam = noTeamConfigurationProfilesResponseData.profiles;
    allProfiles = allProfiles.concat(profilesForThisTeam);
    let profilesInformation = [];
    // Group configuration profiles by their identifier.
    let allProfilesByIdentifier = _.groupBy(allProfiles, 'identifier');
    for(let profileIdentifier in allProfilesByIdentifier) {
      // Iterate through the arrays of profiles with the same unique identifier.
      let teamsForThisProfile = [];
      // Add the profile's UUID and information about the team this profile is assigned to the teams array for profiles.
      for(let profile of allProfilesByIdentifier[profileIdentifier]) {
        let informationAboutThisProfile = {
          uuid: profile.profile_uuid,
          fleetApid: profile.team_id,
          teamName: _.find(teams, {fleetApid: profile.team_id}).teamName,
        };
        teamsForThisProfile.push(informationAboutThisProfile);
      }
      let profile = allProfilesByIdentifier[profileIdentifier][0];// Grab the first profile returned in the api repsonse to build our profile configuration.
      let profileInformation = {
        name: profile.name,
        identifier: profileIdentifier,
        platform: profile.platform,
        createdAt: new Date(profile.created_at).getTime(),
        teams: teamsForThisProfile
      };
      profilesInformation.push(profileInformation);
    }
    // Get the undeployed profiles from the app's database.
    let undeployedProfiles = await UndeployedProfile.find();
    profilesInformation = _.union(profilesInformation, undeployedProfiles);

    // Sort profiles by their name.
    profilesInformation = _.sortByOrder(profilesInformation, 'name', 'asc');

    // Respond with view.
    return {profiles: profilesInformation, teams};

  }


};
