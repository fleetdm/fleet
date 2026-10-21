module.exports = {


  friendlyName: 'Build Windows profile from commands',


  description: 'Write a Windows CSP profile from the commands chosen for it, taking each node\'s Format from Microsoft\'s CSP reference.',


  extendedDescription: `The model returns commands rather than the profile itself because most of the profile is
markup -- Item, Meta, Target, CDATA, and a Wi-Fi network's whole WLAN profile -- and writing it out was most of a Windows
profile's generation time.  Each Format comes from the reference rather than the model, which got int-or-bool wrong more
than anything else, and a value that does not fit its node's Format makes the command unusable rather than being written.`,


  sync: true,


  inputs: {

    commands: {
      type: [{}],
      required: true,
      description: 'e.g. [{verb: "Replace", locURI: "./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera", data: 0}, {verb: "Add", wifi: {ssid: "CorpNet", security: "WPA2Personal", passphrase: "…"}}]',
      extendedDescription: 'A command\'s format is used only for a node the reference does not list.',
    },

    atomic: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether to wrap every command in one <Atomic>.',
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Windows profile',
      outputDescription: '{profile, settingsEnforced: [{name, value}], deliveryNotes: ["sentence"], unusableCommands: ["why"]}.  A profile with any unusable command is not the one that was asked for.',
      outputType: 'ref',
    },

  },


  fn: function ({commands, atomic}) {

    let path = require('path');
    let nodes = require(path.resolve(sails.config.appPath, 'profile-generator/schema/windows-csp-nodes.json')).nodes;

    // LocURIs are compared without their ./Device or ./User scope, which the reference writes only for some CSPs,
    // and without case.  A {Braced} segment is a dynamic node, matching any one segment.
    let comparableLocUri = (locUri)=>{ return String(locUri).trim().replace(/^\.\/(?:Device|User)\/Vendor\//i, './Vendor/').toLowerCase(); };
    // The type names a model reaches for in place of a SyncML format.
    let formatsByAlias = { integer: 'int', boolean: 'bool', string: 'chr' };

    let nodesByComparableLocUri = {};
    let dynamicNodes = [];
    for (let node of nodes) {
      if(/\{[^}]+\}/.test(node.locUri)) {
        dynamicNodes.push({ node, pattern: new RegExp('^' + _.escapeRegExp(comparableLocUri(node.locUri)).replace(/\\\{[^}]+?\\\}/g, '[^/]+') + '$') });
      } else {
        nodesByComparableLocUri[comparableLocUri(node.locUri)] = node;
      }
    }
    let nodeAt = (locUri)=>{
      let comparable = comparableLocUri(locUri);
      let dynamicMatch = _.find(dynamicNodes, ({pattern})=>{ return pattern.test(comparable); });
      return nodesByComparableLocUri[comparable] || (dynamicMatch ? dynamicMatch.node : undefined);
    };

    let items = [];
    let settingsEnforced = [];
    let unusableCommands = [];
    let deliveryNotes = [];
    let isAtomic = atomic;
    for (let command of commands) {
      let verb = command.verb === undefined ? (command.wifi ? 'Add' : 'Replace') : _.find(['Add', 'Replace', 'Exec'], (knownVerb)=>{ return knownVerb.toLowerCase() === String(command.verb).toLowerCase(); });
      if(!verb) {
        unusableCommands.push(`"${command.verb}" is not a command a profile can carry.`);
        continue;
      }

      if(_.isPlainObject(command.wifi)) {
        let wifi = command.wifi;
        let wlanProfileXml;
        try {
          wlanProfileXml = sails.helpers.buildWlanProfileXml.with({
            ssid: wifi.ssid,
            security: wifi.security,
            passphrase: wifi.passphrase || undefined,
            eap: _.isPlainObject(wifi.eap) ? wifi.eap : undefined,
            hidden: wifi.hidden === true,
            connectAutomatically: wifi.connectAutomatically !== false,
          });
        } catch (err) {
          // A field outside what WLAN_profile accepts fails the helper's input validation, which lists each problem.
          unusableCommands.push(`The Wi-Fi network ${JSON.stringify(wifi.ssid)} could not be written: ${err.problems ? err.problems.join('; ') : err.message}`);
          continue;
        }
        let scope = wifi.scope === 'User' ? 'User' : 'Device';
        // The instance is created by the Add of its WlanXml leaf, so it is always an Add.
        items.push({ verb: 'Add', locUri: `./${scope}/Vendor/MSFT/WiFi/Profile/${encodeURIComponent(wifi.ssid)}/WlanXml`, format: 'chr', data: wlanProfileXml });
        settingsEnforced.push({ name: `WiFi/Profile/${wifi.ssid}`, value: wifi.security });
        // The EAP configuration is written here, so the model never sees that it names no server to trust.
        let validatesTheServer = /<ServerNames>[^<]|<TrustedRootCA/.test(wlanProfileXml);
        if(/Enterprise/.test(wifi.security) && !validatesTheServer) {
          deliveryNotes.push(`Add the RADIUS server names and root CA thumbprint for ${wifi.ssid} before deploying it, or users will be asked to trust the server's certificate when they first connect.`);
        }
        continue;
      }

      if(!_.isString(command.locURI) || !command.locURI.trim()) {
        unusableCommands.push('A command had neither a LocURI nor a Wi-Fi network.');
        continue;
      }
      // Exec runs a one-time command rather than setting a value.  Enrolling a SCEP certificate is the one Exec a
      // profile carries.
      if(verb === 'Exec' && !/\/ClientCertificateInstall\/SCEP\/[^/]+\/Install\/Enroll$/i.test(command.locURI.trim())) {
        unusableCommands.push(`${command.locURI.trim()} runs a one-time command, which a profile cannot do.`);
        continue;
      }
      let node = nodeAt(command.locURI);
      if(node && node.mustBeWrappedInAtomic) {
        isAtomic = true;
      }
      let locUri = command.locURI.trim();
      let hasData = command.data !== undefined && command.data !== null;
      let format;
      if(node) {
        format = node.format;
      } else if(/\/Policy\/Config\/[^/]+~Policy~/i.test(locUri)) {
        // An ingested ADMX template's policy, which the reference cannot list, always takes a policy fragment.
        format = 'chr';
      } else if(_.contains(['b64', 'bin', 'bool', 'chr', 'date', 'float', 'int', 'node', 'null', 'time', 'xml'], String(command.format).toLowerCase())) {
        format = String(command.format).toLowerCase();
      } else if(_.has(formatsByAlias, String(command.format).toLowerCase())) {
        format = formatsByAlias[String(command.format).toLowerCase()];
      } else {
        // A value's JavaScript type does not say which DFFormat the node declares, and a wrong <Format> deploys
        // cleanly and enforces nothing.
        unusableCommands.push(`${locUri} is not in the CSP reference and came with no recognized format, so its <Format> is unknown.`);
        continue;
      }
      let whyTheDataDoesNotFit = whyDataDoesNotFit(command.data, format, verb);
      if(whyTheDataDoesNotFit) {
        unusableCommands.push(`${locUri} ${whyTheDataDoesNotFit}`);
        continue;
      }
      items.push({ verb, locUri, format, data: command.data });
      if(hasData) {
        settingsEnforced.push({ name: locUri.split('/').slice(-2).join('/'), value: format === 'bool' || format === 'int' ? dataFor(command.data, format) : String(command.data) });
      }
    }

    let renderedCommands = _.map(items, (item)=>{
      let lines = [`<${item.verb}>`, '  <Item>'];
      if(item.format !== 'null') {
        lines.push('    <Meta>', `      <Format xmlns="syncml:metinf">${item.format}</Format>`, '    </Meta>');
      }
      lines.push('    <Target>', `      <LocURI>${escapeXml(item.locUri)}</LocURI>`, '    </Target>');
      if(item.data !== undefined && item.data !== null) {
        lines.push(`    <Data>${dataFor(item.data, item.format)}</Data>`);
      }
      return lines.concat('  </Item>', `</${item.verb}>`).join('\n');
    });

    let profile = isAtomic ?
      ['<Atomic>'].concat(_.map(renderedCommands, (renderedCommand)=>{ return renderedCommand.replace(/^/gm, '  '); }), '</Atomic>').join('\n') :
      renderedCommands.join('\n');

    return { profile: profile + '\n', settingsEnforced, deliveryNotes, unusableCommands };

  }


};


