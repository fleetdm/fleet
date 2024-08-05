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

    let allProfilesByName = _.groupBy(allProfiles, 'name');
    console.log(allProfilesByName)

    for(let profileUuid in allProfilesByName) {
      let teamsForThisProfile = _.pluck(allProfilesByName[profileUuid], 'team_id');
      console.log(teamsForThisProfile);
      let profile = allProfilesByName[profileUuid][0];// Grab the first profile returned in the api repsonse to build our profile configuration.
      let profileInformation = {
        name: profile.name,
        platform: profile.platform,
        createdAt: new Date(profile.created_at).getTime(),
        uuid: profileUuid,
        teams: teamsForThisProfile,
      }
      profileConfiguration.push(profileInformation)
    }

    // console.log(profileConfiguration);



    // Respond with view.
    return {profileConfiguration};

  }


};
