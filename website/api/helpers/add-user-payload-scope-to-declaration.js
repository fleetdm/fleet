module.exports = {


  friendlyName: 'Add user payload scope to declaration',


  description: 'Add "PayloadScope": "User", and a note about it, to a generated declaration whose type Apple only accepts on the macOS user channel.',


  extendedDescription:
`Fleet delivers a declaration to the device channel unless its PayloadScope says "User", and these types do
nothing there.  The generator only writes declarations for macOS, so the scope is always wanted.  The prompt asks
for it and the note too, and this covers the runs that leave either out.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated declaration, as JSON.'
    },

    deliveryNotes: {
      type: 'string',
      defaultsTo: '',
      description: 'The delivery notes the model wrote.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Declaration with user payload scope',
      outputDescription: '{profile, deliveryNotes}',
      outputType: 'ref',
    },

  },


  fn: async function ({profile, deliveryNotes}) {

    let path = require('path');

    let declaration;
    try {
      declaration = JSON.parse(profile);
    } catch (unusedErr) {
      return {profile, deliveryNotes};
    }
    if(!_.isPlainObject(declaration)) {
      return {profile, deliveryNotes};
    }

    let schemaFile = require(path.resolve(sails.config.appPath, 'profile-generator/schema/apple-ddm-declarations.json'));
    let entry = _.find(schemaFile.entries, {name: declaration.Type});
    if(!entry || !_.contains(entry.userChannelOnly || [], 'macOS') || (declaration.PayloadScope !== undefined && declaration.PayloadScope !== 'User')) {
      return {profile, deliveryNotes};
    }

    if(declaration.PayloadScope === undefined) {
      // Placed after Type and Identifier, where Fleet's docs show it.
      let {Type, Identifier} = declaration;
      profile = JSON.stringify(Object.assign({Type, Identifier, PayloadScope: 'User'}, _.omit(declaration, ['Type', 'Identifier'])), null, 2);
    }
    if(!/\blabel/i.test(deliveryNotes)) {
      deliveryNotes = _.compact([deliveryNotes, 'This declaration fails on iPhones and iPads, so scope it to Macs with a label.']).join(' ');
    }
    return {profile, deliveryNotes};

  }


};