/**
 * Why a command's data cannot be written for its node's format, if it cannot.
 *
 * An int node given true or false is refused rather than turned into a number: DeviceLock/DevicePasswordEnabled
 * takes 0 for enabled, so no conversion is safe.
 *
 * @param  {*} data
 * @param  {String} format  The node's DFFormat; `node` for an interior node.
 * @param  {String} verb
 * @returns {String|undefined}  The rest of a sentence that starts with the node's LocURI.
 */
function whyDataDoesNotFit(data, format, verb) {
  let hasData = data !== undefined && data !== null;
  if(format === 'node') {
    return hasData ? 'is an interior node, which holds other nodes rather than a value.' : undefined;
  }
  if(!hasData) {
    return verb === 'Exec' || format === 'null' ? undefined : 'needs a value.';
  }
  if(_.isObject(data)) {
    return `takes ${format} data, not ${_.isArray(data) ? 'a list' : 'an object'}.`;
  }
  if(format === 'int' && !/^-?\d+$/.test(String(data).trim())) {
    return `takes a whole number, not ${JSON.stringify(data)}.`;
  }
  if(format === 'bool' && !/^(true|false|0|1)$/i.test(String(data).trim())) {
    return `takes true or false, not ${JSON.stringify(data)}.`;
  }
  return undefined;
}


/**
 * The <Data> text for a value that fits its node's format.
 *
 * A bool node given 0 or 1 is written false or true, which means the same thing.
 *
 * @param  {*} value
 * @param  {String} format
 * @returns {String}
 */
function dataFor(value, format) {
  if(format === 'bool') {
    return String(/^(true|1)$/i.test(String(value).trim()));
  }
  if(format === 'int') {
    return String(value).trim();
  }
  // A model used to writing SyncML sometimes wraps a fragment in CDATA itself, and wrapping it again would break it.
  let text = String(value).replace(/^\s*<!\[CDATA\[([\s\S]*)\]\]>\s*$/, '$1');
  if(/[<&]/.test(text)) {
    return `<![CDATA[${text.replace(/\]\]>/g, ']]]]><![CDATA[>')}]]>`;
  }
  // ]]> is the one sequence text outside a CDATA section may not contain.
  return text.replace(/\]\]>/g, ']]&gt;');
}


/**
 * @param  {String} text
 * @returns {String}  text, safe inside an XML element.
 */
function escapeXml(text) {
  return String(text).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
