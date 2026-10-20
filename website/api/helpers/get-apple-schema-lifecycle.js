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
      extendedDescription: 'Leave out to look up the payload type or declaration type itself.  Matched case-insensitively, since a model writing a path may not keep Apple\'s casing.  Some of Apple\'s key names contain dots (com.apple.autologout.AutoLogOutDelay), so the whole path is tried as one key name before it is read as a nested path.',
    },

    isPathFromTheTop: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether keyPath is a key\'s full path from the top of the payload, so a bare name only matches a top-level key rather than the shallowest key with that name.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Lifecycle',
      outputDescription: '{found, supportedOS, keyPath?}: whether the payload type (and key) exists, its lifecycle by platform, e.g. {macOS: {introduced: "11.3", deprecated: "26.0", removed: "27.0"}, iOS: {introduced: "n/a"}}, and the key path as the schema spells it.',
    },

  },


  fn: function ({profileType, name, keyPath, isPathFromTheTop}) {

    let path = require('path');
    let filename = profileType === 'ddm' ? 'apple-ddm-declarations.json' : 'apple-payload-manifests.json';
    let schemaFile = require(path.resolve(sails.config.appPath, `profile-generator/schema/${filename}`));

    // Every entry with the name, since six .mobileconfig payload types share com.apple.MCX.
    let entries = _.filter(schemaFile.entries, (entry)=>{ return entry.name.toLowerCase() === name.toLowerCase(); });
    if(!keyPath) {
      return entries.length > 0 ? { found: true, supportedOS: entries[0].supportedOS } : { found: false };
    }

    let wholePath = keyPath.toLowerCase();
    // Breadth-first, so a bare name finds the shallowest key with it, and each key carries the lifecycle it
    // inherits.  Every key name is tried as the whole path before any path is tried as nested keys.
    let findKey = (isAMatch)=>{
      for (let entry of entries) {
        let queue = _.map(entry.keys, (key)=>{ return { key, keyNames: [key.key], inheritedSupportedOS: entry.supportedOS }; });
        while(queue.length > 0) {
          let {key, keyNames, inheritedSupportedOS} = queue.shift();
          let supportedOS = key.supportedOS || inheritedSupportedOS;
          if(isAMatch(keyNames)) {
            return { keyNames, supportedOS };
          }
          for (let subkey of key.subkeys || []) {
            queue.push({ key: subkey, keyNames: keyNames.concat([subkey.key]), inheritedSupportedOS: supportedOS });
          }
        }
      }
      return undefined;
    };
    let matchedByName = findKey((keyNames)=>{
      return (!isPathFromTheTop || keyNames.length === 1) && _.last(keyNames).toLowerCase() === wholePath;
    });
    if(matchedByName) {
      return { found: true, supportedOS: matchedByName.supportedOS, keyPath: _.last(matchedByName.keyNames) };
    }
    let matchedByPath = _.contains(keyPath, '.') ? findKey((keyNames)=>{
      return keyNames.length > 1 && _.map(keyNames, (keyName)=>{ return keyName.toLowerCase(); }).join('.') === wholePath;
    }) : undefined;
    if(matchedByPath) {
      return { found: true, supportedOS: matchedByPath.supportedOS, keyPath: matchedByPath.keyNames.join('.') };
    }
    return { found: false };

  }


};
