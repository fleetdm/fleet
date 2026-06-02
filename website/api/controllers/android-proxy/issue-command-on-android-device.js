module.exports = {


  friendlyName: 'Issue command on android device',


  description: 'Issues an AMAPI command (e.g. LOCK, RESET_PASSWORD, WIPE) to a device of an Android enterprise.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    deviceId: {
      type: 'string',
      required: true,
    },
    // Captured from the `:deviceId::issueCommand` route. Validated here so requests targeting any
    // other colon-suffixed action (e.g. `/devices/<id>:somethingElse`) are rejected up front rather
    // than silently invoking issueCommand. Mirrors how `modify-enterprise-app-policy.js` captures
    // `googleAction`.
    issueCommand: {
      type: 'string',
      required: true,
      isIn: ['issueCommand'],
    },
    // AMAPI Command fields. Inputs are declared explicitly (rather than forwarding req.body) so the
    // proxy's accepted surface is visible. `type` is not constrained via isIn so the Fleet server can
    // issue any AMAPI command type without a proxy change. Adding entirely new Command FIELDS (e.g. a
    // future *Params sibling Google adds to AMAPI) does still require updating this list.
    type: {
      type: 'string',
      required: true,
      description: 'The AMAPI command type (e.g. LOCK, RESET_PASSWORD, REBOOT, RELINQUISH_OWNERSHIP, CLEAR_APP_DATA, START_LOST_MODE, STOP_LOST_MODE, ADD_ESIM, REMOVE_ESIM, REQUEST_DEVICE_INFO, WIPE).',
    },
    duration: {
      type: 'string',
      description: 'How long the command remains valid (e.g. "315360000s"). Forwarded to AMAPI verbatim.',
    },
    newPassword: {
      type: 'string',
      description: 'New device password for RESET_PASSWORD. Fleet sends an empty string to clear the passcode, so an explicitly empty value is preserved.',
    },
    resetPasswordFlags: {
      type: ['string'],
      description: 'AMAPI reset-password flags (REQUIRE_ENTRY, DO_NOT_ASK_CREDENTIALS_ON_BOOT, LOCK_NOW).',
    },
    wipeParams: {
      type: 'ref',
      description: 'AMAPI WipeParams object (may be empty {}). Required by AMAPI for WIPE commands.',
    },
    addEsimParams: {
      type: 'ref',
      description: 'AMAPI AddEsimParams object, for ADD_ESIM commands.',
    },
    removeEsimParams: {
      type: 'ref',
      description: 'AMAPI RemoveEsimParams object, for REMOVE_ESIM commands.',
    },
    clearAppsDataParams: {
      type: 'ref',
      description: 'AMAPI ClearAppsDataParams object, for CLEAR_APP_DATA commands.',
    },
    startLostModeParams: {
      type: 'ref',
      description: 'AMAPI StartLostModeParams object, for START_LOST_MODE commands.',
    },
    stopLostModeParams: {
      type: 'ref',
      description: 'AMAPI StopLostModeParams object, for STOP_LOST_MODE commands.',
    },
    requestDeviceInfoParams: {
      type: 'ref',
      description: 'AMAPI RequestDeviceInfoParams object, for REQUEST_DEVICE_INFO commands.',
    },
  },


  exits: {
    success: { description: 'The command was successfully issued to the Android device. The AMAPI Operation is returned.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound' },
    deviceNoLongerManaged: { description: 'The specified device is no longer managed by the Android enterprise.', responseType: 'notFound' },
  },


  fn: async function ({
    androidEnterpriseId, deviceId, type, duration, newPassword, resetPasswordFlags,
    wipeParams, addEsimParams, removeEsimParams, clearAppsDataParams,
    startLostModeParams, stopLostModeParams, requestDeviceInfoParams,
  }) {

    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      throw 'missingAuthHeader';
    }

    // Authenticate this request
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      androidEnterpriseId: androidEnterpriseId
    });

    // Return a 404 response if no records are found.
    if (!thisAndroidEnterprise) {
      throw 'notFound';
    }
    // Return an unauthorized response if the provided secret does not match.
    if (thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      throw 'unauthorized';
    }

    // Check the list of Android Enterprises managed by Fleet to see if this Android Enterprise is still managed.
    let isEnterpriseManagedByFleet = await sails.helpers.androidProxy.getIsEnterpriseManagedByFleet(androidEnterpriseId);
    // Return a 404 response if this Android enterprise is no longer managed by Fleet.
    if (!isEnterpriseManagedByFleet) {
      throw 'notFound';
    }

    // Build the AMAPI Command body from declared inputs (not req.body) so the proxy's accepted surface
    // stays explicit. Use `!== undefined` rather than truthy checks because Fleet relies on forwarding
    // an empty newPassword ("") to clear the device passcode and an empty wipeParams ({}) for WIPE.
    let commandBody = { type: type };
    if (duration !== undefined) {
      commandBody.duration = duration;
    }
    if (newPassword !== undefined) {
      commandBody.newPassword = newPassword;
    }
    if (resetPasswordFlags !== undefined) {
      commandBody.resetPasswordFlags = resetPasswordFlags;
    }
    if (wipeParams !== undefined) {
      commandBody.wipeParams = wipeParams;
    }
    if (addEsimParams !== undefined) {
      commandBody.addEsimParams = addEsimParams;
    }
    if (removeEsimParams !== undefined) {
      commandBody.removeEsimParams = removeEsimParams;
    }
    if (clearAppsDataParams !== undefined) {
      commandBody.clearAppsDataParams = clearAppsDataParams;
    }
    if (startLostModeParams !== undefined) {
      commandBody.startLostModeParams = startLostModeParams;
    }
    if (stopLostModeParams !== undefined) {
      commandBody.stopLostModeParams = stopLostModeParams;
    }
    if (requestDeviceInfoParams !== undefined) {
      commandBody.requestDeviceInfoParams = requestDeviceInfoParams;
    }

    // Issue the command to the device for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occur using google's node library.
    let issueCommandResponse = await sails.helpers.flow.build(async () => {
      let { google } = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');
      let googleAuth = new google.auth.GoogleAuth({
        scopes: ['https://www.googleapis.com/auth/androidmanagement'],
        credentials: {
          client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
          private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({ auth: authClient });
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Devices.html#issueCommand
      let response = await androidmanagement.enterprises.devices.issueCommand({
        name: `enterprises/${androidEnterpriseId}/devices/${deviceId}`,
        requestBody: commandBody,
      });
      return response.data;
    }).intercept({status: 429}, (err)=>{
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to issue a command to a device for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept((err)=>{
      let errorString = err.toString();
      if (errorString.includes('Device is no longer being managed')) {
        return {'deviceNoLongerManaged': 'The device is no longer managed by the Android enterprise.'};
      }
      return new Error(`When attempting to issue a command to a device for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    // Return the AMAPI Operation back to the Fleet server.
    return issueCommandResponse;

  }


};
