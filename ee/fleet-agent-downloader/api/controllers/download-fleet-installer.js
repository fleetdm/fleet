module.exports = {


  friendlyName: 'Download fleet installer',


  description: 'Download fleet installer file (returning a stream).',


  inputs: {

  },


  exits: {
    success: {
      outputFriendlyName: 'Fleet installer',
      outputDescription: 'The streaming bytes of a Fleet installer',
      outputType: 'ref',
    }
  },


  fn: async function () {

    let downloading;

    downloading = await sails.startDownload('f6e1476e-f677-4f6d-9aaa-d80091723891.upload', {bucket: sails.config.uploads.bucket});
    this.res.type('application/octet-stream');
    this.res.attachment('Fleet-installer.msi');

    return downloading;
  }


};
