module.exports = {


  friendlyName: 'Add script',


  description: '',

  files: ['newScript'],

  inputs: {

    newScript: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      required: true,
    },

    teams: {
      type: ['ref'],
      required: true,
      description: 'An array of team IDs that this profile will be added to'
    }
  },


  exits: {
    scriptWithThisNameAlreadyExists: {
      description: 'A script with this name already exists on the Fleet Instance',
      statusCode: 409,
    },
  },


  fn: async function ({newScript, teams}) {

    let path = require('path');
    let fileStream = newScript._files[0].stream;
    let scriptFilename = fileStream.filename;
    console.log(scriptFilename)
    // Strip out any automatically added date prefixes from uploaded scritps.
    let datelessExtensionlessFilename = scriptFilename.replace(/^\d{4}-\d{2}-\d{2}\s/, '').replace(/\.[^/.]+$/, '');
    console.log(datelessExtensionlessFilename)
    let extension = '.'+scriptFilename.split('.').pop();
    // Write the filestream to a temporary file to get the contents as text, and delete the temporary file.
    let tempFilePath = `.tmp/${scriptFilename}`;
    await sails.helpers.fs.writeStream.with({
      sourceStream: newScript._files[0].stream,
      destination: tempFilePath,
      force: true,
    });
    let profileContents = await sails.helpers.fs.read(tempFilePath);
    await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));

    let scriptToReturn;
    // Build a dictonary of information about this script to return to the scripts page.
    let newScriptInfo = {
      name: datelessExtensionlessFilename,
      platform: _.endsWith(scriptFilename, '.ps1') ? 'Windows' : 'macOS and Linux',
      profileType: extension,
      createdAt: Date.now()
    };
    let newTeams = [];
    // Send a request to add the script for every team ID in the array of teams.
    for(let teamApid of teams){
      // Build a request body for the team.
      let requestBodyForThisTeam = {
        script: {
          options: {
            filename: datelessExtensionlessFilename + extension,
            contentType: 'application/octet-stream'
          },
          value: profileContents,
        }
      }
      let addScriptUrl;
      // If the script is being added to the "no team" team, then we need to include the team ID of the no team team in the request URL
      if(Number(teamApid) === 0){
        addScriptUrl = `/api/v1/fleet/scripts?team_id=${teamApid}`;
      } else {
        // Otherwise, the team_id needs to be included in the request's formData.
        addScriptUrl = `/api/v1/fleet/scripts`;
        requestBodyForThisTeam.team_id = Number(teamApid)
      }
      // Send a PSOT request to add the script.
      let newScriptResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        baseUrl: sails.config.custom.fleetBaseUrl,
        url:addScriptUrl,
        enctype: 'multipart/form-data',
        body: requestBodyForThisTeam,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        },
      })
      .intercept({raw: {statusCode: 409}}, (err)=>{
        return 'scriptWithThisNameAlreadyExists';
      });
      let parsedJsonResponse = JSON.parse(newScriptResponse.body)
      let uuidForThisProfile = parsedJsonResponse.script_id;
      newTeams.push({
        fleetApid: teamApid,
        scriptFleetApid: JSON.parse(newScriptResponse.body).script_id,
      })
    }
    newScriptInfo.teams = newTeams;




    // All done.
    return newScriptInfo;

  }


};
