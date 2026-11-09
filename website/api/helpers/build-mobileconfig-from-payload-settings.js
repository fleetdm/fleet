module.exports = {


  friendlyName: 'Build mobileconfig from payload settings',


  description: 'Write a .mobileconfig property list from the payloads chosen for it and their settings, typing each value from Apple\'s schema.',


  extendedDescription: `The model returns settings rather than the property list itself because most of the list is
bookkeeping -- tags, PayloadUUIDs, identifiers -- and writing it out was most of a .mobileconfig's generation time.`,


  sync: true,


  inputs: {

    payloads: {
      type: [{}],
      required: true,
      description: 'One entry per dict inside PayloadContent, e.g. [{payloadType: "com.apple.applicationaccess", settings: {allowCamera: false}}].',
      extendedDescription: 'An entry\'s payloadUUID, when given, is used as that dict\'s PayloadUUID, so another payload can reference it.',
    },

    profileSettings: {
      type: {},
      defaultsTo: {},
      description: 'Keys that belong on the root dict, e.g. {PayloadRemovalDisallowed: true}.',
    },

    profileIdentifier: {
      type: 'string',
      required: true,
      description: 'The root PayloadIdentifier.  Each payload\'s is this plus a suffix.',
      example: 'com.example.camera-disabled',
    },

    profileDisplayName: {
      type: 'string',
      required: true,
    },

    profileDescription: {
      type: 'string',
      defaultsTo: '',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Profile',
      outputDescription: '{profile, settingsEnforced: [{name, value}], unusableValues: ["why"]}.  A profile with any unusable value is not the one that was asked for.',
      outputType: 'ref',
    },

  },


  fn: function ({payloads, profileSettings, profileIdentifier, profileDisplayName, profileDescription}) {

    let crypto = require('crypto');
    let path = require('path');
    // Apple documents some payload types in several files, com.apple.MCX in six, so one name can have several entries.
    let schemaEntriesByName = _.groupBy(require(path.resolve(sails.config.appPath, 'profile-generator/schema/apple-payload-manifests.json')).entries, 'name');
    let topLevelKeySchemas = schemaEntriesByName.TopLevel ? schemaEntriesByName.TopLevel[0].keys : [];
    let commonPayloadKeyNames = _.pluck(schemaEntriesByName.CommonPayloadKeys ? schemaEntriesByName.CommonPayloadKeys[0].keys : [], 'key');
    // Written here for every payload dict, so a value the model gives for one of these is not kept.
    const KEYS_WRITTEN_FOR_EVERY_PAYLOAD = ['PayloadType', 'PayloadUUID', 'PayloadIdentifier', 'PayloadVersion'];
    // Left out of the settings an admin is shown, since they describe the profile rather than enforce anything.
    const KEYS_THAT_ENFORCE_NOTHING = ['PayloadType', 'PayloadUUID', 'PayloadIdentifier', 'PayloadVersion', 'PayloadDisplayName', 'PayloadDescription'];
    // A key that only means something on the root dict does nothing inside a payload, so one found there is
    // moved up, unless the payload's own type documents a key of that name.
    let rootOnlyKeyNames = _.difference(_.pluck(topLevelKeySchemas, 'key'), commonPayloadKeyNames.concat('PayloadContent'));

    let unusableValues = [];
    let rootSettings = _.omit(profileSettings, KEYS_WRITTEN_FOR_EVERY_PAYLOAD.concat(['PayloadContent', 'PayloadDisplayName', 'PayloadDescription']));
    let usedIdentifierSuffixes = [];
    let usedUuids = [];
    let payloadDicts = _.map(payloads, (payload)=>{
      let schemaEntries = schemaEntriesByName[payload.payloadType] || [];
      let keySchemas = _.flatten(_.map(schemaEntries, 'keys'));
      let ownKeyNames = _.pluck(keySchemas, 'key');
      let settings = _.omit(payload.settings, (value, key)=>{
        let belongsOnTheRoot = _.contains(rootOnlyKeyNames, key) && !_.contains(ownKeyNames, key);
        if(belongsOnTheRoot && rootSettings[key] === undefined) {
          rootSettings[key] = value;
        }
        return belongsOnTheRoot;
      });

      // Named for the entry that documents the most of what this payload sets.
      let namingEntry = _.max(schemaEntries, (entry)=>{ return _.intersection(_.map(entry.keys, 'key'), Object.keys(settings)).length; });
      let suffix = _.last(payload.payloadType.split('.')).toLowerCase();
      for (let count = 2; usedIdentifierSuffixes.includes(suffix); count++) {
        suffix = `${_.last(payload.payloadType.split('.')).toLowerCase()}-${count}`;
      }
      usedIdentifierSuffixes.push(suffix);

      // Either spelling names this payload's PayloadUUID, so another payload can reference it.  Its casing is
      // kept, since the reference repeats it as written.
      let payloadUuid = settings.PayloadUUID !== undefined ? settings.PayloadUUID : payload.payloadUUID;
      if(payloadUuid === undefined) {
        payloadUuid = crypto.randomUUID().toUpperCase();
      } else if(!_.isString(payloadUuid) || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(payloadUuid)) {
        unusableValues.push(`The ${payload.payloadType} payload's PayloadUUID, ${JSON.stringify(payloadUuid)}, is not a UUID.`);
      } else if(_.contains(usedUuids, payloadUuid.toUpperCase())) {
        unusableValues.push(`Two payloads share the PayloadUUID ${payloadUuid}.`);
      }
      usedUuids.push(String(payloadUuid).toUpperCase());

      let dict = _.extend({
        PayloadDisplayName: schemaEntries.length > 0 ? namingEntry.title : payload.payloadType,
        PayloadIdentifier: `${profileIdentifier}.${suffix}`,
        PayloadType: payload.payloadType,
        PayloadUUID: payloadUuid,
        PayloadVersion: 1,
      }, _.omit(settings, KEYS_WRITTEN_FOR_EVERY_PAYLOAD));
      return { dict, keySchemas };
    });

    let rootDict = _.extend({ PayloadContent: _.map(payloadDicts, 'dict') }, rootSettings, {
      PayloadDescription: profileDescription || undefined,
      PayloadDisplayName: profileDisplayName,
      PayloadIdentifier: profileIdentifier,
      PayloadType: 'Configuration',
      PayloadUUID: crypto.randomUUID().toUpperCase(),
      PayloadVersion: 1,
    });
    let rootKeySchemas = _.reject(topLevelKeySchemas, {key: 'PayloadContent'}).concat({ key: 'PayloadContent', type: 'array' });
    let lines = renderValue(rootDict, { type: 'dictionary', subkeys: rootKeySchemas }, 0, '', unusableValues, {
      // Each payload dict is typed from its own payload type's keys, which the root's schema cannot express.
      keySchemasForPayloadAt: (idx)=>{ return payloadDicts[idx].keySchemas; },
      pathForPayloadAt: (idx)=>{ return `${payloads[idx].payloadType}/`; },
    });

    // A null or undefined setting is left out of the profile, so it is left out of what the profile enforces too.
    let settingsEnforced = _.map(_.filter(_.flatten(_.map(payloadDicts, 'dict').concat(_.omit(rootDict, 'PayloadContent')).map((dict)=>{
      return _.pairs(_.omit(dict, KEYS_THAT_ENFORCE_NOTHING));
    })), ([, value])=>{ return value !== undefined && value !== null; }), ([name, value])=>{
      return { name, value: _.isObject(value) ? JSON.stringify(value) : String(value) };
    });

    return {
      profile: [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">',
        '<plist version="1.0">',
      ].concat(lines, ['</plist>', '']).join('\n'),
      settingsEnforced,
      unusableValues,
    };

  }


};


