module.exports = {


  friendlyName: 'Detect new teams and transfer software',


  description: '',


  fn: async function () {
    sails.log('Running custom shell script... (`sails run detect-new-teams-and-transfer-software`)');
    console.time('detect and transfer software script');

    // Get teams
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
    let allTeamApids = [];
    for(let team of allTeams) {
      allTeamApids.push(team.id);
    }
    // Add the "team" for hosts with no team
    allTeamApids.push(0);
    allTeamApids = allTeamApids.map(String);
    // Get software deployed to all teams
    let softwareDeployedToAllTeams = await AllTeamsSoftware.find();
    let WritableStream = require('stream').Writable;
    let axios = require('axios');
    // Loop through the software deployed to all teams.
    for(let software of softwareDeployedToAllTeams) {
      let teamsThisSoftwareIsNotDeployedOn = _.difference(allTeamApids, software.teamApids);
      console.time(`transfer for ${software.fleetApid} (${teamsThisSoftwareIsNotDeployedOn.length} new teams)`);
      if(teamsThisSoftwareIsNotDeployedOn.length > 0) {
        sails.log.info(`${teamsThisSoftwareIsNotDeployedOn.length} new team(s) detected for software id ${software.fleetApid}!`);
        // Get software installer:
        let teamIdToGetInstallerFrom = software.teamApids[0];
        sails.log.info(`Getting information about an installer (ID: ${software.fleetApid})`);
        let softwareResponse = await sails.helpers.http.get.with({
          url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}?team_id=${teamIdToGetInstallerFrom}&available_for_install=true`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        let installerInformation = softwareResponse.software_title.software_package;
        if(!installerInformation){
          sails.log.warn(`No installer found on Fleet instance for ${softwareResponse.software_title}. Skipping...`);
          continue;
        }
        // Get a download stream of the software installer and upload it to the s3 bucket.
        let downloadApiUrl = `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}/package?alt=media&team_id=${teamIdToGetInstallerFrom}`;
        let softwareStream = await sails.helpers.http.getStream.with({
          url: downloadApiUrl,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        })
        .tolerate((error)=>{
          sails.log.warn(`When sending an API request to get the streaming bytes of a software installer (${software.fleetApid}), an error occurred., Full error: ${require('util').inspect(error, {depth: null})}`);
        });
        sails.log.info(`Uploading ${installerInformation.name} to s3 bucket.`);
        let tempUploadedSoftware = await sails.uploadOne(softwareStream, {adapter: require('skipper-disk'), maxBytes: sails.config.uploads.maxBytes});
        // let tempUploadedSoftware = await sails.uploadOne(softwareStream, {bucket: sails.config.uploads.bucketWithPostfix});
        let softwareFd = tempUploadedSoftware.fd;
        sails.log.info(`${installerInformation.name} upload complete, starting transfer to Fleet instance for added teams.`);
        // Clone the array of current teams this software is assigned to.
        let newTeamIdsForThisSoftware = _.clone(software.teamApids);
        // Batch teams in groups of five, and send each request to add the software to each team simultaneously.
        let batchedTeamsThisSoftwareIsNoteDeployedOn = _.chunk(teamsThisSoftwareIsNotDeployedOn, 5);
        for(let batch of batchedTeamsThisSoftwareIsNoteDeployedOn) {
          await sails.helpers.flow.simultaneouslyForEach(batch, async (team)=>{
            sails.log.info(`Copying ${installerInformation.name} to team_id ${team}`);
            // Send an api request to send the file to the Fleet server for each new team.
            let transferWasSuccessful = true;
            await sails.cp(softwareFd, {bucket: sails.config.uploads.bucketWithPostfix},
            // await sails.cp(softwareFd, {
            //     adapter: require('skipper-disk'),
            //     maxBytes: sails.config.uploads.maxBytes,
            //   },
              {
                adapter: ()=>{
                  return {
                    ls: undefined,
                    rm: undefined,
                    read: undefined,
                    receive: (unusedOpts)=>{
                      // This `_write` method is invoked each time a new file is received
                      // from the Readable stream (Upstream) which is pumping filestreams
                      // into this receiver.  (filename === `__newFile.filename`).
                      var receiver__ = WritableStream({ objectMode: true });
                      // Create a new drain (writable stream) to send through the individual bytes of this file.
                      receiver__._write = (__newFile, encoding, doneWithThisFile)=>{

                        let FormData = require('form-data');
                        let form = new FormData();
                        form.append('team_id', team);
                        form.append('install_script', installerInformation.install_script);
                        form.append('post_install_script', installerInformation.post_install_script);
                        form.append('pre_install_query', installerInformation.pre_install_query);
                        form.append('uninstall_script', installerInformation.uninstall_script);
                        form.append('software', __newFile, {
                          filename: installerInformation.name,
                          contentType: 'application/octet-stream'
                        });
                        (async ()=>{
                          await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
                            headers: {
                              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                              ...form.getHeaders()
                            },
                            maxRedirects: 0,
                          });
                        })()
                        .then(()=>{
                          doneWithThisFile();
                        })
                        .catch((err)=>{
                          doneWithThisFile(err);
                        });
                      };//ƒ
                      return receiver__;
                    }
                  };
                },
              })
              .tolerate({status: 409}, ()=>{
                // If this software item already exists on this team, log a warning and continue
                sails.log.verbose(`${installerInformation.name} already exists on this team (id: ${team}), skipping.....`);
              })
              .tolerate((error)=>{
                // If any other error occurs while transfering this installer, change the transferWasSuccessful flag to false.
                transferWasSuccessful = false;
                sails.log.warn(`When attempting to upload a software installer (${installerInformation.name} to team_id:${team}, an unexpected error occurred communicating with the Fleet API. This script will continue to attempt to transfer software, and will try to transfer this software item to this team on the next run of this script. ${require('util').inspect(error, {depth: null})}`);
              });
            if(!transferWasSuccessful){
              // If this flag was set to false, do not add this team's APID to the list of teams for this software. This will result in this software installer being re-sent durring the next run of this script.
              return;
            }
            newTeamIdsForThisSoftware.push(team);
            // Create a copy of the software's new teams array and update the Database record.
            let newTeamsToUpdateDatabaseRecordWith = _.clone(newTeamIdsForThisSoftware);
            await AllTeamsSoftware.updateOne({id: software.id}).set({teamApids: newTeamsToUpdateDatabaseRecordWith});

          });//∞ for each new team.
        }//∞ each batch of 5 new teams.

        sails.log.info(`software transfer complete for ${installerInformation.name}, updating database record with new teams.`);
        // Update the AllTeamsSoftware record's teamApids value
        // Delete the temporary file stored in s3.

        await sails.rm(sails.config.uploads.prefixForFileDeletion+tempUploadedSoftware.fd);
        // await sails.rm(tempUploadedSoftware.fd, {adapter: require('skipper-disk')});
      }//ﬁ
      console.timeEnd(`transfer for ${software.fleetApid} (${teamsThisSoftwareIsNotDeployedOn.length} new teams)`);
    }//∞ for each AllTeamsSoftware record.
    console.timeEnd('detect and transfer software script');
  }


};

