module.exports = {


  friendlyName: 'Build WLAN profile XML',


  description: 'Write the WLAN_profile document a Windows WiFi CSP WlanXml node takes, from a network\'s SSID, security, and EAP method.',


  extendedDescription: `WLAN_profile enforces element order and closed value lists, and an enterprise network's EAP
configuration is most of the document, so it is written here rather than by the model.  PEAP and TLS follow Microsoft's
published samples (https://learn.microsoft.com/en-us/windows/win32/nativewifi/wireless-profile-samples), whose
namespaces read https:// where Windows only accepts the http:// ones the schemas declare.  Microsoft publishes no TTLS
sample, so TTLS follows its EapTtlsConnectionPropertiesV1 schema, except for TrustedRootCAHash, which is written the way
GEANT's eduroam configuration tool writes it rather than as the schema's TrustedRootCAHashes.`,


  sync: true,


  inputs: {

    ssid: {
      type: 'string',
      required: true,
    },

    security: {
      type: 'string',
      required: true,
      isIn: ['Open', 'WPA2Personal', 'WPA3Personal', 'WPA3PersonalTransition', 'WPA2Enterprise', 'WPA3Enterprise', 'WPA3Enterprise192'],
    },

    passphrase: {
      type: 'string',
      description: 'A personal network\'s password.  Without one, the user is prompted for it.',
    },

    eap: {
      type: {},
      description: 'An enterprise network\'s EAP configuration: {method, serverNames?, trustedRootCAThumbprints?, authMode?}.',
      extendedDescription: 'serverNames and trustedRootCAThumbprints are each a list, or a string holding one entry.',
      example: { method: 'PEAP', serverNames: ['radius.example.com'], trustedRootCAThumbprints: ['3b1efd3a66ea28b16697394703a72ca340a05bd5'] },
    },

    hidden: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Whether the network does not broadcast its SSID.',
    },

    connectAutomatically: {
      type: 'boolean',
      defaultsTo: true,
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'WLAN profile XML',
      outputDescription: 'The WLANProfile document, on one line.',
    },

  },


  fn: function ({ssid, security, passphrase, eap, hidden, connectAutomatically}) {

    const AUTH_ENCRYPTION_BY_SECURITY = {
      Open: { authentication: 'open', encryption: 'none', useOneX: false },
      WPA2Personal: { authentication: 'WPA2PSK', encryption: 'AES', useOneX: false },
      WPA3Personal: { authentication: 'WPA3SAE', encryption: 'AES', useOneX: false },
      WPA3PersonalTransition: { authentication: 'WPA3SAE', encryption: 'AES', useOneX: false, transitionMode: true },
      WPA2Enterprise: { authentication: 'WPA2', encryption: 'AES', useOneX: true },
      WPA3Enterprise: { authentication: 'WPA3ENT', encryption: 'AES', useOneX: true },
      WPA3Enterprise192: { authentication: 'WPA3ENT192', encryption: 'GCMP256', useOneX: true },
    };

    const EAP_CONFIG_BUILDERS_BY_METHOD = {

      // Server certificate validation stays on for every method.  Without server names or a root CA to trust, the
      // user is asked to confirm the server instead, rather than validation being switched off.

      // PEAP with EAP-MSCHAPv2 inside, from the WPA2-Enterprise with PEAP-MSCHAPv2 sample.
      PEAP: (serverValidation)=>{
        return eapHostConfig(25, 0, '<Eap xmlns="http://www.microsoft.com/provisioning/BaseEapConnectionPropertiesV1"><Type>25</Type>' +
          '<EapType xmlns="http://www.microsoft.com/provisioning/MsPeapConnectionPropertiesV1">' +
          `<ServerValidation><DisableUserPromptForServerValidation>${hasServerValidation(serverValidation)}</DisableUserPromptForServerValidation><ServerNames>${escapeXml(serverValidation.serverNames)}</ServerNames>${trustedRootCaElements(serverValidation, 'TrustedRootCA')}</ServerValidation>` +
          '<FastReconnect>true</FastReconnect><InnerEapOptional>false</InnerEapOptional>' +
          mschapv2Eap() +
          '<EnableQuarantineChecks>false</EnableQuarantineChecks><RequireCryptoBinding>false</RequireCryptoBinding>' +
          '<PeapExtensions><PerformServerValidation xmlns="http://www.microsoft.com/provisioning/MsPeapConnectionPropertiesV2">true</PerformServerValidation>' +
          `<AcceptServerName xmlns="http://www.microsoft.com/provisioning/MsPeapConnectionPropertiesV2">${serverValidation.serverNames !== ''}</AcceptServerName></PeapExtensions>` +
          '</EapType></Eap>');
      },

      // EAP-TLS with a certificate from the device's store, from the EAP configuration sample.
      TLS: (serverValidation)=>{
        return eapHostConfig(13, 0, '<Eap xmlns="http://www.microsoft.com/provisioning/BaseEapConnectionPropertiesV1"><Type>13</Type>' +
          '<EapType xmlns="http://www.microsoft.com/provisioning/EapTlsConnectionPropertiesV1">' +
          '<CredentialsSource><CertificateStore><SimpleCertSelection>true</SimpleCertSelection></CertificateStore></CredentialsSource>' +
          `<ServerValidation><DisableUserPromptForServerValidation>${hasServerValidation(serverValidation)}</DisableUserPromptForServerValidation><ServerNames>${escapeXml(serverValidation.serverNames)}</ServerNames>${trustedRootCaElements(serverValidation, 'TrustedRootCA')}</ServerValidation>` +
          '<DifferentUsername>false</DifferentUsername>' +
          '<PerformServerValidation xmlns="http://www.microsoft.com/provisioning/EapTlsConnectionPropertiesV2">true</PerformServerValidation>' +
          `<AcceptServerName xmlns="http://www.microsoft.com/provisioning/EapTlsConnectionPropertiesV2">${serverValidation.serverNames !== ''}</AcceptServerName>` +
          '</EapType></Eap>');
      },

      // EAP-TTLS with MSCHAPv2 inside.  Microsoft's TTLS method is AuthorId 311, unlike its PEAP and TLS methods.
      TTLS: (serverValidation)=>{
        return eapHostConfig(21, 311, '<EapTtls xmlns="http://www.microsoft.com/provisioning/EapTtlsConnectionPropertiesV1">' +
          `<ServerValidation><ServerNames>${escapeXml(serverValidation.serverNames)}</ServerNames>${trustedRootCaElements(serverValidation, 'TrustedRootCAHash')}<DisablePrompt>${hasServerValidation(serverValidation)}</DisablePrompt></ServerValidation>` +
          '<Phase2Authentication><MSCHAPv2Authentication><UseWinlogonCredentials>false</UseWinlogonCredentials></MSCHAPv2Authentication></Phase2Authentication>' +
          '<Phase1Identity><IdentityPrivacy>false</IdentityPrivacy></Phase1Identity>' +
          '</EapTtls>');
      },

    };

    let authEncryption = AUTH_ENCRYPTION_BY_SECURITY[security];
    let isEnterprise = authEncryption.useOneX;
    if(security === 'Open' && passphrase) {
      throw new Error('An open network has no password.  A network with one is WPA2 or WPA3 Personal.');
    }

    // The method's usual spellings -- EAP-TLS, PEAP-MSCHAPv2 -- name the same three methods.
    let eapMethod = eap && eap.method ? String(eap.method).toUpperCase().replace(/^EAP-/, '').replace(/[-/ ]?MSCHAPV2$/, '') : undefined;
    if(isEnterprise && !EAP_CONFIG_BUILDERS_BY_METHOD[eapMethod]) {
      throw new Error(`An enterprise network needs an EAP method of ${Object.keys(EAP_CONFIG_BUILDERS_BY_METHOD).join(', ')}, not ${eapMethod || 'none'}.`);
    }
    if(security === 'WPA3Enterprise192' && eapMethod !== 'TLS') {
      throw new Error('WPA3-Enterprise 192-bit mode only supports EAP-TLS.');
    }
    let serverValidation = isEnterprise ? serverValidationFrom(eap) : undefined;

    let securityXml = '<authEncryption>' +
      `<authentication>${authEncryption.authentication}</authentication>` +
      `<encryption>${authEncryption.encryption}</encryption>` +
      `<useOneX>${isEnterprise}</useOneX>` +
      (authEncryption.transitionMode ? '<transitionMode xmlns="http://www.microsoft.com/networking/WLAN/profile/v4">true</transitionMode>' : '') +
      '</authEncryption>';
    if(!isEnterprise && passphrase) {
      securityXml += `<sharedKey><keyType>passPhrase</keyType><protected>false</protected><keyMaterial>${escapeXml(passphrase)}</keyMaterial></sharedKey>`;
    }
    if(isEnterprise) {
      securityXml += '<OneX xmlns="http://www.microsoft.com/networking/OneX/v1">' +
        `<authMode>${_.contains(['machineOrUser', 'machine', 'user', 'guest'], eap.authMode) ? eap.authMode : 'machineOrUser'}</authMode>` +
        `<EAPConfig>${EAP_CONFIG_BUILDERS_BY_METHOD[eapMethod](serverValidation)}</EAPConfig>` +
        '</OneX>';
    }

    return '<?xml version="1.0"?>' +
      '<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">' +
      `<name>${escapeXml(ssid)}</name>` +
      `<SSIDConfig><SSID><hex>${Buffer.from(ssid, 'utf8').toString('hex').toUpperCase()}</hex><name>${escapeXml(ssid)}</name></SSID>${hidden ? '<nonBroadcast>true</nonBroadcast>' : ''}</SSIDConfig>` +
      '<connectionType>ESS</connectionType>' +
      `<connectionMode>${connectAutomatically ? 'auto' : 'manual'}</connectionMode>` +
      `<MSM><security>${securityXml}</security></MSM>` +
      '</WLANProfile>';

  }


};


