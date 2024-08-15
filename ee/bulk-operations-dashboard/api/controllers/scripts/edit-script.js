module.exports = {


  friendlyName: 'Edit script',


  description: '',

  files: ['newScript'],

  inputs: {
    script: {
      type: {},
      description: 'The script that is being editted',
      required: true,
    },
    newTeamIds: {
      type: ['ref'],
      description: 'An array of teams that this script will be added to.'
    },
    newScript: {
      type: 'ref',
      description: 'A file that will be replacing the script.'
    },
  },


  exits: {
    // payloadIdentifierDoesNotMatch: {}
  },
  // What does this need to do?
  // Provided with a list of newTeamIds:
  //  - It will download the script from the fleet server and send requests to add it to the team IDs provided
  //   , and remove it from any team IDs that are not present in both script.teams and newTeamIds array.
  // Provided with a newScript:
  //  - It will remove the old script, then add the new script to the team IDs in the newTeamIds array.



  // What this needs to do/does:
  // Add deployed script to one/multiple teams « Download script from fleet instance, add script to multiple teams. script.teams && newTeamIds contains ids not in script.teams.
  // Add undeployed script to one/multiple teams « Upload script stored in DB to the ids of teams in the newTeamIds array. (script.id && newTeamIds)
  // undeploy script « download script from fleet instance, save it to the app's database, remove script from all teams. (script.teams && newTeamIds.length === 0)
  // remove from one team « Remove script from team on Fleet instance. (script.teams && _.difference(_.pluck(script.teams), 'id'), newTeamIds).length > 0)

  // edge cases to handle:
  // Deploying a new version of a script to a new team while removing it from the rest. Does the bundle indentifier need to match if it is being removed?
  // Why does it need to match for this dashboard?
    // Because the Fleet server needs it to match?




  fn: async function ({script, newTeamIds, newScript}) {
    // console.log(script);
    // console.log('newTeamIds', newTeamIds)
    // console.log('newScript', newScript)
    // console.log('-------------------------')
    if(newScript.isNoop){
      newScript.noMoreFiles();
      newScript = undefined;
    }
    let scriptContents; // The raw text contents of a script file.
    let filename;
    let extension;
    // If there is not a new script, and the script is deployed (has teams array === deployed), download the script to be able to add it to other teams.
    if(!newScript){
      console.log('Existing script!');
      let scriptFleetApid = script.teams[0].scriptFleetApid;
      let scriptDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/scripts/${scriptFleetApid}?alt=media`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      });
      console.log(scriptDownloadResponse);
      let contentDispositionHeader = scriptDownloadResponse.headers['content-disposition'];
      let filenameMatch = contentDispositionHeader.match(/filename="(.+?)"/);
      filename = filenameMatch[1];
      console.log(filenameMatch, filename);
      extension = '.'+filename.split('.').pop();
      filename = filename.replace(/^\d{4}-\d{2}-\d{2}[_|\s]?/, '');
      scriptContents = scriptDownloadResponse.body;
    } else if(newScript) {
      console.log('New script!')
      let path = require('path');
      let fileStream = newScript._files[0].stream;
      let scriptFilename = fileStream.filename;
      filename = scriptFilename.replace(/^\d{4}-\d{2}-\d{2}[_|\s]?/, '').replace(/\.[^/.]+$/, '');
      console.log(filename);
      extension = '.'+scriptFilename.split('.').pop();
      let tempFilePath = `.tmp/${scriptFilename}`;
      let scriptPlatform = 'macOS & Linux';
      if(_.endsWith(scriptFilename, '.ps1')) {
        scriptPlatform = 'Windows';
      }
      await sails.helpers.fs.writeStream.with({
        sourceStream: newScript._files[0].stream,
        destination: tempFilePath,
        force: true,
      });
      scriptContents = await sails.helpers.fs.read(tempFilePath);
      await sails.helpers.fs.rmrf(path.join(sails.config.appPath, tempFilePath));
    }
    // If this is a deployed script, get a list of teams that have been added, and teams that have been removed.
    // if(script.teams) {
    // Note: there might be an edgecase where we don't have this information. If a script is added, the uuid is added to the scripts object until a page refresh.
    // ∆: update add script endpint to send a request to get the added script's uuid to prevent this.

    let currentScriptTeamIds = _.pluck(script.teams, 'fleetApid');
    let addedTeams = _.difference(newTeamIds, currentScriptTeamIds);
    let removedTeams = _.difference(currentScriptTeamIds, newTeamIds);
    console.log('added!', addedTeams);
    console.log('removed!',removedTeams);

    let removedTeamsInfo = _.filter(script.teams, (team)=>{
      return removedTeams.includes(team.fleetApid);
    });
    console.log(removedTeamsInfo);
    for(let script of removedTeamsInfo){
      console.log(`removing ${script.name} from team id ${script.teamName}`);
      await sails.helpers.http.sendHttpRequest.with({
        method: 'DELETE',
        baseUrl: sails.config.custom.fleetBaseUrl,
        url: `/api/v1/fleet/scripts/${script.scriptFleetApid}`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        }
      })
    }

    let newTeams = [];
    for(let teamApid of addedTeams){
      console.log(`adding ${script.name} to team id ${teamApid}`);
      // Build a request body for the team.
      let requestBodyForThisTeam = {
        script: {
          options: {
            filename: filename,
            contentType: 'application/octet-stream'
          },
          value: scriptContents,
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
      let newScriptResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'POST',
        baseUrl: sails.config.custom.fleetBaseUrl,
        url: addScriptUrl,
        enctype: 'multipart/form-data',
        body: requestBodyForThisTeam,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        },
      });
      let parsedJsonResponse = JSON.parse(newScriptResponse.body)
      let fleetApidForThisScript = parsedJsonResponse.script_id;
      newTeams.push({
        fleetApid: teamApid,
        uuid: fleetApidForThisScript,
      })
    }

    // All done.
    return;

  }


};
