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
    let teamApids = _.pluck(teams, 'fleetApid');
    // Get all of the software packages on the Fleet instance.
    for(let teamApid of teamApids){
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
      for(let softwareWithInstaller of softwareWithSoftwarePackages) {
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
        };
        let teamInfo = {
          softwareFleetApid: softwareWithInstaller.id,
          fleetApid: teamApid,
          teamName: _.find(teams, {fleetApid: teamApid}).teamName,
        };
        teamsinformationForSoftware.push(teamInfo);

        allSoftware.push(packageInfo);
        allSoftwareWithPackages.push(packageInfo);
      }
    }
    for(let software of allSoftwareWithPackages) {
      software.teams = _.where(teamsinformationForSoftware, {'softwareFleetApid': software.fleetApid});
      allSoftware.push(software);
    }
    allSoftware = _.uniq(allSoftware, 'fleetApid');
    let undeployedSoftware = await UndeployedSoftware.find();
    allSoftware = allSoftware.concat(undeployedSoftware);
    return allSoftware;

  }


};
