module.exports = {


  friendlyName: 'Edit profile',


  description: '',

  files: ['newProfile'],

  inputs: {
    profile: {
      type: {},
      description: 'The configuration profile that is being editted',
      required: true,
    },
    newTeamIds: {
      type: ['ref'],
      description: 'An array of teams that this profile will be deployed on or Undefined if the profile is being removed from a team.'
    },
    newProfile: {
      type: 'ref',
      description: 'A file that will be replacing the profile.'
    },
  },


  exits: {
    // payloadIdentifierDoesNotMatch: {}
  },
  // For Monday eric:
  // What this does currently:
  // Downlaods a profile if a new one is not provided.

  // Flows to add:
  // Replacing a configuration file.
  //  - with new teams
  //  - without new teams
  // Replacing a deployed configuration file.
  // reaplacing an deployed profile and removing the teams
  // payloadIdentifierDoesNotMatch;



  // What this needs to do/does:
  // Add deployed profile to one/multiple teams « Download profile from fleet instance, add profile to multiple teams. profile.teams && newTeamIds contains ids not in profile.teams.
  // Add undeployed profile to one/multiple teams « Upload profile stored in DB to the ids of teams in the newTeamIds array. (profile.id && newTeamIds)
  // undeploy profile « download profile from fleet instance, save it to the app's database, remove profile from all teams. (profile.teams && newTeamIds.length === 0)
  // remove from one team « Remove profile from team on Fleet instance. (profile.teams && _.difference(_.pluck(profile.teams), 'id'), newTeamIds).length > 0)

  // edge cases to handle:
  // Deploying a new version of a profile to a new team while removing it from the rest. Does the bundle indentifier need to match if it is being removed?
  // Why does it need to match for this dashboard?
    // Because the Fleet server needs it to match?




  fn: async function ({profile, newTeamIds, newProfile}) {
    console.log('Inputs:')
    console.log('profile:',profile);
    console.log('newTeamIds:', newTeamIds)
    // console.log('newProfile:', newProfile)
    console.log('-------------------------')
    if(newProfile.isNoop){
      newProfile.noMoreFiles();
      newProfile = undefined;
    }
    //  ╔═╗╔═╗╔╦╗  ╔═╗╦═╗╔═╗╔═╗╦╦  ╔═╗
    //  ║ ╦║╣  ║   ╠═╝╠╦╝║ ║╠╣ ║║  ║╣
    //  ╚═╝╚═╝ ╩   ╩  ╩╚═╚═╝╚  ╩╩═╝╚═╝
    let profileContents; // The raw text contents of a profile file.
    let filename;
    let extension;
    // If there is not a new profile, and the profile is deployed (has teams array === deployed), download the profile to be able to add it to other teams.
    if(!newProfile && profile.teams){
      console.log('Existing deployed profile!');
      let profileUuid = profile.teams[0].uuid;
      let profileDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/configuration_profiles/${profileUuid}?alt=media`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      });
      let contentDispositionHeader = profileDownloadResponse.headers['content-disposition'];
      let filenameMatch = contentDispositionHeader.match(/filename="(.+?)"/);
      filename = filenameMatch[1];
      extension = '.'+filename.split('.').pop();
      profileContents = profileDownloadResponse.body;
    } else if(newProfile) {
      console.log('Replacing an existing(/undeployed) profile!')
      let path = require('path');
      let fileStream = newProfile._files[0].stream;
      let profileFileName = fileStream.filename;
      filename = profileFileName.replace(/^\d{4}-\d{2}-\d{2}_/, '').replace(/\.[^/.]+$/, '');
      extension = '.'+profileFileName.split('.').pop();
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
      profileContents = await sails.helpers.fs.read(tempFilePath);
      await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));
    } else if (!newProfile && !profile.teams){// Undeployed profiles are stored in the app's database.
      console.log('editing an undeployed profile!')
      profileContents = profile.profileContents;
      filename = profile.name + profile.profileType;
      extension = profile.profileType;
    }
    console.log('filename:', filename);
    console.log('extension:', extension);


    let updatedProfile = {
      name: profile.name,
      createdAt: Date.now(),
      platform: extension === '.xml' ? 'windows' : 'darwin',
      teams: []// Will be added further down the action.
    };
    console.log(profileContents);
    //  ╔═╗╔═╗╔═╗╦╔═╗╔╗╔  ╔═╗╦═╗╔═╗╔═╗╦╦  ╔═╗
    //  ╠═╣╚═╗╚═╗║║ ╦║║║  ╠═╝╠╦╝║ ║╠╣ ║║  ║╣
    //  ╩ ╩╚═╝╚═╝╩╚═╝╝╚╝  ╩  ╩╚═╚═╝╚  ╩╩═╝╚═╝
    let currentProfileTeamIds = _.pluck(profile.teams, 'fleetApid');
    let addedTeams = _.difference(newTeamIds, currentProfileTeamIds);
    let removedTeams = _.difference(currentProfileTeamIds, newTeamIds);
    let removedTeamsInfo = _.filter(profile.teams, (team)=>{
      return removedTeams.includes(team.fleetApid);
    });
    console.log('currentProfileTeamIds', currentProfileTeamIds)
    console.log('addedTeams:', addedTeams);
    console.log('removedTeams:',removedTeams);
    console.log('removedTeamsInfo: ', removedTeamsInfo);

    for(let team of removedTeamsInfo){
      console.log(`removing ${profile.name} from team id ${team.teamName}`);
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        baseUrl: sails.config.custom.fleetBaseUrl,
        url: `/api/v1/fleet/configuration_profiles/${team.uuid}`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        }
      })
    }

    let deployedTeams = [];
    for(let teamApid of addedTeams){
      console.log(`Adding ${profile.name} to team id ${teamApid}`);
      let newProfileResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        baseUrl: sails.config.custom.fleetBaseUrl,
        url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
        enctype: 'multipart/form-data',
        body: {
          team_id: teamApid,
          profile: {
            options: {
              filename: filename + extension,
              contentType: 'application/octet-stream'
            },
            value: profileContents,
          }
        },
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        },
      });
      let parsedJsonResponse = JSON.parse(newProfileResponse.body)
      let uuidForThisProfile = parsedJsonResponse.profile_uuid;
      deployedTeams.push({
        fleetApid: teamApid,
        uuid: JSON.parse(newProfileResponse.body).profile_uuid
      })
    }// After every added team
    for(let team of _.difference(profile.teams, removedTeamsInfo)) {
      deployedTeams.push(team);
    }
    console.log('newTeams:', deployedTeams)
    if(profile.id && deployedTeams.length > 0){
      console.log('Undeployed profile has been deployed. deleting DB record!')
      await UndeployedProfile.destroy({id: profile.id});
    }
    updatedProfile.teams = deployedTeams;

    // }
    console.log(!profile.id);
    console.log(updatedProfile.teams.length > 0);
    if(!profile.id && updatedProfile.teams.length === 0) {
      console.log('Creating database record for a (now) undeployed profile!')
      await UndeployedProfile.create({
        name: profile.name,
        platform: extension === '.xml' ? 'windows' : 'darwin',
        profileContents,
        profileType: extension,
      });
    } else if (profile.id){
      console.log('Updating existing undeployed profile!')
      await UndeployedProfile.updateOne({id: profile.id}).set({
        profileContents,
      });
    }


    console.log('all done! updatedProfile:', updatedProfile);
    // All done.
    return updatedProfile;

  }


};
