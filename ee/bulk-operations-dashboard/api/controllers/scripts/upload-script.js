module.exports = {


  friendlyName: 'Upload script',


  description: 'Uploads a script to the connected Fleet instance',

  files: ['newScript'],

  inputs: {

    newScript: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      required: true,
    },

    teams: {
      type: ['string'],
      description: 'An array of team IDs that this profile will be added to'
    }
  },


  exits: {
    success: {
      outputDescription: 'The new script has been uploaded',
      outputType: {},
    },

    scriptWithThisNameAlreadyExists: {
      description: 'A script with this name already exists on the Fleet Instance',
      statusCode: 409,
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


  fn: async function ({newScript, teams}) {

    let util = require('util');
    let script = await sails.reservoir(newScript)
    .intercept('E_EXCEEDS_UPLOAD_LIMIT', 'tooBig')
    .intercept((err)=>new Error('The script upload failed. '+util.inspect(err)));
    if(!script) {
      throw 'noFileAttached';
    }
    // Get the file contents and filename.
    let scriptContents = script[0].contentBytes;
    let scriptFilename = script[0].name;
    // Strip out any automatically added date prefixes from uploaded scripts.
    let datelessExtensionlessFilename = scriptFilename.replace(/^\d{4}-\d{2}-\d{2}\s/, '').replace(/\.[^/.]+$/, '');
    let extension = '.'+scriptFilename.split('.').pop();
    // Build a dictonary of information about this script to return to the scripts page.
    let newScriptInfo = {
      name: datelessExtensionlessFilename,
      platform: _.endsWith(scriptFilename, '.ps1') ? 'Windows' : 'macOS & Linux',
      scriptType: extension,
      createdAt: Date.now()
    };
    if(!teams) {
      newScriptInfo.scriptContents = scriptContents;
      await UndeployedScript.create(newScriptInfo).fetch();
    } else {
      // Send a request to add the script for every team ID in the array of teams.
      for(let teamApid of teams){
        // Build a request body for the team.
        let requestBodyForThisTeam = {
          script: {
            options: {
              filename: datelessExtensionlessFilename + extension,
              contentType: 'application/octet-stream'
            },
            value: scriptContents,
          }
        };
        let addScriptUrl;
        // If the script is being added to the "no team" team, then we need to include the team ID of the no team team in the request URL
        if(Number(teamApid) === 0){
          addScriptUrl = `/api/v1/fleet/scripts?team_id=${teamApid}`;
        } else {
          // Otherwise, the team_id needs to be included in the request's formData.
          addScriptUrl = `/api/v1/fleet/scripts`;
          requestBodyForThisTeam.team_id = Number(teamApid);// eslint-disable-line camelcase
        }
        // Send a PSOT request to add the script.
        await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          baseUrl: sails.config.custom.fleetBaseUrl,
          url:addScriptUrl,
          enctype: 'multipart/form-data',
          body: requestBodyForThisTeam,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          },
        })
        .intercept({raw: {statusCode: 409}}, ()=>{
          return 'scriptWithThisNameAlreadyExists';
        });
      }
    }
    // All done.
    return newScriptInfo;

  }


};
