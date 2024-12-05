module.exports = {


  friendlyName: 'List software',


  description: 'Returns a JSON list of all software with installer packages on a Fleet instance.',


  inputs: {
    platform: {
      type: 'string',
      isIn: [
        'darwin',
        'windows',
        'linux',
      ],
      description: 'If provided, the API resopnse will only include software for the specified platform.',
    }
  },


  exits: {
    success: {
      description: 'A list of software has been returned to the requesting user.',
    }
  },


  fn: async function ({ platform }) {
    // Get teams information.
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
    // Now collect information about software.

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
          software_title_name: softwareWithInstaller.name,
          software_title_id: softwareWithInstaller.id,
          installer_name: packageInformation.name,
          installer_version: packageInformation.version,
          platform: _.endsWith(packageInformation.name, 'deb') ? 'linux' : _.endsWith(packageInformation.name, 'pkg') ? 'darwin' : 'windows',
          teams: [],
        };
        let teamInfo = {
          softwareFleetApid: softwareWithInstaller.id,
          id: teamApid,
          team_name: _.find(teams, {fleetApid: teamApid}).teamName,
        };
        teamsinformationForSoftware.push(teamInfo);
        allSoftwareWithPackages.push(packageInfo);
      }
    }
    for(let software of allSoftwareWithPackages) {
      software.teams = _.where(teamsinformationForSoftware, {'softwareFleetApid': software.software_title_id});
      software.teams = _.map(software.teams, function(team) {
        return _.omit(team, 'softwareFleetApid');
      });
      allSoftware.push(software);
    }
    allSoftware = _.uniq(allSoftware, 'software_title_id');

    // IF platform is provided, filter the results to only return software for the specified platform.
    if(platform) {
      allSoftware = _.filter(allSoftware, (software)=>{
        return software.platform === platform;
      });
    }

    return this.res.json(allSoftware);
  }


};
