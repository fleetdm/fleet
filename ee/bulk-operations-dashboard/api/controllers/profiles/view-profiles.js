module.exports = {


  friendlyName: 'View profiles',


  description: 'Display "Profiles" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/profiles'
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
    let teamsInformation = [];
    for(let team of allTeams) {
      teams.push({
        fleetApid: team.id,
        teamName: team.name,
      });
    }

    // console.log(teamApids);
    let allProfiles = [];

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

    // Grab all of the configuration profiles on the Fleet instance.
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

    // console.log(allProfiles);

    let profileConfiguration = [];

    let allProfilesByIdentifier = _.groupBy(allProfiles, 'identifier');

    for(let profileIdentifier in allProfilesByIdentifier) {
      let teamIdsForThisProfile = _.pluck(allProfilesByIdentifier[profileIdentifier], 'team_id');
      let teamsForThisProfile = [];
      // console.log(teamsForThisProfile);
      // let platforms = _.uniq(_.pluck(allProfilesByIdentifier[profileIdentifier], 'platform'));
      for(let profile of allProfilesByIdentifier[profileIdentifier]){
        let informationAboutThisProfile = {
          uuid: profile.profile_uuid,
          fleetApid: profile.team_id,
          teamName: _.find(allTeams, {id: profile.team_id}).name,
        }
        teamsForThisProfile.push(informationAboutThisProfile);
      }
      let profile = allProfilesByIdentifier[profileIdentifier][0];// Grab the first profile returned in the api repsonse to build our profile configuration.
      let profileInformation = {
        name: profile.name,
        identifier: profileIdentifier,
        platform: profile.platform,
        createdAt: new Date(profile.created_at).getTime(),
        teams: teamsForThisProfile
      }
      profileConfiguration.push(profileInformation)
    }
    profileConfiguration = _.sortByOrder(profileConfiguration, 'name', 'asc');

    console.log(profileConfiguration);
    // Respond with view.
    return {profiles: profileConfiguration, teams};

  }


};
