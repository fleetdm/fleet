module.exports = {


  friendlyName: 'Get software',


  description: 'Builds and returns an array of deployed software installers on the Fleet instance and undeployed software stored in the dashboard\'s datastore.',


  exits: {
    success: {
      outputType: [{}],
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

    let allSoftware = [];

    let allSoftwareWithPackages = [];
    let teamsinformationForSoftware = [];

    let softwareDeployedToAllTeams = await AllTeamsSoftware.find();
    let allTeamsSoftwareApids = _.pluck(softwareDeployedToAllTeams, 'fleetApid');
    let teamApids = _.pluck(teams, 'fleetApid');
    // Get all of the software packages on the Fleet instance.
    let batchesOfTeamIds = _.chunk(teamApids, 5);
    for(let batchOfTeamIds of batchesOfTeamIds){
      await sails.helpers.flow.simultaneouslyForEach(batchOfTeamIds, async(teamApid)=>{
      let configurationProfilesResponseData = await sails.helpers.http.get.with({
        url: `/api/latest/fleet/software/titles?team_id=${teamApid}`,
        baseUrl: sails.config.custom.fleetBaseUrl,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      })
      .timeout(120000)
      .retry(['requestFailed', {name: 'TimeoutError'}]);
      let softwareForThisTeam = configurationProfilesResponseData.software_titles;

      let softwareWithSoftwarePackages = _.filter(softwareForThisTeam, (software)=>{
        return !_.isEmpty(software.software_package);
      });
      let batchesOfSoftwareWithInstallers = _.chunk(softwareWithSoftwarePackages, 5);
      for(let batch of batchesOfSoftwareWithInstallers) {
        await sails.helpers.flow.simultaneouslyForEach(batch, async(softwareWithInstaller)=>{
          let softwareWithInstallerResponse = await sails.helpers.http.get.with({
            url: `/api/latest/fleet/software/titles/${softwareWithInstaller.id}?team_id=${teamApid}&available_for_install=true`,
            baseUrl: sails.config.custom.fleetBaseUrl,
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
            }
          })
          .timeout(120000)
          .retry(['requestFailed', {name: 'TimeoutError'}]);
          let packageInformation = softwareWithInstallerResponse.software_title.software_package;
          let packageInfo = {
            fleetApid: softwareWithInstaller.id,
            name: packageInformation.name,
            createdAt: new Date(packageInformation.uploaded_at).getTime(),
            platform: _.endsWith(packageInformation.name, 'deb') ? 'Linux' : _.endsWith(packageInformation.name, 'pkg') ? 'macOS' : 'Windows',
            preInstallQuery: packageInformation.pre_install_query,
            installScript: packageInformation.install_script,
            postInstallScript: packageInformation.post_install_script,
            uninstallScript: packageInformation.uninstall_script,
            teams: [],
            isDeployedToAllTeams: false,
          };
          let teamInfo = {
            softwareFleetApid: softwareWithInstaller.id,
            fleetApid: teamApid,
            teamName: _.find(teams, {fleetApid: teamApid}).teamName,
          };
          teamsinformationForSoftware.push(teamInfo);
          if(teamApid === 3) {
            console.log(`{fleetApid: ${softwareWithInstaller.id}, teamApids: ['3']},`)
          }
          allSoftware.push(packageInfo);
          allSoftwareWithPackages.push(packageInfo);
        });// After each software with installer
      }// After each batch of five software items.
      })
    }// After each batch of five teams.
    // After each team on the Fleet instance.
    for(let software of allSoftwareWithPackages) {
      software.teams = _.where(teamsinformationForSoftware, {'softwareFleetApid': software.fleetApid});
      software.teams = _.uniq(software.teams, 'fleetApid')
      if(allTeamsSoftwareApids.includes(software.fleetApid)){
        software.isDeployedToAllTeams = true;
      }
      allSoftware.push(software);
    }
    allSoftware = _.uniq(allSoftware, 'fleetApid');
    let undeployedSoftware = await UndeployedSoftware.find();
    allSoftware = allSoftware.concat(undeployedSoftware);
    return allSoftware;

  }


};
