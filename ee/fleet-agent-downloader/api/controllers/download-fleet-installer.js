module.exports = {


  friendlyName: 'Download fleet installer',


  description: 'Download fleet installer file (returning a stream).',


  exits: {
    success: {
      outputFriendlyName: 'Fleet installer',
      outputDescription: 'The streaming bytes of a Fleet installer',
      outputType: 'ref',
    }
  },


  fn: async function () {

    let downloading;

    downloading = await sails.startDownload(sails.config.custom.uploadedInstallerFileDescriptor, {bucket: sails.config.uploads.bucket});

    this.res.type('application/x-ms-installer');
    this.res.attachment('Fleet-installer.msi');

    return downloading;
  }


};
