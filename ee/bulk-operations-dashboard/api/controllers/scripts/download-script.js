module.exports = {


  friendlyName: 'Download script',


  description: 'Download script file (returning a stream).',


  inputs: {
    fleetApid: {
      type: 'number',
      description: 'The Fleet API ID of the script to download.',
    },
    id: {
      type: 'number',
      description: 'The database ID of the undeployed script to download'
    }
  },


  exits: {
    success: {
      outputFriendlyName: 'File',
      outputDescription: 'The streaming bytes of the file.',
      outputType: 'ref'
    },

    notFound: {
      description: 'No script exists with the specified ID or UUID.',
      responseType: 'notFound'
    },
  },


  fn: async function ({fleetApid, id}) {
    let filename;
    let download;
    if(id){
      let scriptToDownload = await UndeployedScript.findOne({id: id});

      filename = datePrefix + scriptToDownload.name + scriptToDownload.scriptType;
      scriptContents = scriptToDownload.scriptContents;
      if(scriptToDownload.scriptType === '.sh'){
        this.res.type('application/x-apple-aspen-config');
      } else {
        this.res.type('application/octet-stream');
      }
      download = scriptContents;
    } else {
      let scriptDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'GET',
        url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/scripts/${fleetApid}?alt=media`,
        headers: {
          Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
        }
      });
      console.log(scriptDownloadResponse.headers);
      let contentDispositionHeader = scriptDownloadResponse.headers['content-disposition'];
      let filenameMatch = contentDispositionHeader.replace(/^attachment;filename="(.+?)"$/, '$1');
      filename = filenameMatch;
      let contentType = scriptDownloadResponse.headers['content-type'];
      download = scriptDownloadResponse.body;
      this.res.type(contentType);
      this.res.attachment(filename);
    }
    // All done.
    return download;

  }


};
