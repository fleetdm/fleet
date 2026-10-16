module.exports = {


  friendlyName: 'Add user payload scope to declaration',


  description: 'Add "PayloadScope": "User" to a generated declaration whose type Apple only accepts on the macOS user channel.',


  extendedDescription:
`Fleet delivers a declaration to the device channel unless its PayloadScope says "User", and these types do
nothing there.  The generator only writes declarations for macOS, so the scope is always wanted.  The prompt asks
for it too, and this covers the runs that leave it out.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated declaration, as JSON.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Declaration with user payload scope',
      outputType: 'string',
    },

  },


  fn: async function ({profile}) {

    let path = require('path');

    let declaration;
    try {
      declaration = JSON.parse(profile);
    } catch (unusedErr) {
      return profile;
    }
    if(!_.isPlainObject(declaration) || declaration.PayloadScope !== undefined) {
      return profile;
    }

    let schemaFile = require(path.resolve(sails.config.appPath, 'profile-generator/schema/apple-ddm-declarations.json'));
    let entry = _.find(schemaFile.entries, {name: declaration.Type});
    if(!entry || !_.contains(entry.userChannelOnly || [], 'macOS')) {
      return profile;
    }

    // Placed after Type and Identifier, where Fleet's docs show it.
    let {Type, Identifier} = declaration;
    return JSON.stringify(Object.assign({Type, Identifier, PayloadScope: 'User'}, _.omit(declaration, ['Type', 'Identifier'])), null, 2);

  }


};
