module.exports = {


  friendlyName: 'Edit software',


  description: '',

  files: ['newSoftware'],

  inputs: {
    newSoftware: {
      type: 'ref',
      description: 'The optional streaming bytes of a new software version.'
    },
    newTeamIds: {
      type: ['ref'],
      description: 'The new teams that this software will be deployed to.'
    },
    software: {
      type: {},
      description: ''
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
      description: 'The software already exists on the Fleet server.'
    }
  },


  fn: async function ({newSoftware, newTeamIds, software, preInstallQuery, installScript, postInstallScript, uninstallScript}) {
    if(newSoftware.isNoop) {
      newSoftware.noMoreFiles();
      newSoftware = undefined;
    }
    var WritableStream = require('stream').Writable;
    let { Readable } = require('stream');
    let axios = require('axios');
    // Cast the strings in the newTeamIds array to numbers.
    newTeamIds = newTeamIds.map(Number);
    let currentSoftwareTeamIds = _.pluck(software.teams, 'fleetApid');
    // If the teams have changed, or a new installer package was provided, we'll need to upload the package to an s3 bucket to deploy it to other teams.
    if(_.xor(newTeamIds, currentSoftwareTeamIds).length !== 0 || newSoftware) {
      let softwareFd;
      let softwareName;
      let softwareMime;
      if(software.teams && !newSoftware) {
        console.log('Editing deployed software!');
        // This software is deployed.
        // get software from Fleet instance and upload to s3.
        let teamIdToGetInstallerFrom = software.teams[0].fleetApid;
        let downloadApiUrl = `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/${software.fleetApid}/package?alt=media&team_id=${teamIdToGetInstallerFrom}`;
        // console.time('fleetInstance»s3(cp)');
        // let foo = await sails.cp(downloadApiUrl, {
        //   adapter: () => {
        //     return {
        //       ls: undefined,
        //       rm: undefined,
        //       receive: undefined,
        //       read: (downloadApiUrl) => {
        //         // Create a readable stream
        //         const readable = new Readable({
        //           read() {
        //             // Empty _read method; we'll handle data pushing with events below
        //           }
        //         });

        //         // Now we'll fetch the data asynchronously and pipe it into the readable stream
        //         (async () => {
        //           try {
        //             const streamResponse = await axios({
        //               url: downloadApiUrl,
        //               method: 'GET',
        //               responseType: 'stream',
        //               headers: {
        //                 Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
        //               },
        //             });

        //             console.log('Received stream from API, piping data...');

        //             // Pipe data from the response stream into the readable stream
        //             streamResponse.data.on('data', (chunk) => {
        //               const canContinue = readable.push(chunk);
        //               if (!canContinue) {
        //                 streamResponse.data.pause();  // Pause if we can't push more data
        //               }
        //             });

        //             // Resume the stream when readable is ready
        //             readable.on('drain', () => {
        //               streamResponse.data.resume();
        //             });

        //             // When the source stream ends, we signal end of the readable stream
        //             streamResponse.data.on('end', () => {
        //               readable.push(null); // Signal end of stream
        //             });

        //             // Handle any errors from the source stream
        //             streamResponse.data.on('error', (err) => {
        //               readable.emit('error', err); // Propagate the error to the readable stream
        //             });

        //           } catch (error) {
        //             console.error('Error during read operation:', error);
        //             readable.emit('error', new Error('Failed to download file: ' + error.message));
        //           }
        //         })();

        //         return readable;
        //       },
        //     };
        //   },
        // },{});
        // console.log(foo);
        // console.timeEnd('fleetInstance»s3(cp)');
        console.time('fleetInstance»s3(upload+stream)');
        let softwareStream = await sails.helpers.http.getStream.with({
          url: dowloadApiUrl,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
          }
        });
        let tempUploadedSoftware = await sails.uploadOne(softwareStream);
        console.timeEnd('fleetInstance»s3(upload+stream)');
        console.log(tempUploadedSoftware);
        softwareFd = tempUploadedSoftware.fd;
        softwareName = software.name;
        softwareMime = tempUploadedSoftware.type;
      } else if(newSoftware) {
        console.log('replacing software package!');
        let uploadedSoftware = await sails.uploadOne(newSoftware);
        softwareFd = uploadedSoftware.fd;
        softwareName = uploadedSoftware.filename;
        softwareMime = uploadedSoftware.type;
        let newSoftwareExtension = '.'+softwareName.split('.').pop();
        let existingSoftwareExtension = '.'+software.name.split('.').pop();
        console.log('new extension!', newSoftwareExtension)
        console.log('old extension!', existingSoftwareExtension)
        if(newSoftwareExtension !== existingSoftwareExtension) {
          await sails.rm(softwareFd);
          throw {wrongInstallerExtension: `Couldn't edit ${software.name}. The selected package should be a ${existingSoftwareExtension} file`}
        }
      } else {
        console.log('Editing undeployed software!');
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
        for(let team of addedTeams) {
          // Send an api request to send the file to the Fleet server for each added team.
          await sails.cp(softwareFd, {},
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
                      form.append('team_id', team); // eslint-disable-line camelcase
                      form.append('install_script', installScript);
                      form.append('post_install_script', postInstallScript);
                      form.append('pre_install_query', preInstallQuery);
                      form.append('uninstall_script', uninstallScript);
                      form.append('software', __newFile, {
                        filename: software.name,
                        contentType: 'application/octet-stream'
                      });
                      (async ()=>{
                        try {
                          await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
                            headers: {
                              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                              ...form.getHeaders()
                            },
                          })
                        } catch(error){
                          throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
                        }
                      })()
                      .then(()=>{
                        console.log('ok supposedly a file is finished uploading');
                        doneWithThisFile();
                      })
                      .catch((err)=>{
                        doneWithThisFile(err);
                      });
                    };//ƒ
                    return receiver__;
                  }
                }
              },
            }).intercept((err)=>{
              throw 'softwareUploadFailed'
            });
          }// After every new team this is deployed to.
          if(newSoftware) {
            // If a new installer package was provided, send patch requests to update the installer package on teams that it is already deployed to.
            for(let teamApid of unchangedTeamIds) {
              console.log(`Adding new version of ${softwareName} to teamId ${teamApid}`);
              await sails.helpers.http.sendHttpRequest.with({
                method: 'DELETE',
                baseUrl: sails.config.custom.fleetBaseUrl,
                url: `/api/v1/fleet/software/titles/${software.fleetApid}/available_for_install?team_id=${teamApid}`,
                headers: {
                  Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                }
              });
              await sails.cp(softwareFd, {},
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
                      form.append('team_id', teamApid); // eslint-disable-line camelcase
                      form.append('install_script', installScript);
                      form.append('post_install_script', postInstallScript);
                      form.append('pre_install_query', preInstallQuery);
                      form.append('uninstall_script', uninstallScript);
                      form.append('software', __newFile, {
                        filename: software.name,
                        contentType: 'application/octet-stream'
                      });
                      (async ()=>{
                        try {
                          await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
                            headers: {
                              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                              ...form.getHeaders()
                            },
                          })
                        } catch(error){
                          throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
                        }
                      })()
                      .then(()=>{
                        console.log('ok supposedly a file is finished uploading');
                        doneWithThisFile();
                      })
                      .catch((err)=>{
                        doneWithThisFile(err);
                      });
                    };//ƒ
                    return receiver__;
                  }
                }
              },
            });
          }// After every team the software is currently deployed to.
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
          await sails.rm(software.uploadFd);
          await UndeployedSoftware.destroyOne({id: software.id});
        }

      } else if(software.teams && newTeamIds.length === 0) {
        // If this is a deployed software that is being unassigned, save information about the uploaded file in our s3 bucket.
        if(newSoftware) {
          // remove the old copy.
          console.log('Removing old package for ',softwareName);
          await UndeployedSoftware.create({
            uploadFd: softwareFd,
            uploadMime: softwareMime,
            name: softwareName,
            platform: _.endsWith(softwareName, '.deb') ? 'Linux' : _.endsWith(softwareName, '.pkg') ? 'macOS' : 'Windows',
          })
        } else {
          console.log('undeploying ',softwareName);
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
        }

      } else {
        // console.log('updating existing db record!');
        await UndeployedSoftware.updateOne({id: software.id}).set({
          name: softwareName,
          uploadMime: softwareMime,
          uploadFd: softwareFd,
        })
      }
    } else if(preInstallQuery !== software.preInstallQuery ||
      installScript !== software.installScript ||
      postInstallScript !== software.postInstallScript ||
      uninstallScript !== software.uninstallScript) {
      // PATCH /api/v1/fleet/software/titles/:title_id/package
      if(newTeamIds.length !== 0) {
        for(let teamApid of newTeamIds){
          let updatedSoftware = await sails.helpers.http.sendHttpRequest.with({
            method: 'PATCH',
            baseUrl: sails.config.custom.fleetBaseUrl,
            url: `/api/v1/fleet/software/titles/${software.fleetApid}/package?team_id=${teamApid}`,
            enctype: 'multipart/form-data',
            headers: {
              Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
            },
            body: {
              team_id: teamApid,
              pre_install_query: preInstallQuery,
              install_script: installScript,
              post_install_script: postInstallScript,
              uninstall_script: uninstallScript,
            }
          })
        }
      } else if(software.id) {
        await UndeployedSoftware.updateOne({id: software.id}).set({
          preInstallQuery,
          installScript,
          postInstallScript,
          uninstallScript,
        })
      }
    }


    return;

  }


};
