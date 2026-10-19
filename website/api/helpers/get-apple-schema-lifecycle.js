module.exports = {


  friendlyName: 'Get Apple schema lifecycle',


  description: 'Look up the OS versions an Apple payload type, declaration type, or one of their keys was introduced, deprecated, and removed in.',


  sync: true,


  extendedDescription: 'Reads the schemas built by `sails run regenerate-apple-profile-schemas`.  A key with no lifecycle of its own has the lifecycle of the payload type, declaration type, or key it sits in, so the nearest one is returned.',


  inputs: {

    profileType: {
      type: 'string',
      required: true,
      isIn: ['mobileconfig', 'ddm'],
    },

    name: {
      type: 'string',
      required: true,
      description: 'The payload type or declaration type, e.g. com.apple.applicationaccess.',
    },

    keyPath: {
      type: 'string',
      description: 'A key, as a dotted path from the top of the payload (Deferrals.MajorPeriodInDays), or as a bare name found wherever it is in the payload.',
      extendedDescription: 'Leave out to look up the payload type or declaration type itself.  Matched case-insensitively, since a model writing a path may not keep Apple\'s casing.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Lifecycle',
      outputDescription: '{found, supportedOS, keyPath?}: whether the payload type (and key) exists, its lifecycle by platform, e.g. {macOS: {introduced: "11.3", deprecated: "26.0", removed: "27.0"}, iOS: {introduced: "n/a"}}, and the key path as the schema spells it.',
    },

  },


  fn: function ({profileType, name, keyPath}) {

    let path = require('path');
    let filename = profileType === 'ddm' ? 'apple-ddm-declarations.json' : 'apple-payload-manifests.json';
    let schemaFile = require(path.resolve(sails.config.appPath, `profile-generator/schema/${filename}`));

    // Every entry with the name, since six .mobileconfig payload types share com.apple.MCX.
    let entries = _.filter(schemaFile.entries, (entry)=>{ return entry.name.toLowerCase() === name.toLowerCase(); });
    if(!keyPath) {
      return entries.length > 0 ? { found: true, supportedOS: entries[0].supportedOS } : { found: false };
    }

    let segments = _.map(keyPath.split('.'), (segment)=>{ return segment.toLowerCase(); });
    for (let entry of entries) {
      // Breadth-first, so a bare name finds the shallowest key with it, and each key carries the lifecycle it
      // inherits.
      let queue = _.map(entry.keys, (key)=>{ return { key, keyNames: [key.key], inheritedSupportedOS: entry.supportedOS }; });
      while(queue.length > 0) {
        let {key, keyNames, inheritedSupportedOS} = queue.shift();
        let supportedOS = key.supportedOS || inheritedSupportedOS;
        let lowercaseKeyNames = _.map(keyNames, (keyName)=>{ return keyName.toLowerCase(); });
        let isAMatch = segments.length === 1 ? _.last(lowercaseKeyNames) === segments[0] : _.isEqual(lowercaseKeyNames, segments);
        if(isAMatch) {
          return { found: true, supportedOS, keyPath: segments.length === 1 ? _.last(keyNames) : keyNames.join('.') };
        }
        for (let subkey of key.subkeys || []) {
          queue.push({ key: subkey, keyNames: keyNames.concat([subkey.key]), inheritedSupportedOS: supportedOS });
        }
      }
    }
    return { found: false };

  }


};
