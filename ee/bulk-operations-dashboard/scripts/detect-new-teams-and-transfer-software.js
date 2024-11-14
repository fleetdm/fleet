module.exports = {


  friendlyName: 'Detect new teams and transfer software',


  description: '',


  fn: async function () {
    sails.log('Running custom shell script... (`sails run detect-new-teams-and-transfer-software`)');

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
    for(let software of softwareDeployedToAllTeams) {
      console.log(software)
      // console.log(allTeamApids)
      let teamsThisSoftwareIsNotDeployedOn = _.difference(allTeamApids, software.teamApids);
      if(teamsThisSoftwareIsNotDeployedOn.length > 0) {
        sails.log(`${teamsThisSoftwareIsNotDeployedOn.length} New team(s) detected!`)
        // Get software installer:
        let teamIdToGetInstallerFrom = software.teamApids[0];
        sails.log(`Getting information about this installer`);
        let softwareResponse = await sails.helpers.http.get.with({
          url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}`,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        let installerInformation = softwareResponse.software_title.software_package;
        sails.log('got installer info!', installerInformation);
        sails.log(`Downloading installer to upload into an s3 bucket.`);
        let downloadApiUrl = `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}/package?alt=media&team_id=${teamIdToGetInstallerFrom}`;
        console.log(downloadApiUrl);
        let softwareStream = await sails.helpers.http.getStream.with({
          url: downloadApiUrl,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        sails.log(`Uploading installer to s3 bucket.`);
        let tempUploadedSoftware = await sails.uploadOne(softwareStream, {bucket: sails.config.uploads.bucketWithPostfix});
        softwareFd = tempUploadedSoftware.fd;
        softwareMime = tempUploadedSoftware.type;
        sails.log(`Upload complete, starting transfer to Fleet instance for added teams.`);

        await sails.helpers.flow.simultaneouslyForEach(teamsThisSoftwareIsNotDeployedOn, async (team)=>{
        sails.log(`Copying ${installerInformation.name} to team_id ${team}`)
          // console.log(`transfering ${software.name} to fleet instance for team id ${team}`);
          // Send an api request to send the file to the Fleet server for each added team.
          await sails.cp(softwareFd, {bucket: sails.config.uploads.bucketWithPostfix},
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
                        });
                      })()
                      .then(()=>{
                        // console.log('ok supposedly a file is finished uploading');
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
          .intercept(async (error)=>{
            // Note: with this current behavior, all errors from this upload are currently swallowed and a softwareUploadFailed response is returned.
            // FUTURE: Test to make sure that uploading duplicate software to a team results in a 409 response.
            // Before handline errors, decide what to do about the file uploaded to s3, if this is undeployed software, we'll leave it alone, but if this was a temporary file created to transfer it between teams on the Fleet instance, we'll delete the file.
            if(!software.id) {// If the software does not have an ID, it not stored in the app's database/s3 bucket, so we can safely delete the file in s3.
              await sails.rm(sails.config.uploads.prefixForFileDeletion+softwareFd);
            }
            return new Error(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API, ${require('util').inspect(error, {depth: null})}`);
          });
          // console.timeEnd(`transfering ${software.name} to fleet instance for team id ${team}`);
        });
        sails.log(`software transfer complete for ${installerInformation.name}, updating database record with new teams.`)
        await AllTeamsSoftware.updateOne({id: software.id}).set({teamApids: allTeamApids});
      }
    }





  }


};

