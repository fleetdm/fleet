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
    }
  },


  exits: {
    success: {
      outputDescription: 'The new software has been uploaded',
      outputType: {},
    },

    softwareWithThisNameAlreadyExists: {
      description: 'A software with this name already exists on the Fleet Instance',
      statusCode: 409,
    },

    noFileAttached: {
      description: 'No file was attached.',
      responseType: 'badRequest'
    },

    tooBig: {
      description: 'The file is too big.',
      responseType: 'badRequest'
    },
  },


  fn: async function ({newSoftware, teams}) {

    let util = require('util');
    let software = await sails.reservoir(newSoftware)
    .intercept('E_EXCEEDS_UPLOAD_LIMIT', 'tooBig')
    .intercept((err)=>new Error('The software upload failed. '+util.inspect(err)));
    if(!software) {
      throw 'noFileAttached';
    }
    console.log(software);
    // Get the file contents and filename.
    let softwareContents = software[0].contentBytes;
    // console.log(softwareContents);
    let softwareFilename = software[0].name;
    // Strip out any automatically added date prefixes from uploaded softwares.
    let datelessExtensionlessFilename = softwareFilename.replace(/^\d{4}-\d{2}-\d{2}\s/, '').replace(/\.[^/.]+$/, '');
    let extension = '.'+softwareFilename.split('.').pop();
    // Build a dictonary of information about this software to return to the softwares page.
    let newSoftwareInfo = {
      name: datelessExtensionlessFilename,
      platform: _.endsWith(softwareFilename, '.deb') ? 'Linux' : _.endsWith(softwareFilename, '.pkg') ? 'macOS' : 'Windows',
      softwareType: extension,
      createdAt: Date.now()
    };

    newSoftwareInfo.softwareContents = Buffer.from(softwareContents);
    await UndeployedSoftware.create(newSoftwareInfo).fetch();
    // All done.
    return;

  }


};
