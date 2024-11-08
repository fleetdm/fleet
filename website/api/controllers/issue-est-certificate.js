module.exports = {


  friendlyName: 'Issue certificate via EST',


  description: 'Take a certificate signing request and authentication token, then use the EST protocol to request a certificate to be issued.',

  extendedDescription: '',


  inputs: {
    csrData: {
      required: true,
      type: 'string',
      description: 'CSR'
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
      outputType: {certificate:'string'},
      outputFriendlyName: 'Certificate',
    },

    invalidToken: {
      description: 'The IdP auth token is invalid',
      statusCode: 403,
    },

    badRequest: {
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
    // TODO can this be done with sails.helpers.http? Couldn't figure out how to send form data.
    const urlencoded = new URLSearchParams();
    urlencoded.append('client_id', IDP_CLIENT_ID);
    urlencoded.append('token', authToken);
    const introspectResponse = await sails.helpers.http.sendHttpRequest.with({
      url: INTROSPECT_ENDPOINT,
      method: 'POST',
      enctype: 'application/x-www-form-urlencoded',
      data: {
        client_id: IDP_CLIENT_ID,
        token: authToken
      },
    });

    if (!introspectResponse.data.active) {
      throw 'invalidToken';
    }

    const introspectUsername = introspectResponse.data.username;

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
      throw 'badRequest';
    }
    if (!csrEmail.startsWith(csrUsername)) {
      throw 'badRequest';
    }

    // Ensure username from IdP auth matches username in CSR. If they don't match, perhaps the user
    // is trying to get a certificate with another user's name?
    if (csrEmail !== introspectUsername) {
      throw 'invalidToken';
    }

    // Ask the PKI provider for a certificate
    const estResponse = await axios({
      url: EST_ENDPOINT,
      method: 'POST',
      data: csrData.replace(/(-----(BEGIN|END) CERTIFICATE REQUEST-----|\n)/g, ''),
      headers: {
        'Content-Type': 'application/pkcs10',
		    'Authorization': `Basic ${Buffer.from(`${EST_CLIENT_ID}:${EST_CLIENT_KEY}`).toString('base64')}`,
      },
    });

    return estResponse.data;
  }

};
