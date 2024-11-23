module.exports = {


  friendlyName: 'Get a device certificate via EST protocol',


  description: 'Take a certificate signing request and authentication token, then use the EST protocol to request a certificate to be issued.',

  extendedDescription: 'This action is the result of a customer hackathon working on issuing device certificates to Linux devices.',

  moreInfoUrl: 'https://github.com/fleetdm/confidential/issues/8785',


  inputs: {
    csrData: {
      required: true,
      type: 'string',
      description: 'Certificate Signing Request (CSR) data'
    },
    authToken: {
      required: true,
      type: 'string',
      description: 'Authorization token provided by IdP',
    }
  },


  exits: {

    success: {
      description: 'Successfully generated certificate',
      extendedDescription: 'This action is the result of a hackathon where we were relying on this datashape.',
      outputType: {certificate:'string'},
      outputFriendlyName: 'Certificate',
    },

    invalidToken: {
      description: 'The IdP auth token is invalid',
      statusCode: 403,
    },

    invalidCsr: {
      description: 'The provided CSR data was invalid.',
      responseType: 'badRequest'
    }

  },

  fn: async function ({ csrData, authToken }) {
    const INTROSPECT_ENDPOINT = sails.config.custom.certIssueIdpIntrospectEndpoint;
    if (!INTROSPECT_ENDPOINT) {
      throw new Error('sails.config.custom.certIssueIdpIntrospectEndpoint is required');
    }
    const IDP_CLIENT_ID = sails.config.custom.certIssueIdpClientId;
    if (!IDP_CLIENT_ID) {
      throw new Error('sails.config.custom.certIssueIdpClientId is required');
    }
    const EST_ENDPOINT = sails.config.custom.certIssueEstEndpoint;
    if (!EST_ENDPOINT) {
      throw new Error('sails.config.custom.certIssueEstEndpoint is required');
    }
    const EST_CLIENT_ID = sails.config.custom.certIssueEstClientId;
    if (!EST_CLIENT_ID) {
      throw new Error('sails.config.custom.certIssueEstClientId is required');
    }
    const EST_CLIENT_KEY = sails.config.custom.certIssueEstClientKey;
    if (!EST_CLIENT_KEY) {
      throw new Error('sails.config.custom.certIssueEstClientKey is required');
    }

    // Ask the IdP to introspect the auth token (ensure it's valid and extract the values).
    const introspectResponse = await sails.helpers.http.sendHttpRequest.with({
      url: INTROSPECT_ENDPOINT,
      method: 'POST',
      enctype: 'application/x-www-form-urlencoded',
      body: {
        'client_id': IDP_CLIENT_ID,
        'token': authToken,
      },
    });

    if (!introspectResponse.body) {
      throw 'invalidToken';
    }

    const introspectBody = JSON.parse(introspectResponse.body);
    if (!introspectBody.active) {
      throw 'invalidToken';
    }
    const introspectUsername = introspectBody.username;

    // Extract the email and username from the CSR. Ensure they match.
    let jsrsasign = require('jsrsasign');
    const csrUtil = jsrsasign.asn1.csr.CSRUtil;
    const csrObj = csrUtil.getParam(csrData);
    let csrEmail = '';
    let csrUsername = '';
    for (const extension of csrObj.extreq) {
      if (extension.extname === 'subjectAltName') {
        for (const extentry of extension.array) {
          if ('rfc822' in extentry) {
            csrEmail = extentry.rfc822;
          }
          if ('other' in extentry) {
            if ('oid' in extentry.other && extentry.other.oid === '1.3.6.1.4.1.311.20.2.3') {
              csrUsername = extentry.other.value.utf8str.str;
            }
          }
        }
      }
    }
    if (csrEmail === '') {
      throw 'invalidCsr';
    }
    if (!csrEmail.startsWith(csrUsername)) {
      throw 'invalidCsr';
    }

    // Ensure username from IdP auth matches username in CSR. If they don't match, perhaps the user
    // is trying to get a certificate with another user's name?
    if (csrEmail !== introspectUsername) {
      throw 'invalidToken';
    }

    // Ask the PKI provider for a certificate
    const request = require('@sailshq/request');
    const estResponse = await new Promise((resolve, reject) => {
      request({
        url: EST_ENDPOINT,
        method: 'POST',
        body: csrData.replace(/(-----(BEGIN|END) CERTIFICATE REQUEST-----|\n)/g, ''),
        headers: {
          'Content-Type': 'application/pkcs10',
          'Authorization': `Basic ${Buffer.from(`${EST_CLIENT_ID}:${EST_CLIENT_KEY}`).toString('base64')}`,
        },
      }, (err, response)=>{
        if (err) {
          reject(err);
        } else {
          resolve(response);
        }
      });
    });

    return estResponse.body;
  }

};
