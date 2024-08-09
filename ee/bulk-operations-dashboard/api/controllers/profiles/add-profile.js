module.exports = {


  friendlyName: 'Add profile',


  description: '',

  files: ['newProfile'],

  inputs: {
    newProfile: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      // required: true,
    },
    teams: {
      type: ['ref'],
      description: 'An array of team IDs that this profile will be added to'
    }
  },


  exits: {

  },


  fn: async function ({newProfile, teams}) {
    // console.log(newProfile);
    console.log(teams);
    let path = require('path');

    let fileStream = newProfile._files[0].stream;

    // console.log(fileStream);
    let profileFileName = fileStream.filename;
    let tempFilePath = `.tmp/${profileFileName}`;
    let profilePlatform = 'darwin';
    if(_.endsWith(profileFileName, '.xml')) {
      profilePlatform = 'windows';
    }

    await sails.helpers.fs.writeStream.with({
      sourceStream: newProfile._files[0].stream,
      destination: tempFilePath,
      force: true,
    });
    let profileToReturn;


    let profileContents = await sails.helpers.fs.read(tempFilePath);
    // console.log(profileContents);
    // console.log(file);
    // console.log(newProfile);
    let newProfileInfo = {
      name: profileFileName,
      platform: profilePlatform,
    };
    if(!teams) {
      newProfileInfo.profileContents = profileContents;
      profileToReturn = await UndeployedProfile.create(newProfileInfo).fetch();
    } else {
      newProfile.teams = [];
      for(let teamApid in teams){
        let newProfileResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: {
            profile: profileContents,
          },
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        });
        console.log(newProfileResponse);
        newProfileInfo.teams.push({
          teamApid: teamApid,
          uuid: newProfileResponse.profile_uuid
        })
      }
      profileToReturn = newProfileInfo;
    }

    await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));


    // All done.
    return profileToReturn;

  }


};