// How a reason names what a key takes.
const WHAT_EACH_SCHEMA_TYPE_TAKES = {
  boolean: 'true or false',
  integer: 'a whole number',
  real: 'a number',
  string: 'text',
  data: 'base64 data',
  date: 'a date',
  array: 'a list',
  dictionary: 'a dictionary',
};


/**
 * @param  {String} keyPath
 * @param  {String} schemaType
 * @param  {*} value
 * @returns {String}  Why the value cannot be written for that key.
 */
function whyValueDoesNotFit(keyPath, schemaType, value) {
  return `${keyPath} takes ${WHAT_EACH_SCHEMA_TYPE_TAKES[schemaType]}, not ${_.trunc(JSON.stringify(value), {length: 60})}.`;
}


/**
 * @param  {String} keyPath  Where the parent sits; a payload's own path ends in "/".
 * @param  {String} key
 * @returns {String}  Where the child sits, e.g. com.apple.applicationaccess/allowCamera or Services.Camera[0].
 */
function childPath(keyPath, key) {
  return !keyPath || _.endsWith(keyPath, '/') ? `${keyPath}${key}` : `${keyPath}.${key}`;
}


/**
 * The property list lines for one value, typed by the schema key it was set on.
 *
 * JSON cannot tell a real from an integer that happens to be whole, or a date or base64 data from a
 * string, so the schema's type decides wherever there is one, and a value that does not fit it is unusable
 * rather than written as whatever JSON made it: a boolean key set to "disabled" deploys and does nothing.
 * A key the schema does not list, like any key of a third-party preference domain, is typed the way JSON
 * typed it.
 *
 * @param  {*} value
 * @param  {Dictionary} keySchema  The schema entry for the key this value was set on, if there is one.
 * @param  {Number} depth
 * @param  {String} keyPath  Where the value sits, for the reason a value cannot be written.
 * @param  {Array} unusableValues  Collects why a value cannot be written.
 * @param  {Dictionary} root  {keySchemasForPayloadAt, pathForPayloadAt}, only while rendering the root dict.
 * @returns {Array}  Lines, indented for their depth.
 */
