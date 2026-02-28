module.exports = {


  friendlyName: 'Modify android policies',


  description: 'Modifies a policy of an Android enterprise',


  inputs: {
    // Provided via path parameter
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    policyId: {
      type: 'string',
      required: true,
    },

    // Sent via query parameter
    updateMask: {
      type: 'string',
    },

    // Sent via request body
    // [?]: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#resource:-policy
    name: { type: 'string'},
    version: { type: 'string'},
    applications: { type: [{}]},
    maximumTimeToLock: { type: 'string'},
    screenCaptureDisabled: { type: 'boolean'},
    cameraDisabled: { type: 'boolean'},
    keyguardDisabledFeatures: { type: ['string'] },
    defaultPermissionPolicy: { type: 'string' },
    persistentPreferredActivities: { type: [{}] },
    openNetworkConfiguration: { type: {} },
    systemUpdate: { type: {} },
    accountTypesWithManagementDisabled: { type: ['string'] },
    addUserDisabled: { type: 'boolean'},
    adjustVolumeDisabled: { type: 'boolean'},
    factoryResetDisabled: { type: 'boolean'},
    installAppsDisabled: { type: 'boolean'},
    mountPhysicalMediaDisabled: { type: 'boolean'},
    modifyAccountsDisabled: { type: 'boolean'},
    safeBootDisabled: { type: 'boolean'},
    uninstallAppsDisabled: { type: 'boolean'},
    statusBarDisabled: { type: 'boolean'},
    keyguardDisabled: { type: 'boolean'},
    minimumApiLevel: { type: 'number' },
    statusReportingSettings: { type: {} },
    bluetoothContactSharingDisabled: { type: 'boolean'},
    shortSupportMessage: { type: {} },
    longSupportMessage: { type: {} },
    passwordRequirements:  { type: {} },
    wifiConfigsLockdownEnabled: { type: 'boolean'},
    bluetoothConfigDisabled: { type: 'boolean'},
    cellBroadcastsConfigDisabled: { type: 'boolean'},
    credentialsConfigDisabled: { type: 'boolean'},
    mobileNetworksConfigDisabled: { type: 'boolean'},
    tetheringConfigDisabled: { type: 'boolean'},
    vpnConfigDisabled: { type: 'boolean'},
    wifiConfigDisabled: { type: 'boolean'},
    createWindowsDisabled: { type: 'boolean'},
    networkResetDisabled: { type: 'boolean'},
    outgoingBeamDisabled: { type: 'boolean'},
    outgoingCallsDisabled: { type: 'boolean'},
    removeUserDisabled: { type: 'boolean'},
    shareLocationDisabled: { type: 'boolean'},
    smsDisabled: { type: 'boolean'},
    unmuteMicrophoneDisabled: { type: 'boolean'},
    usbFileTransferDisabled: { type: 'boolean'},
    ensureVerifyAppsEnabled: { type: 'boolean'},
    permittedInputMethods: { type: {} },
    stayOnPluggedModes: { type: ['string']},
    recommendedGlobalProxy:  { type: {} },
    setUserIconDisabled: { type: 'boolean'},
    setWallpaperDisabled: { type: 'boolean'},
    choosePrivateKeyRules:  { type: [{}] },
    alwaysOnVpnPackage:  { type: {} },
    frpAdminEmails:  { type: ['string'] },
    deviceOwnerLockScreenInfo:  { type: {} },
    dataRoamingDisabled: { type: 'boolean'},
    locationMode: { type: 'string' },
    networkEscapeHatchEnabled: { type: 'boolean'},
    bluetoothDisabled: { type: 'boolean'},
    complianceRules:  { type: [{}] },
    blockApplicationsEnabled: { type: 'boolean'},
    installUnknownSourcesAllowed: { type: 'boolean'},
    debuggingFeaturesAllowed: { type: 'boolean'},
    funDisabled: { type: 'boolean'},
    autoTimeRequired: { type: 'boolean'},
    permittedAccessibilityServices:  { type: {} },
    appAutoUpdatePolicy: { type: 'string'},
    kioskCustomLauncherEnabled: { type: 'boolean'},
    androidDevicePolicyTracks: { type: ['string'] },
    skipFirstUseHintsEnabled: { type: 'boolean'},
    privateKeySelectionEnabled: { type: 'boolean'},
    encryptionPolicy: { type: 'string' },
    usbMassStorageEnabled: { type: 'boolean'},
    permissionGrants:  { type: [{}] },
    playStoreMode: { type: 'string' },
    setupActions:  { type: [{}] },
    passwordPolicies: { type: [{}] },
    policyEnforcementRules: { type: [{}] },
    kioskCustomization: { type: {} },
    advancedSecurityOverrides: { type: {} },
    personalUsagePolicies: { type: {} },
    autoDateAndTimeZone: { type: 'string' },
    oncCertificateProviders: { type: [{}] },
    crossProfilePolicies: { type: {} },
    preferentialNetworkService: { type: 'string' },
    usageLog: { type: {} },
    cameraAccess: { type: 'string' },
    microphoneAccess: { type: 'string' },
    deviceConnectivityManagement: { type: {} },
    deviceRadioState: { type: {} },
    credentialProviderPolicyDefault: { type: 'string' },
    printingPolicy: { type: 'string' },
    displaySettings: { type: {} },
    assistContentPolicy: { type: 'string' },
    workAccountSetupConfig: { type: {} },
    wipeDataFlags: { type: ['string'] },
    enterpriseDisplayNameVisibility: { type: 'string' },
    appFunctions: { type: 'string' },
    defaultApplicationSettings:{ type: [{}] },
  },


  exits: {
    success: { description: 'The policy of an Android enterprise was successfully updated.' },
    missingAuthHeader: { description: 'This request was missing an authorization header.', responseType: 'unauthorized'},
    unauthorized: { description: 'Invalid authentication token.', responseType: 'unauthorized'},
    notFound: { description: 'No Android enterprise found for this Fleet server.', responseType: 'notFound'},
    invalidPolicy: { description: 'Invalid patch policy request', responseType: 'badRequest' },
    policyNotFound: { description: 'The specified policy was not found on this Android enterprise', responseType: 'notFound' },
  },


  fn: async function ({
    androidEnterpriseId, policyId, updateMask, name, version, applications, maximumTimeToLock, screenCaptureDisabled,
    cameraDisabled, keyguardDisabledFeatures, defaultPermissionPolicy, persistentPreferredActivities, openNetworkConfiguration,
    systemUpdate, accountTypesWithManagementDisabled, addUserDisabled, adjustVolumeDisabled, factoryResetDisabled,
    installAppsDisabled, mountPhysicalMediaDisabled, modifyAccountsDisabled, safeBootDisabled, uninstallAppsDisabled,
    statusBarDisabled, keyguardDisabled, minimumApiLevel, statusReportingSettings, bluetoothContactSharingDisabled,
    shortSupportMessage, longSupportMessage, passwordRequirements, wifiConfigsLockdownEnabled, bluetoothConfigDisabled,
    cellBroadcastsConfigDisabled, credentialsConfigDisabled, mobileNetworksConfigDisabled, tetheringConfigDisabled,
    vpnConfigDisabled, wifiConfigDisabled, createWindowsDisabled, networkResetDisabled, outgoingBeamDisabled,
    outgoingCallsDisabled, removeUserDisabled, shareLocationDisabled, smsDisabled, unmuteMicrophoneDisabled,
    usbFileTransferDisabled, ensureVerifyAppsEnabled, permittedInputMethods, stayOnPluggedModes, recommendedGlobalProxy,
    setUserIconDisabled, setWallpaperDisabled, choosePrivateKeyRules, alwaysOnVpnPackage, frpAdminEmails,
    deviceOwnerLockScreenInfo, dataRoamingDisabled, locationMode, networkEscapeHatchEnabled, bluetoothDisabled,
    complianceRules, blockApplicationsEnabled, installUnknownSourcesAllowed, debuggingFeaturesAllowed, funDisabled,
    autoTimeRequired, permittedAccessibilityServices, appAutoUpdatePolicy, kioskCustomLauncherEnabled, androidDevicePolicyTracks,
    skipFirstUseHintsEnabled, privateKeySelectionEnabled, encryptionPolicy, usbMassStorageEnabled, permissionGrants,
    playStoreMode, setupActions, passwordPolicies, policyEnforcementRules, kioskCustomization, advancedSecurityOverrides,
    personalUsagePolicies, autoDateAndTimeZone, oncCertificateProviders, crossProfilePolicies, preferentialNetworkService,
    usageLog, cameraAccess, microphoneAccess, deviceConnectivityManagement, deviceRadioState, credentialProviderPolicyDefault,
    printingPolicy, displaySettings, assistContentPolicy, workAccountSetupConfig, wipeDataFlags, enterpriseDisplayNameVisibility,
    appFunctions, defaultApplicationSettings
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
    if(!isEnterpriseManagedByFleet) {
      throw 'notFound';
    }

    // Update the policy for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let modifyPoliciesResponse = await sails.helpers.flow.build(async () => {
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
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Policies.html#patch
      let patchPoliciesResponse = await androidmanagement.enterprises.policies.patch({
        name: `enterprises/${androidEnterpriseId}/policies/${policyId}`,
        requestBody: {
          name,
          version,
          applications,
          maximumTimeToLock,
          screenCaptureDisabled,
          cameraDisabled,
          keyguardDisabledFeatures,
          defaultPermissionPolicy,
          persistentPreferredActivities,
          openNetworkConfiguration,
          systemUpdate,
          accountTypesWithManagementDisabled,
          addUserDisabled,
          adjustVolumeDisabled,
          factoryResetDisabled,
          installAppsDisabled,
          mountPhysicalMediaDisabled,
          modifyAccountsDisabled,
          safeBootDisabled,
          uninstallAppsDisabled,
          statusBarDisabled,
          keyguardDisabled,
          minimumApiLevel,
          statusReportingSettings,
          bluetoothContactSharingDisabled,
          shortSupportMessage,
          longSupportMessage,
          passwordRequirements,
          wifiConfigsLockdownEnabled,
          bluetoothConfigDisabled,
          cellBroadcastsConfigDisabled,
          credentialsConfigDisabled,
          mobileNetworksConfigDisabled,
          tetheringConfigDisabled,
          vpnConfigDisabled,
          wifiConfigDisabled,
          createWindowsDisabled,
          networkResetDisabled,
          outgoingBeamDisabled,
          outgoingCallsDisabled,
          removeUserDisabled,
          shareLocationDisabled,
          smsDisabled,
          unmuteMicrophoneDisabled,
          usbFileTransferDisabled,
          ensureVerifyAppsEnabled,
          permittedInputMethods,
          stayOnPluggedModes,
          recommendedGlobalProxy,
          setUserIconDisabled,
          setWallpaperDisabled,
          choosePrivateKeyRules,
          alwaysOnVpnPackage,
          frpAdminEmails,
          deviceOwnerLockScreenInfo,
          dataRoamingDisabled,
          locationMode,
          networkEscapeHatchEnabled,
          bluetoothDisabled,
          complianceRules,
          blockApplicationsEnabled,
          installUnknownSourcesAllowed,
          debuggingFeaturesAllowed,
          funDisabled,
          autoTimeRequired,
          permittedAccessibilityServices,
          appAutoUpdatePolicy,
          kioskCustomLauncherEnabled,
          androidDevicePolicyTracks,
          skipFirstUseHintsEnabled,
          privateKeySelectionEnabled,
          encryptionPolicy,
          usbMassStorageEnabled,
          permissionGrants,
          playStoreMode,
          setupActions,
          passwordPolicies,
          policyEnforcementRules,
          kioskCustomization,
          advancedSecurityOverrides,
          personalUsagePolicies,
          autoDateAndTimeZone,
          oncCertificateProviders,
          crossProfilePolicies,
          preferentialNetworkService,
          usageLog,
          cameraAccess,
          microphoneAccess,
          deviceConnectivityManagement,
          deviceRadioState,
          credentialProviderPolicyDefault,
          printingPolicy,
          displaySettings,
          assistContentPolicy,
          workAccountSetupConfig,
          wipeDataFlags,
          enterpriseDisplayNameVisibility,
          appFunctions,
          defaultApplicationSettings,
        },
        updateMask: updateMask,
      });
      return patchPoliciesResponse.data;
    }).intercept({ status: 429 }, (err) => {
      // If the Android management API returns a 429 response, log an additional warning that will trigger a help-p1 alert.
      sails.log.warn(`p1: Android management API rate limit exceeded!`);
      return new Error(`When attempting to update a policy for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }).intercept({ status: 400 }, (err) => {
      return {'invalidPolicy': `Attempted to update a policy with an invalid value for an Android enterprise (${androidEnterpriseId}): ${err}`};
    }).intercept({ status: 404 }, (err) => {
      return {'policyNotFound': `Specified policy not found on this Android enterprise (${androidEnterpriseId}): ${err}`};
    }).intercept((err) => {
      return new Error(`When attempting to update a policy for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${require('util').inspect(err)}`);
    });


    // Return the modified policy back to the Fleet server.
    return modifyPoliciesResponse;

  }


};
