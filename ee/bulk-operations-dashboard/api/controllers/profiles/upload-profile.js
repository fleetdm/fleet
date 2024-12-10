module.exports = {


  friendlyName: 'Upload profile',


  description: '',

  files: ['newProfile'],

  inputs: {
    newProfile: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      required: true,
    },
    teams: {
      type: ['string'],
      description: 'An array of team IDs that this profile will be added to'
    },
    profileTarget: {
      type: 'string',
      description: 'The target for this configuration profile',
      defaultsTo: 'all',
      isIn: ['all', 'custom'],
    },
    labelTargetBehavior: {
      type: 'string',
      isIn: ['include', 'exclude'],
    },
    labels: {
      type: ['string'],
      description: 'A list of the names of labels that will be included/excluded.'
    }
  },


  exits: {
    success: {
      outputDescription: 'The new profile has been uploaded',
      outputType: {},
    },

    noFileAttached: {
      description: 'No file was attached.',
      responseType: 'badRequest'
    },

    tooBig: {
      description: 'The file is too big.',
      responseType: 'badRequest'
    },

  },


  fn: async function ({newProfile, teams, profileTarget, labelTargetBehavior, labels}) {
    let util = require('util');
    let profile = await sails.reservoir(newProfile)
    .intercept('E_EXCEEDS_UPLOAD_LIMIT', 'tooBig')
    .intercept((err)=>new Error('The configuration profile upload failed. '+util.inspect(err)));
    if(!profile) {
      throw 'noFileAttached';
    }
    let profileContents = profile[0].contentBytes;
    let profileFileName = profile[0].name;
    let datelessExtensionlessFilename = profileFileName.replace(/^\d{4}-\d{2}-\d{2}_/, '').replace(/\.[^/.]+$/, '');
    let extension = '.'+profileFileName.split('.').pop();
    let profilePlatform = 'darwin';
    if(_.endsWith(profileFileName, '.xml')) {
      profilePlatform = 'windows';
    }

    let profileToReturn;
    let newProfileInfo = {
      name: datelessExtensionlessFilename,
      platform: profilePlatform,
      profileType: extension,
      createdAt: Date.now(),
      profileTarget,
      labels,
      labelTargetBehavior,
    };
    if(!teams) {
      newProfileInfo.profileContents = profileContents;
      profileToReturn = await UndeployedProfile.create(newProfileInfo).fetch();
    } else {
      let newTeams = [];
      for(let teamApid of teams){
        let bodyForThisRequest = {
          team_id: teamApid,// eslint-disable-line camelcase
          labels_exclude_any: labelTargetBehavior === 'exclude' ? labels : undefined,// eslint-disable-line camelcase
          labels_include_all: labelTargetBehavior === 'include' ? labels : undefined,// eslint-disable-line camelcase
          profile: {
            options: {
              filename: profileFileName,
              contentType: 'application/octet-stream'
            },
            value: profileContents,
          }
        };
        let newProfileResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles?team_id=${teamApid}`,
          enctype: 'multipart/form-data',
          body: bodyForThisRequest,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        });
        let parsedJsonResponse = JSON.parse(newProfileResponse.body);
        let uuidForThisProfile = parsedJsonResponse.profile_uuid;
        // send a request to the Fleet instance to get the bundleId of the new profile.
        await sails.helpers.http.get.with({
          baseUrl: sails.config.custom.fleetBaseUrl,
          url: `/api/v1/fleet/configuration_profiles/${uuidForThisProfile}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        newTeams.push({
          fleetApid: teamApid,
          uuid: JSON.parse(newProfileResponse.body).profile_uuid
        });
      }
      newProfileInfo.teams = newTeams;
      profileToReturn = newProfileInfo;
    }




    // All done.
    return profileToReturn;

  }


};
