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
    },
    unauthorized: {
      description: 'The provided token was invalid.',
      responseType: 'unauthorized'
    }
  },


  fn: async function ({ platform }) {

    if (!this.req.get('authorization')) {
      return this.res.unauthorized();
    }
    let authorizationHeader = this.req.get('authorization');
    if(!_.startsWith(authorizationHeader, 'Bearer ')) {
      return this.res.unauthorized();
    }
    let tokenInAuthorizationHeader = authorizationHeader.split('Bearer ')[1];
    if(!tokenInAuthorizationHeader) {
      return this.res.unauthorized();
    }



    // Get teams information.
    let teamsResponseData = await sails.helpers.http.get.with({
      url: '/api/v1/fleet/teams',
      baseUrl: sails.config.custom.fleetBaseUrl,
      headers: {
        Authorization: `Bearer ${tokenInAuthorizationHeader}`
      }
    })
    .timeout(120000)
    .retry(['requestFailed', {name: 'TimeoutError'}])
    .intercept('non200Response', 'unauthorized');

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
    await sails.helpers.flow.simultaneouslyForEach(teamApids, async (teamApid)=>{
      let configurationProfilesResponseData = await sails.helpers.http.get.with({
        url: `/api/latest/fleet/software/titles?team_id=${teamApid}`,
        baseUrl: sails.config.custom.fleetBaseUrl,
        headers: {
          Authorization: `Bearer ${tokenInAuthorizationHeader}`
        }
      })
      .timeout(120000)
      .retry(['requestFailed', {name: 'TimeoutError'}])
      .intercept('non200Response', 'unauthorized');
      let softwareForThisTeam = configurationProfilesResponseData.software_titles;
      let softwareWithSoftwarePackages = _.filter(softwareForThisTeam, (software)=>{
        return !_.isEmpty(software.software_package);
      });
      // Exclude Fleet maintained apps from the list of software. (If a software item has a package_url value, it is a Fleet maintained app)
      let softwarePackagesWithNoDownloadUrl = _.filter(softwareWithSoftwarePackages, (software)=>{
        let softwarePackage = software.software_package;
        return softwarePackage.package_url !== undefined;
      });
      await sails.helpers.flow.simultaneouslyForEach(softwarePackagesWithNoDownloadUrl, async (softwareWithInstaller)=>{
        let softwareWithInstallerResponse = await sails.helpers.http.get.with({
          url: `/api/latest/fleet/software/titles/${softwareWithInstaller.id}?team_id=${teamApid}&available_for_install=true`,
          baseUrl: sails.config.custom.fleetBaseUrl,
          headers: {
            Authorization: `Bearer ${tokenInAuthorizationHeader}`
          }
        })
        .timeout(120000)
        .retry(['requestFailed', {name: 'TimeoutError'}])
        .intercept('non200Response', 'unauthorized');
        let packageInformation = softwareWithInstallerResponse.software_title.software_package;
        let packageInfo = {
          software_title_name: softwareWithInstaller.name,// eslint-disable-line camelcase
          software_title_id: softwareWithInstaller.id,// eslint-disable-line camelcase
          installer_name: packageInformation.name,// eslint-disable-line camelcase
          installer_version: packageInformation.version,// eslint-disable-line camelcase
          platform: _.endsWith(packageInformation.name, 'deb') ? 'linux' : _.endsWith(packageInformation.name, 'pkg') ? 'darwin' : 'windows',
          teams: [],
        };
        let teamInfo = {
          softwareFleetApid: softwareWithInstaller.id,
          id: teamApid,
          team_name: _.find(teams, {fleetApid: teamApid}).teamName,// eslint-disable-line camelcase
        };
        teamsinformationForSoftware.push(teamInfo);
        allSoftwareWithPackages.push(packageInfo);
      });// After every software item with an installer

    });// After every team

    for(let software of allSoftwareWithPackages) {
      software.teams = _.where(teamsinformationForSoftware, {'softwareFleetApid': software.software_title_id});
      software.teams = _.map(software.teams, (team)=>{
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
