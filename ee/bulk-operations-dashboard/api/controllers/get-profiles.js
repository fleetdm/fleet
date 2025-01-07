module.exports = {


  friendlyName: 'Get profiles',


  description: 'Builds and returns an array of deployed configuration profiles on the Fleet instance and undeployed profiles stored in the dashboard\'s datastore.',

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

    let teams = [];
    let teams = allTeams.map((team)=>{
      return {
        fleetApid: team.id,
        teamName: team.name
      };
    });
    // Add the "team" for hosts with no team
    teams.push({
      fleetApid: 0,
      teamName: 'No team',
    });


    let allProfiles = [];
    let teamApids = _.pluck(allTeams, 'id');
    // Get all of the configuration profiles on the Fleet instance.
    await sails.helpers.flow.simultaneouslyForEach(teamApids, async (teamApid)=>{
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
    });
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
    let profilesForNoTeam = noTeamConfigurationProfilesResponseData.profiles;
    allProfiles = allProfiles.concat(profilesForNoTeam);
    let profilesInformation = [];
    await sails.helpers.flow.simultaneouslyForEach(allProfiles, async (profile)=>{
      let profileInformation = {
        name: profile.name,
        identifier: profile.identifier,
        platform: profile.platform,
        createdAt: new Date(profile.created_at).getTime(),
        team: {
          uuid: profile.profile_uuid,
          fleetApid: profile.team_id,
          teamName: _.find(teams, {fleetApid: profile.team_id}).teamName,
        },
        profileTarget: 'all',
      };
      if(profile.labels_include_all) {
        profileInformation.labels = _.pluck(profile.labels_include_all, 'name');
        profileInformation.profileTarget = 'custom';
        profileInformation.labelTargetBehavior = 'include';
      } else if(profile.labels_exclude_any){
        profileInformation.labels = _.pluck(profile.labels_exclude_any, 'name');
        profileInformation.profileTarget = 'custom';
        profileInformation.labelTargetBehavior = 'exclude';
      }
      profilesInformation.push(profileInformation);
    });
    // Group the profiles based on identifier, labels, and labelTargetBehavior
    let profilesGroupedbyLabelsAndIdentifier = _.groupBy(profilesInformation, (profile)=>{
      return `${profile.identifier}|${JSON.stringify(profile.labels)}|${profile.labelTargetBehavior}`;
    });

    // map the grouped profiles and merge profiles that have the same labels, target behavior, and identifier.
    let allProfilesOnFleetInstance = Object.values(profilesGroupedbyLabelsAndIdentifier).map(profileGroup => {
      return {
        ...profileGroup[0],// Expand the first item in the profileGroup
        teams: profileGroup.map(item => item.team)// Merge the teams arrays
      };
    });
    // Get the undeployed profiles from the app's database.
    let undeployedProfiles = await UndeployedProfile.find();
    allProfilesOnFleetInstance = _.union(allProfilesOnFleetInstance, undeployedProfiles);
    // Sort profiles by their name.
    allProfilesOnFleetInstance = _.sortByOrder(allProfilesOnFleetInstance, 'name', 'asc');

    // return the updated list of profiles
    return allProfilesOnFleetInstance;
  }


};