function renderValue(value, keySchema, depth, keyPath, unusableValues, root) {
  let indent = '\t'.repeat(depth);
  let schemaType = keySchema ? keySchema.type : undefined;
  let subkeys = keySchema && keySchema.subkeys ? keySchema.subkeys : [];

  if(_.isArray(value)) {
    if(WHAT_EACH_SCHEMA_TYPE_TAKES[schemaType] && schemaType !== 'array') {
      unusableValues.push(whyValueDoesNotFit(keyPath, schemaType, value));
    }
    let lines = [`${indent}<array>`];
    _.each(value, (item, idx)=>{
      let itemSchema = root ? { type: 'dictionary', subkeys: root.keySchemasForPayloadAt(idx) } : subkeys[0];
      lines = lines.concat(renderValue(item, itemSchema, depth + 1, root ? root.pathForPayloadAt(idx) : `${keyPath}[${idx}]`, unusableValues));
    });
    return lines.concat(`${indent}</array>`);
  }

  if(_.isPlainObject(value)) {
    if(WHAT_EACH_SCHEMA_TYPE_TAKES[schemaType] && schemaType !== 'dictionary') {
      unusableValues.push(whyValueDoesNotFit(keyPath, schemaType, value));
    }
    let lines = [`${indent}<dict>`];
    for (let key of Object.keys(value)) {
      if(value[key] === undefined || value[key] === null) {
        continue;
      }
      // Apple spells keys inconsistently, so the schema lookup tolerates a casing mismatch for typing's sake.
      // The key itself is written exactly as given.
      let childSchema = _.find(subkeys, {key}) ||
        _.find(subkeys, (subkey)=>{ return subkey.key.toLowerCase() === key.toLowerCase(); }) ||
        _.find(subkeys, {key: 'ANY'});
      lines.push(`${indent}\t<key>${escapeXml(key)}</key>`);
      lines = lines.concat(renderValue(value[key], childSchema, depth + 1, childPath(keyPath, key), unusableValues, key === 'PayloadContent' ? root : undefined));
    }
    return lines.concat(`${indent}</dict>`);
  }

  let plistType = plistTypeFor(value, schemaType);
  if(!plistType) {
    unusableValues.push(whyValueDoesNotFit(keyPath, schemaType, value));
    plistType = plistTypeFor(value, undefined);
  }
  switch (plistType) {
    case 'boolean': return [`${indent}<${value === true || value === 1 || /^(true|1)$/i.test(String(value)) ? 'true' : 'false'}/>`];
    case 'integer': return [`${indent}<integer>${Math.trunc(Number(value))}</integer>`];
    case 'real': return [`${indent}<real>${Number(value)}</real>`];
    case 'data': return [`${indent}<data>${base64DataFor(value)}</data>`];
    case 'date': {
      let date = new Date(value);
      if(isNaN(date.getTime())) {
        unusableValues.push(whyValueDoesNotFit(keyPath, 'date', value));
        return [`${indent}<date></date>`];
      }
      // The only form a property list date takes: no fractional seconds, always UTC.
      return [`${indent}<date>${date.toISOString().replace(/\.\d{3}Z$/, 'Z')}</date>`];
    }
    default: return [`${indent}<string>${escapeXml(String(value))}</string>`];
  }
}


/**
 * The property list type for a scalar value.
 *
 * @param  {*} value
 * @param  {String} schemaType
 * @returns {String|undefined}  boolean, integer, real, data, date, or string; undefined when the schema
 *   gives a type the value does not fit.
 */
function plistTypeFor(value, schemaType) {
  let isNumeric = (_.isNumber(value) || (_.isString(value) && value.trim() !== '')) && _.isFinite(Number(value));
  switch (schemaType) {
    case 'boolean':
      if(_.isBoolean(value) || value === 0 || value === 1 || /^(true|false|0|1)$/i.test(String(value))) { return 'boolean'; }
      break;
    case 'integer':
      if(isNumeric && Number.isInteger(Number(value))) { return 'integer'; }
      break;
    case 'real':
      if(isNumeric) { return 'real'; }
      break;
    case 'string':
      return 'string';
    case 'data':
    case 'date':
      if(_.isString(value)) { return schemaType; }
      break;
  }
  if(WHAT_EACH_SCHEMA_TYPE_TAKES[schemaType]) {
    return undefined;
  }
  // No schema type, or one like "any" that accepts whatever JSON gave.
  if(_.isBoolean(value)) { return 'boolean'; }
  if(_.isNumber(value)) { return Number.isInteger(value) ? 'integer' : 'real'; }
  return 'string';
}


/**
 * The base64 text a <data> element holds.
 *
 * Apple types shared secrets and certificates as data, which the model is asked to give as base64.  A PEM
 * certificate's armor is not part of that base64, and a value that is not base64 at all is the plain text the
 * model forgot to encode, so it is encoded here.
 *
 * @param  {String} value
 * @returns {String}
 */
function base64DataFor(value) {
  let withoutArmor = String(value).replace(/-----(?:BEGIN|END)[^-]*-----/g, '').replace(/\s+/g, '');
  let isBase64 = withoutArmor.length > 0 && withoutArmor.length % 4 === 0 && /^[A-Za-z0-9+/]+={0,2}$/.test(withoutArmor);
  return isBase64 ? withoutArmor : Buffer.from(String(value), 'utf8').toString('base64');
}


/**
 * @param  {String} text
 * @returns {String}  text, safe inside a property list element.
 */
function escapeXml(text) {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