/**
 * @param  {Number} type  The EAP type, e.g. 25 for PEAP.
 * @param  {Number} authorId
 * @param  {String} configXml  What goes inside <Config>.
 * @returns {String}
 */
function eapHostConfig(type, authorId, configXml) {
  let eapCommon = (name, value)=>{ return `<${name} xmlns="http://www.microsoft.com/provisioning/EapCommon">${value}</${name}>`; };
  return '<EapHostConfig xmlns="http://www.microsoft.com/provisioning/EapHostConfig">' +
    `<EapMethod>${eapCommon('Type', type)}${eapCommon('VendorId', 0)}${eapCommon('VendorType', 0)}${eapCommon('AuthorId', authorId)}</EapMethod>` +
    `<Config xmlns="http://www.microsoft.com/provisioning/EapHostConfig">${configXml}</Config>` +
    '</EapHostConfig>';
}


/**
 * The RADIUS servers and root CAs an enterprise network trusts, in the form the EAP configurations take them.
 *
 * Windows separates server names with semicolons, and a thumbprint is the 40 hex digits of a certificate's SHA-1
 * hash, which admins often paste with a "SHA1" label or separators.
 *
 * @param  {Dictionary} eap  {serverNames?, trustedRootCAThumbprints?}, each a list or a string holding one entry.
 * @returns {Dictionary}  {serverNames: "a;b", thumbprints: ["40 hex digits"]}
 * @throws {Error}  A thumbprint that is not 40 hex digits.
 */
