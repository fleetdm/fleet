module.exports = {


  friendlyName: 'Add lifecycle notes to Apple profile',


  description: 'Write the delivery notes a generated Apple profile needs about OS versions: what it uses that Apple has deprecated or removed, and when a declaration needs a newer OS than the .mobileconfig that could do the same.',


  extendedDescription: `Worked out from Apple's lifecycle data rather than left to the model, which does not reliably know
which keys Apple has retired, and wrote nothing about a removed software update key in probes.`,


  sync: true,


  inputs: {

    profileType: {
      type: 'string',
      required: true,
      isIn: ['mobileconfig', 'ddm'],
    },

    profile: {
      type: 'string',
      required: true,
      description: 'The generated .mobileconfig, or the generated declaration as JSON.',
    },

    platformsNamed: {
      type: ['string'],
      defaultsTo: [],
      description: 'The Apple platforms the instructions name.  With none, every platform the profile applies to is reported on.',
    },

    mobileconfigAlternative: {
      type: ['string'],
      defaultsTo: [],
      description: 'For a declaration, the .mobileconfig keys that could have done the same, as payloadType/key, e.g. com.apple.mobiledevice.passwordpolicy/minLength.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Delivery notes',
      outputDescription: 'Sentences to add to the profile\'s delivery notes, e.g. ["com.apple.SoftwareUpdate stops working in macOS 27.0."].',
    },

  },


  fn: function ({profileType, profile, platformsNamed, mobileconfigAlternative}) {

    let lifecycleOf = (name, keyPath)=>{
      return sails.helpers.getAppleSchemaLifecycle.with({ profileType, name, keyPath });
    };
    let platformsToReportOn = (supportedOS)=>{
      let platformsAvailableOn = _.filter(Object.keys(supportedOS || {}), (platformName)=>{
        return supportedOS[platformName].introduced && supportedOS[platformName].introduced !== 'n/a';
      });
      return platformsNamed.length > 0 ? _.intersection(platformsAvailableOn, platformsNamed) : platformsAvailableOn;
    };

    // What the profile uses, by the payload type or declaration type it is in.
    let keyNamesByTypeName = {};
    if(profileType === 'mobileconfig') {
      let typeNames = _.without(_.uniq(_.map(profile.match(/<key>\s*PayloadType\s*<\/key>\s*<string>[^<]+<\/string>/g) || [], (match)=>{
        return match.replace(/^[\s\S]*<string>\s*|\s*<\/string>$/g, '');
      })), 'Configuration');
      let keyNames = _.uniq(_.map(profile.match(/<key>[^<]+<\/key>/g) || [], (match)=>{ return match.replace(/<\/?key>/g, '').trim(); }));
      for (let typeName of typeNames) {
        keyNamesByTypeName[typeName] = keyNames;
      }
    } else {
      let declaration;
      try {
        declaration = JSON.parse(profile);
      } catch (unusedErr) {
        return [];
      }
      let keyPaths = [];
      let collectKeyPaths = (value, pathSoFar)=>{
        if(!_.isObject(value) || _.isArray(value)) {
          return;
        }
        for (let keyName of Object.keys(value)) {
          keyPaths.push(pathSoFar.concat([keyName]).join('.'));
          collectKeyPaths(value[keyName], pathSoFar.concat([keyName]));
        }
      };
      collectKeyPaths(declaration.Payload, []);
      keyNamesByTypeName[declaration.Type] = keyPaths;
    }

    // Everything deprecated or removed on a platform being reported on.  A key is only reported where its
    // lifecycle differs from its payload type's, since a payload type that is removed takes its keys with it.
    let retirements = [];
    let minimumVersionByPlatform = {};
    for (let typeName of Object.keys(keyNamesByTypeName)) {
      let typeLifecycle = lifecycleOf(typeName);
      if(!typeLifecycle.found) {
        continue;// A preference domain with no manifest, or a third-party one: nothing is known about it.
      }
      let reportOn = platformsToReportOn(typeLifecycle.supportedOS);
      for (let platformName of reportOn) {
        let versions = typeLifecycle.supportedOS[platformName];
        retirements.push({ name: typeName, platformName, deprecated: versions.deprecated, removed: versions.removed });
        minimumVersionByPlatform[platformName] = latestVersion([minimumVersionByPlatform[platformName], versions.introduced]);
      }
      for (let keyName of keyNamesByTypeName[typeName]) {
        let keyLifecycle = lifecycleOf(typeName, keyName);
        if(!keyLifecycle.found) {
          continue;
        }
        for (let platformName of reportOn) {
          let versions = (keyLifecycle.supportedOS || {})[platformName] || {};
          let typeVersions = typeLifecycle.supportedOS[platformName];
          // Only what the key adds: when the payload type is removed, its own note covers every key in it.
          let deprecated = versions.deprecated !== typeVersions.deprecated ? versions.deprecated : undefined;
          let removed = versions.removed !== typeVersions.removed ? versions.removed : undefined;
          if(!typeVersions.removed && (deprecated || removed)) {
            retirements.push({ name: keyName, platformName, deprecated, removed });
          }
          if(versions.introduced && versions.introduced !== 'n/a') {
            minimumVersionByPlatform[platformName] = latestVersion([minimumVersionByPlatform[platformName], versions.introduced]);
          }
        }
      }
    }

    let notes = [];

    // One sentence per OS version, naming everything that version retires, with the platforms that retire the
    // same things in the same version run together ("iOS and macOS 27.0").  Removed outranks deprecated:
    // something that has stopped working does not also need to be called deprecated.
    let sentencesFor = (retirementsOfOneKind, versionOf)=>{
      let namesByPlatformAndVersion = _.groupBy(retirementsOfOneKind, (retirement)=>{ return `${retirement.platformName} ${versionOf(retirement)}`; });
      let platformsByVersionAndNames = {};
      for (let platformAndVersion of Object.keys(namesByPlatformAndVersion)) {
        let retirementsHere = namesByPlatformAndVersion[platformAndVersion];
        let key = JSON.stringify([versionOf(retirementsHere[0]), _.uniq(_.pluck(retirementsHere, 'name'))]);
        platformsByVersionAndNames[key] = (platformsByVersionAndNames[key] || []).concat([retirementsHere[0].platformName]);
      }
      return _.map(Object.keys(platformsByVersionAndNames), (key)=>{
        let [version, names] = JSON.parse(key);
        return { names, where: `${listed(platformsByVersionAndNames[key])} ${version}` };
      });
    };
    for (let {names, where} of sentencesFor(_.filter(retirements, 'removed'), (retirement)=>{ return retirement.removed; })) {
      notes.push(`${listed(names)} ${names.length === 1 ? 'stops' : 'stop'} working in ${where}.`);
    }
    for (let {names, where} of sentencesFor(_.filter(retirements, (retirement)=>{ return retirement.deprecated && !retirement.removed; }), (retirement)=>{ return retirement.deprecated; })) {
      notes.push(`Apple deprecated ${listed(names)} in ${where}; ${names.length === 1 ? 'it still works' : 'they still work'} for now.`);
    }

    // A declaration only gets a minimum OS when the .mobileconfig it was chosen over would have worked on
    // an older one -- otherwise the minimum is no news.
    if(profileType === 'ddm' && mobileconfigAlternative.length > 0) {
      let alternativeMinimumByPlatform = {};
      for (let alternative of mobileconfigAlternative) {
        let [payloadType, keyName] = alternative.split('/');
        let alternativeLifecycle = sails.helpers.getAppleSchemaLifecycle.with({ profileType: 'mobileconfig', name: payloadType, keyPath: keyName });
        for (let platformName of Object.keys(alternativeLifecycle.supportedOS || {})) {
          let introduced = alternativeLifecycle.supportedOS[platformName].introduced;
          if(introduced && introduced !== 'n/a') {
            alternativeMinimumByPlatform[platformName] = latestVersion([alternativeMinimumByPlatform[platformName], introduced]);
          }
        }
      }
      let newerThanTheAlternative = _.filter(Object.keys(minimumVersionByPlatform), (platformName)=>{
        return alternativeMinimumByPlatform[platformName] && latestVersion([minimumVersionByPlatform[platformName], alternativeMinimumByPlatform[platformName]]) !== alternativeMinimumByPlatform[platformName];
      });
      if(newerThanTheAlternative.length > 0) {
        notes.push(`This declaration needs ${listed(_.map(newerThanTheAlternative, (platformName)=>{ return `${platformName} ${minimumVersionByPlatform[platformName]}`; }))} or later.`);
      }
    }

    return notes;

  }


};


/**
 * The latest of some dotted version numbers, ignoring missing ones.
 *
 * @param  {Array} versions  e.g. ['14.0', undefined, '15.1']
 * @returns {String|undefined}  e.g. '15.1'
 */
function latestVersion(versions) {
  let parts = (version)=>{ return _.map(String(version).split('.'), Number); };
  return _.reduce(_.compact(versions), (latest, version)=>{
    if(latest === undefined) {
      return version;
    }
    let [a, b] = [parts(latest), parts(version)];
    for (let i = 0; i < Math.max(a.length, b.length); i++) {
      if((a[i] || 0) !== (b[i] || 0)) {
        return (b[i] || 0) > (a[i] || 0) ? version : latest;
      }
    }
    return latest;
  }, undefined);
}


/**
 * Names joined the way a sentence lists them: "a", "a and b", "a, b, and c".
 *
 * @param  {Array} names
 * @returns {String}
 */
function listed(names) {
  if(names.length <= 2) {
    return names.join(' and ');
  }
  return `${_.initial(names).join(', ')}, and ${_.last(names)}`;
}
