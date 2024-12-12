/**
 * cloud.setup.js
 *
 * Configuration for this Sails app's generated browser SDK ("Cloud").
 *
 * Above all, the purpose of this file is to provide endpoint definitions,
 * each of which corresponds with one particular route+action on the server.
 *
 * > This file was automatically generated.
 * > (To regenerate, run `sails run rebuild-cloud-sdk`)
 */

Cloud.setup({

  /* eslint-disable */
  methods: {"confirmEmail":{"verb":"GET","url":"/email/confirm","args":["token"]},"logout":{"verb":"GET","url":"/api/v1/account/logout","args":[]},"updatePassword":{"verb":"PUT","url":"/api/v1/account/update-password","args":["password"]},"updateProfile":{"verb":"PUT","url":"/api/v1/account/update-profile","args":["fullName","emailAddress"]},"login":{"verb":"PUT","url":"/api/v1/entrance/login","args":["emailAddress","password","rememberMe"]},"sendPasswordRecoveryEmail":{"verb":"POST","url":"/api/v1/entrance/send-password-recovery-email","args":["emailAddress"]},"updatePasswordAndLogin":{"verb":"POST","url":"/api/v1/entrance/update-password-and-login","args":["password","token"]},"deleteProfile":{"verb":"POST","url":"/api/v1/delete-profile","args":["profile"]},"downloadProfile":{"verb":"GET","url":"/download-profile","args":["id","uuid"]},"uploadProfile":{"verb":"POST","url":"/api/v1/upload-profile","args":["newProfile","teams","profileTarget","labelTargetBehavior","labels"]},"editProfile":{"verb":"POST","url":"/api/v1/edit-profile","args":["profile","newTeamIds","newProfile","profileTarget","labelTargetBehavior","labels"]},"getProfiles":{"verb":"GET","url":"/api/v1/get-profiles","args":[]},"getScripts":{"verb":"GET","url":"/api/v1/get-scripts","args":[]},"deleteScript":{"verb":"POST","url":"/api/v1/delete-script","args":["script"]},"downloadScript":{"verb":"GET","url":"/download-script","args":["fleetApid","id"]},"uploadScript":{"verb":"POST","url":"/api/v1/upload-script","args":["newScript","teams"]},"editScript":{"verb":"POST","url":"/api/v1/edit-script","args":["script","newTeamIds","newScript"]},"getSoftware":{"verb":"GET","url":"/api/v1/get-software","args":[]},"downloadSoftware":{"verb":"GET","url":"/download-software","args":["id","fleetApid","teamApid"]},"deleteSoftware":{"verb":"POST","url":"/api/v1/software/delete-software","args":["software"]},"editSoftware":{"verb":"POST","url":"/api/v1/software/edit-software","args":["newSoftware","newTeamIds","software","preInstallQuery","installScript","postInstallScript","uninstallScript"]},"uploadSoftware":{"verb":"POST","url":"/api/v1/software/upload-software","args":["newSoftware","teams"]},"getLabels":{"verb":"GET","url":"/api/v1/get-labels","args":[]}}
  /* eslint-enable */

});