function serverValidationFrom(eap) {
  let asList = (value)=>{ return _.filter(_.isArray(value) ? value : [value], (entry)=>{ return _.isString(entry) && entry.trim() !== ''; }); };
  let serverNames = _.compact(asList(eap.serverNames).join(' ').split(/[\s,;]+/)).join(';');
  let thumbprints = _.map(asList(eap.trustedRootCAThumbprints), (thumbprint)=>{
    let hexDigits = thumbprint.trim().replace(/^sha-?1\s*[:=]?\s*/i, '').replace(/[\s:-]/g, '').toLowerCase();
    if(!/^[0-9a-f]{40}$/.test(hexDigits)) {
      throw new Error(`${JSON.stringify(thumbprint)} is not a root CA thumbprint, which is the 40 hex digits of the certificate's SHA-1 hash.`);
    }
    return hexDigits;
  });
  return { serverNames, thumbprints };
}


/**
 * @returns {String}  EAP-MSCHAPv2, prompting for credentials rather than reusing the Windows sign-in.
 */
function mschapv2Eap() {
  return '<Eap xmlns="http://www.microsoft.com/provisioning/BaseEapConnectionPropertiesV1"><Type>26</Type>' +
    '<EapType xmlns="http://www.microsoft.com/provisioning/MsChapV2ConnectionPropertiesV1"><UseWinLogonCredentials>false</UseWinLogonCredentials></EapType>' +
    '</Eap>';
}


/**
 * @param  {Dictionary} serverValidation  From serverValidationFrom.
 * @returns {Boolean}  Whether the network says which RADIUS servers or root CAs to trust, so the user need not confirm the server.
 */
function hasServerValidation(serverValidation) {
  return serverValidation.serverNames !== '' || serverValidation.thumbprints.length > 0;
}


/**
 * One element per trusted root CA, its thumbprint written as Windows exports it: hex bytes separated by spaces.
 *
 * @param  {Dictionary} serverValidation  From serverValidationFrom.
 * @param  {String} elementName  TrustedRootCA for PEAP and TLS, TrustedRootCAHash for TTLS.
 * @returns {String}
 */
function trustedRootCaElements(serverValidation, elementName) {
  return _.map(serverValidation.thumbprints, (thumbprint)=>{
    return `<${elementName}>${thumbprint.match(/../g).join(' ')}</${elementName}>`;
  }).join('');
}


/**
 * @param  {String} text
 * @returns {String}  text, safe inside an XML element.
 */
function escapeXml(text) {
  return String(text).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
