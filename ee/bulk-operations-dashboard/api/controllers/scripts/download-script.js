module.exports = {


  friendlyName: 'Download script',


  description: 'Download script file (returning a stream).',


  inputs: {
    id: {
      type: 'number',
      description: 'The fleet API ID of the script to download.',
      required: true,
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


  fn: async function ({id}) {
    let filename;
    let download;
    let profileDownloadResponse = await sails.helpers.http.sendHttpRequest.with({
      method: 'GET',
      url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/scripts/${id}?alt=media`,
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
    this.res.attachment(filename);
    // All done.
    return download;

  }


};
