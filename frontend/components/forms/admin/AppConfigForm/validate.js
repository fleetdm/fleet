import { size } from "lodash";

export default (formData) => {
  const errors = {};
  const {
    authentication_type: authType,
    kolide_server_url: kolideServerUrl,
    org_name: orgName,
    enable_smtp: enableSMTP,
    password: smtpPassword,
    sender_address: smtpSenderAddress,
    server: smtpServer,
    port: smtpServerPort,
    user_name: smtpUserName,
    enable_sso: enableSSO,
    metadata,
    metadata_url: metadataURL,
    entity_id: entityID,
    idp_name: idpName,
    host_expiry_enabled: hostExpiryEnabled,
    host_expiry_window: hostExpiryWindow = 0,
  } = formData;

  if (enableSSO) {
    if (!metadata && !metadataURL) {
      errors.metadata_url = "Metadata URL must be present";
    }
    if (!entityID) {
      errors.entity_id = "Entity ID must be present";
    }
    if (!idpName) {
      errors.idp_name = "Identity Provider Name must be present";
    }
  }

  if (!kolideServerUrl) {
    errors.kolide_server_url = "Fleet Server URL must be present";
  }

  if (!orgName) {
    errors.org_name = "Organization Name must be present";
  }

  if (enableSMTP) {
    if (!smtpSenderAddress) {
      errors.sender_address = "SMTP Sender Address must be present";
    }

    if (!smtpServer) {
      errors.server = "SMTP Server must be present";
    }

    if (!smtpServerPort) {
      errors.server = "SMTP Server Port must be present";
    }

    if (authType !== "authtype_none") {
      if (!smtpUserName) {
        errors.user_name = "SMTP Username must be present";
      }

      if (!smtpPassword) {
        errors.password = "SMTP Password must be present";
      }
    }
  }

  if (hostExpiryEnabled) {
    if (isNaN(hostExpiryWindow) || Number(hostExpiryWindow) <= 0) {
      errors.host_expiry_window =
        "Host Expiry Window must be a positive number";
    }
  }

  const valid = !size(errors);

  return { valid, errors };
};
