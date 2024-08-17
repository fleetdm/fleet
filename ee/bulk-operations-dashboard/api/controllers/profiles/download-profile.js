module.exports = {


  friendlyName: 'Download profile',


  description: 'Download profile file (returning a stream).',


  inputs: {
    id: {
      type: 'number',
      description: 'The database ID of the undeployed profile to download.'
    },
    uuid: {
      type: 'string',
      description: 'The uuid of a profile on a team.'
    },
  },


  exits: {
    success: {
      outputFriendlyName: 'File',
      outputDescription: 'The streaming bytes of the file.',
      outputType: 'ref'
    },

    notFound: {
      description: 'No profile exists with the specified ID or UUID.',
      responseType: 'notFound'
    },
  },


  fn: async function ({id, uuid}) {
    if(!uuid && !id){
      return this.res.badRequest();
    }
    let datePrefix = new Date().toISOString();
    datePrefix = datePrefix.split('T')[0] +'_';
    let profileContents;
    let filename;
    let download;
    if(id){
      let profileToDownload = await UndeployedProfile.findOne({id: id});

      filename = datePrefix + profileToDownload.name + profileToDownload.profileType;
      profileContents = profileToDownload.profileContents;
      if(profileToDownload.profileType === '.mobileconfig'){
        this.res.type('application/x-apple-aspen-config');
      } else {
        this.res.type('application/octet-stream');
      }
      download = profileContents;
    } else {
      let profileDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/configuration_profiles/${uuid}?alt=media`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      });
      let contentDispositionHeader = profileDownloadResponse.headers['content-disposition'];
      let filenameMatch = contentDispositionHeader.replace(/^attachment;filename="(.+?)"$/, '$1');
      filename = filenameMatch;
      let contentType = profileDownloadResponse.headers['content-type'];
      download = profileDownloadResponse.body;
      this.res.type(contentType);
    }
    this.res.attachment(filename);
    // All done.
    return download;

  }


};
