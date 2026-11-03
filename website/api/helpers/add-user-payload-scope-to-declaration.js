module.exports = {


  friendlyName: 'Add user payload scope to declaration',


  description: 'Add "PayloadScope": "User", and a note about it, to a generated declaration whose type Apple only accepts on the macOS user channel.',


  extendedDescription:
`Fleet delivers a declaration to the device channel unless its PayloadScope says "User", and these types do
nothing there on a Mac.  Fleet fails a User-scoped declaration on iOS and iPadOS, so a request that names only
those platforms is left alone.  The prompt asks for the scope and the note too, and this covers the runs that
leave either out.`,


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

    platformsNamed: {
      type: ['string'],
      defaultsTo: [],
      description: 'The Apple platforms the instructions name, e.g. ["iOS"].  With none, the declaration is taken to be for Macs.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Declaration with user payload scope',
      outputDescription: '{profile, deliveryNotes}',
      outputType: 'ref',
    },

  },


  fn: async function ({profile, deliveryNotes, platformsNamed}) {

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
    let isForMacs = platformsNamed.length === 0 || _.contains(platformsNamed, 'macOS');
    if(!entry || !_.contains(entry.userChannelOnly || [], 'macOS') || !isForMacs) {
      return {profile, deliveryNotes};
    }

    // Any other scope delivers it to the device channel, where it does nothing.
    if(declaration.PayloadScope !== 'User') {
      // Placed after Type and Identifier, where Fleet's docs show it.
      let {Type, Identifier} = declaration;
      profile = JSON.stringify(Object.assign({Type, Identifier, PayloadScope: 'User'}, _.omit(declaration, ['Type', 'Identifier', 'PayloadScope'])), null, 2);
    }
    if(!/user channel/i.test(deliveryNotes)) {
      deliveryNotes = _.compact([deliveryNotes, 'This declaration needs the Mac\'s user channel, which Fleet creates for Macs that enroll automatically, and fails on iPhones and iPads, so scope it to those Macs with a label.']).join(' ');
    }
    return {profile, deliveryNotes};

  }


};
