module.exports = {


  friendlyName: 'Download software',


  description: 'Download software file (returning a stream).',


  inputs: {
    id: {
      type: 'number',
      description: 'The database ID of the undeployed software to download.'
    },
    uuid: {
      type: 'string',
      description: 'The uuid of a software on a team.'
    },
  },


  exits: {
    success: {
      outputFriendlyName: 'File',
      outputDescription: 'The streaming bytes of the file.',
      outputType: 'ref'
    },

    notFound: {
      description: 'No software exists with the specified ID or UUID.',
      responseType: 'notFound'
    },
  },


  fn: async function ({id, uuid}) {
    if(!uuid && !id){
      return this.res.badRequest();
    }
    let datePrefix = new Date().toISOString();
    datePrefix = datePrefix.split('T')[0] +'_';
    let softwareContents;
    let filename;
    let download;

    if(id){
      let softwareToDownload = await UndeployedSoftware.findOne({id: id});

      filename = datePrefix + softwareToDownload.name + softwareToDownload.softwareType;
      softwareContents = softwareToDownload.softwareContents;
      if(softwareToDownload.softwareType === '.msi'){
        this.res.type('application/x-msdownload');
      } else if(softwareToDownload.softwareType === '.exe') {
        this.res.type('application/x-msdos-program');
      } else if(softwareToDownload.softwareType === '.deb') {
        this.res.type('application/x-debian-package')
      } else if(softwareToDownload.softwareType === '.pkg') {
        this.res.type('application/octet-stream')
      }
      download = softwareContents;
    } else {

    }
    // All done.
    this.res.attachment(filename);
    // All done.
    return download;

  }


};
