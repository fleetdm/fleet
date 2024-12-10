module.exports = {


  friendlyName: 'Upload software',


  description: '',

  files: ['newSoftware'],

  inputs: {
    newSoftware: {
      type: 'ref',
      description: 'An Upstream with an incoming file upload.',
      required: true,
    },

    teams: {
      type: ['string'],
      description: 'An array of team IDs that this profile will be added to'
    },

    isDeployedToAllTeams: {
      type: 'boolean',
      description: 'Whether or not this software will be deployed to all teams.',
      extendedDescription: 'This determines whether or not the software will have a AllTeamsSoftware record created for it in the database and has no impact on the teams the software is transfered to in this action.',
      defaultsTo: false,
    }
  },


  exits: {
    success: {
      outputDescription: 'The new software has been uploaded',
      outputType: {},
    },

    softwareAlreadyExistsOnThisTeam: {
      description: 'A software with this name already exists on the Fleet Instance',
      statusCode: 409,
    },

    softwareUploadFailed: {
      description:'An unexpected error occurred communicating with the Fleet API'
    },

    couldNotReadVersion: {
      description:'Fleet could not read version information from the provided software installer.'
    }

  },


  fn: async function ({newSoftware, teams, isDeployedToAllTeams}) {
    let uploadedSoftware;
    if(teams.length === 0) {
      uploadedSoftware = await sails.uploadOne(newSoftware, {bucket: sails.config.uploads.bucketWithPostfix});
      let datelessFilename = uploadedSoftware.filename.replace(/^\d{4}-\d{2}-\d{2}\s/, '');
      // Build a dictonary of information about this software to return to the softwares page.
      let newSoftwareInfo = {
        name: datelessFilename,
        platform: _.endsWith(datelessFilename, '.deb') ? 'Linux' : _.endsWith(datelessFilename, '.pkg') ? 'macOS' : 'Windows',
        createdAt: Date.now(),
        uploadFd:  uploadedSoftware.fd,
        uploadMime: uploadedSoftware.type,
      };
      await UndeployedSoftware.create(newSoftwareInfo);
    } else {
      uploadedSoftware = await sails.uploadOne(newSoftware, {bucket: sails.config.uploads.bucketWithPostfix});
      for(let teamApid of teams) {
        var WritableStream = require('stream').Writable;
        await sails.cp(uploadedSoftware.fd, {bucket: sails.config.uploads.bucketWithPostfix}, {
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
                  let axios = require('axios');
                  let FormData = require('form-data');
                  let form = new FormData();
                  form.append('team_id', teamApid);
                  form.append('software', __newFile, {
                    filename: uploadedSoftware.filename,
                    contentType: 'application/octet-stream'
                  });
                  (async ()=>{
                    await axios.postForm(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
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
          }
        })
        .intercept({response: {status: 409}}, async (error)=>{// handles errors related to duplicate software items.
          await sails.rm(sails.config.uploads.prefixForFileDeletion+uploadedSoftware.fd);
          return {'softwareAlreadyExistsOnThisTeam': error};
        })
        .intercept({name: 'AxiosError', response: {status: 400}}, async (error)=>{// Handles errors related to malformed installer packages
          await sails.rm(sails.config.uploads.prefixForFileDeletion+uploadedSoftware.fd);
          let axiosError = error;
          if(axiosError.response.data) {
            if(axiosError.response.data.errors && _.isArray(axiosError.response.data.errors)){
              if(axiosError.response.data.errors[0] && axiosError.response.data.errors[0].reason) {
                let errorMessageFromFleetInstance = axiosError.response.data.errors[0].reason;
                if(_.startsWith(errorMessageFromFleetInstance, `Couldn't add. Fleet couldn't read the version`)){
                  return 'couldNotReadVersion';
                } else {
                  sails.log.warn(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API. Error returned from Fleet API: ${errorMessageFromFleetInstance} \n Axios error: ${require('util').inspect(error, {depth: 3})}`);
                  return {'softwareUploadFailed': error};
                }
              }
            }
          }
          sails.log.warn(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API, ${require('util').inspect(error, {depth: 3})}`);
          return {'softwareUploadFailed': error};
        })
        .intercept({name: 'AxiosError'}, async (error)=>{// Handles any other error.
          await sails.rm(sails.config.uploads.prefixForFileDeletion+uploadedSoftware.fd);
          sails.log.warn(`When attempting to upload a software installer, an unexpected error occurred communicating with the Fleet API, ${require('util').inspect(error, {depth: 3})}`);
          return {'softwareUploadFailed': error};
        });
      }
      if(isDeployedToAllTeams) {
        // IF the software is deployed to all teams, we'll send a request to the Fleet API to get information about the uploaded software.
        // This way, we can create an AllTeamsSoftware record for the newly uploaded installer.
        // Get all software for a team that the new software was uploaded to.
        let softwareResponse = await sails.helpers.http.get.with({
          url: `/api/latest/fleet/software/titles?team_id=${teams[0]}`,
          baseUrl: sails.config.custom.fleetBaseUrl,
          headers: {
            Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
          }
        })
        .timeout(120000)
        .retry(['requestFailed', {name: 'TimeoutError'}]);
        // Filter out software without installer packages.
        let softwareForThisTeam = _.filter(softwareResponse.software_titles, (software)=>{
          return !_.isEmpty(software.software_package);
        });
        // Find the software with an installer package that matches the uploaded installer's filename
        let softwareThatWasJustUploaded = _.find(softwareForThisTeam, (softwareWithInstaller)=>{
          return softwareWithInstaller.software_package.name === uploadedSoftware.filename;
        });
        // Create a new DB record for this software.
        await AllTeamsSoftware.create({
          fleetApid: softwareThatWasJustUploaded.id,
          teamApids: teams,
        });
      }

      // Remove the file from the s3 bucket after it has been sent to the Fleet server.
      await sails.rm(sails.config.uploads.prefixForFileDeletion+uploadedSoftware.fd);
    }
    return;

  }


};
