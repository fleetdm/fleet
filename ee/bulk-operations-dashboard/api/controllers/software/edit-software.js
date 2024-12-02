module.exports = {


  friendlyName: 'Edit software',


  description: 'Edits deployed software on a Fleet instance or undeployed software in the app\'s database',

  files: ['newSoftware'],

  inputs: {
    newSoftware: {
      type: 'ref',
      description: 'The optional streaming bytes of a new software version.'
    },
    newTeamIds: {
      type: ['string'],
      description: 'The new teams that this software will be deployed to.'
    },
    software: {
      type: {},
      description: 'The software that will be editted.'
    },

    preInstallQuery: {
      type: 'string',
    },

    installScript: {
      type: 'string',
    },

    postInstallScript: {
      type: 'string',
    },

    uninstallScript: {
      type: 'string',
    },

  },


  exits: {
    wrongInstallerExtension: {
      description: 'The provided replacement software\'s has the wrong extension.',
      statusCode: 400,
    },
    softwareUploadFailed: {
      description: 'The software upload failed'
    }
  },


  fn: async function ({newSoftware, newTeamIds, software, preInstallQuery, installScript, postInstallScript, uninstallScript}) {
    if(newSoftware.isNoop) {
      newSoftware.noMoreFiles();
      newSoftware = undefined;
    }
    var WritableStream = require('stream').Writable;
    // let { Readable } = require('stream');
    let axios = require('axios');
    // Cast the strings in the newTeamIds array to numbers.
    if(newTeamIds){
      newTeamIds = newTeamIds.map(Number);
    } else {
      newTeamIds = [];
    }

    let currentSoftwareTeamIds = _.pluck(software.teams, 'fleetApid');
    // If the teams have changed, or a new installer package was provided, we'll need to upload the package to an s3 bucket to deploy it to other teams.
    if(_.xor(newTeamIds, currentSoftwareTeamIds).length !== 0 || newSoftware) {
      let softwareFd;
      let softwareName;
      let softwareMime;
      if(software.teams && !newSoftware) {
        // console.log('Editing deployed software!');
        // This software is deployed.
        // get software from Fleet instance and upload to s3.
        let teamIdToGetInstallerFrom = software.teams[0].fleetApid;
        let downloadApiUrl = `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}/package?alt=media&team_id=${teamIdToGetInstallerFrom}`;
        let softwareStream = await sails.helpers.http.getStream.with({
          url: downloadApiUrl,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        })
        .intercept('non200Response', (error)=>{
          return new Error(`When attempting to transfer the installer for ${software.name} to a new team on the Fleet instance, the Fleet isntance returned a non-200 response when a request was sent to get a download stream of the installer on team_id ${teamIdToGetInstallerFrom}. Full Error: ${require('util').inspect(error, {depth: 1})}`);
        });
        let tempUploadedSoftware = await sails.uploadOne(softwareStream, {bucket: sails.config.uploads.bucketWithPostfix});
        softwareFd = tempUploadedSoftware.fd;
        softwareName = software.name;
        softwareMime = tempUploadedSoftware.type;
      } else if(newSoftware) {
        // If a new copy of the installer was uploaded, we'll
        // console.log('replacing software package!');
        let uploadedSoftware = await sails.uploadOne(newSoftware, {bucket: sails.config.uploads.bucketWithPostfix});
        softwareFd = uploadedSoftware.fd;
        softwareName = uploadedSoftware.filename;
        softwareMime = uploadedSoftware.type;
        let newSoftwareExtension = '.'+softwareName.split('.').pop();
        let existingSoftwareExtension = '.'+software.name.split('.').pop();
        if(newSoftwareExtension !== existingSoftwareExtension) {
          await sails.rm(sails.config.uploads.prefixForFileDeletion+softwareFd);
          throw {wrongInstallerExtension: `Couldn't edit ${software.name}. The selected package should be a ${existingSoftwareExtension} file`};
        }
      } else {
        // console.log('Editing undeployed software!');
        softwareFd = software.uploadFd;
        softwareName = software.name;
        softwareMime = software.uploadMime;
      }
      // Now apply the edits.
      if(newTeamIds.length !== 0) {
        let currentSoftwareTeamIds = _.pluck(software.teams, 'fleetApid') || [];
        let addedTeams = _.difference(newTeamIds, currentSoftwareTeamIds);
        let removedTeams = _.difference(currentSoftwareTeamIds, newTeamIds);
        let unchangedTeamIds = _.difference(currentSoftwareTeamIds, removedTeams);
        // for(let team of addedTeams) {
        await sails.helpers.flow.simultaneouslyForEach(addedTeams, async (team)=>{
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
                      form.append('install_script', installScript);
                      form.append('post_install_script', postInstallScript);
                      form.append('pre_install_query', preInstallQuery);
                      form.append('uninstall_script', uninstallScript);
                      form.append('software', __newFile, {
                        filename: software.name,
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
            // Log a warning containing an error
            sails.log.warn(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API, Full error: ${require('util').inspect(error, {depth: 2})}`);
            return {'softwareUploadFailed': error};
          });
          // console.timeEnd(`transfering ${software.name} to fleet instance for team id ${team}`);
        });
        // }// After every new team this is deployed to.
        if(newSoftware) {
          // If a new installer package was provided, send patch requests to update the installer package on teams that it is already deployed to.
          await sails.helpers.flow.simultaneouslyForEach(unchangedTeamIds, async (teamApid)=>{
            // console.log(`Adding new version of ${softwareName} to teamId ${teamApid}`);
            await sails.helpers.http.sendHttpRequest.with({
              method: 'DELETE',
              baseUrl: sails.config.custom.fleetBaseUrl,
              url: `/api/v1/fleet/software/titles/${software.fleetApid}/available_for_install?team_id=${teamApid}`,
              headers: {
                Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
              }
            });
            // console.log(`transfering the changed installer ${software.name} to fleet instance for team id ${teamApid}`);
            // console.time(`transfering ${software.name} to fleet instance for team id ${teamApid}`);
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
                      form.append('team_id', teamApid);
                      form.append('install_script', installScript);
                      form.append('post_install_script', postInstallScript);
                      form.append('pre_install_query', preInstallQuery);
                      form.append('uninstall_script', uninstallScript);
                      form.append('software', __newFile, {
                        filename: software.name,
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
              // Before handling errors, decide what to do about the file uploaded to s3, if this is undeployed software, we'll leave it alone, but if this was a temporary file created to transfer it between teams on the Fleet instance, we'll delete the file.
              if(!software.id) {
                await sails.rm(sails.config.uploads.prefixForFileDeletion+softwareFd);
              }
              // Log a warning containing an error
              sails.log.warn(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API, ${require('util').inspect(error, {depth: 2})}`);
              return {'softwareUploadFailed': error};
            });
            // console.timeEnd(`transfering ${software.name} to fleet instance for team id ${teamApid}`);
          });// After every team the software is currently deployed to.
        } else if(preInstallQuery !== software.preInstallQuery ||
          installScript !== software.installScript ||
          postInstallScript !== software.postInstallScript ||
          uninstallScript !== software.uninstallScript) {
          await sails.helpers.flow.simultaneouslyForEach(unchangedTeamIds, async (teamApid)=>{
            await sails.helpers.http.sendHttpRequest.with({
              method: 'PATCH',
              baseUrl: sails.config.custom.fleetBaseUrl,
              url: `/api/v1/fleet/software/titles/${software.fleetApid}/package?team_id=${teamApid}`,
              enctype: 'multipart/form-data',
              headers: {
                Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
              },
              body: {
                team_id: teamApid, // eslint-disable-line camelcase
                pre_install_query: preInstallQuery, // eslint-disable-line camelcase
                install_script: installScript, // eslint-disable-line camelcase
                post_install_script: postInstallScript, // eslint-disable-line camelcase
                uninstall_script: uninstallScript, // eslint-disable-line camelcase
              }
            });
          });
        }
        // Now delete the software from teams it was removed from.
        for(let team of removedTeams) {
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            baseUrl: sails.config.custom.fleetBaseUrl,
            url: `/api/v1/fleet/software/titles/${software.fleetApid}/available_for_install?team_id=${team}`,
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
            }
          });
        }
        // If the software had been previously undeployed, delete the installer in s3 and the db record.
        if(software.id) {
          await sails.rm(sails.config.uploads.prefixForFileDeletion+software.uploadFd);
          await UndeployedSoftware.destroyOne({id: software.id});
        }

      } else if(software.teams && newTeamIds.length === 0) {
        // If this is a deployed software that is being unassigned, save information about the uploaded file in our s3 bucket.
        if(newSoftware) {
          // remove the old copy.
          // console.log('Removing old package for ',softwareName);
          await UndeployedSoftware.create({
            uploadFd: softwareFd,
            uploadMime: softwareMime,
            name: softwareName,
            platform: software.platform,
            postInstallScript,
            preInstallQuery,
            installScript,
            uninstallScript,
          });
        } else {
          // Save the information about the undeployed software in the app's DB.
          await UndeployedSoftware.create({
            uploadFd: softwareFd,
            uploadMime: softwareMime,
            name: software.name,
            platform: software.platform,
            postInstallScript,
            preInstallQuery,
            installScript,
            uninstallScript,
          });
        }
        // Now delete the software on the Fleet instance.
        for(let team of software.teams) {
          await sails.helpers.http.sendHttpRequest.with({
            method: 'DELETE',
            baseUrl: sails.config.custom.fleetBaseUrl,
            url: `/api/v1/fleet/software/titles/${software.fleetApid}/available_for_install?team_id=${team.fleetApid}`,
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
            }
          });
        }

      } else {
        // console.log('updating existing db record!');
        await UndeployedSoftware.updateOne({id: software.id}).set({
          name: softwareName,
          uploadMime: softwareMime,
          uploadFd: softwareFd,
          preInstallQuery,
          installScript,
          postInstallScript,
          uninstallScript,
        });
        // console.log('removing old stored copy of '+softwareName);
        await sails.rm(sails.config.uploads.prefixForFileDeletion+software.uploadFd);
      }
    } else if(preInstallQuery !== software.preInstallQuery ||
      installScript !== software.installScript ||
      postInstallScript !== software.postInstallScript ||
      uninstallScript !== software.uninstallScript) {
      // PATCH /api/v1/fleet/software/titles/:title_id/package
      if(newTeamIds.length !== 0) {
        for(let teamApid of newTeamIds){
          await sails.helpers.http.sendHttpRequest.with({
            method: 'PATCH',
            baseUrl: sails.config.custom.fleetBaseUrl,
            url: `/api/v1/fleet/software/titles/${software.fleetApid}/package?team_id=${teamApid}`,
            enctype: 'multipart/form-data',
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
            },
            body: {
              team_id: teamApid, // eslint-disable-line camelcase
              pre_install_query: preInstallQuery, // eslint-disable-line camelcase
              install_script: installScript, // eslint-disable-line camelcase
              post_install_script: postInstallScript, // eslint-disable-line camelcase
              uninstall_script: uninstallScript, // eslint-disable-line camelcase
            }
          });
        }
      } else if(software.id) {
        await UndeployedSoftware.updateOne({id: software.id}).set({
          preInstallQuery,
          installScript,
          postInstallScript,
          uninstallScript,
        });
      }
    }


    return;

  }


};
