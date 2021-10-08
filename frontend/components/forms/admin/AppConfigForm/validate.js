import { size } from "lodash";
import validateYaml from "components/forms/validators/validate_yaml";
import constructErrorString from "utilities/yaml";

export default (formData) => {
  const errors = {};
  const {
    authentication_type: authType,
    server_url: kolideServerUrl,
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
    agent_options: agentOptions,
    enable_host_status_webhook: enableHostStatusWebhook,
    destination_url: destinationUrl,
    host_percentage: hostPercentage,
    days_count: daysCount,
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
    errors.server_url = "Fleet server URL must be present";
  }

  if (!orgName) {
    errors.org_name = "Organization name must be present";
  }

  if (enableSMTP) {
    if (!smtpSenderAddress) {
      errors.sender_address = "SMTP sender address must be present";
    }

    if (!smtpServer) {
      errors.server = "SMTP server must be present";
    }

    if (!smtpServerPort) {
      errors.server = "SMTP server port must be present";
    }

    if (authType !== "authtype_none") {
      if (!smtpUserName) {
        errors.user_name = "SMTP username must be present";
      }

      if (!smtpPassword) {
        errors.password = "SMTP password must be present";
      }
    }
  }

  if (enableHostStatusWebhook) {
    if (!destinationUrl) {
      errors.destination_url = "Destination URL must be present";
    }

    if (!hostPercentage) {
      errors.host_percentage = "Host percentage must be present";
    }

    if (!daysCount) {
      errors.days_count = "Days count must be present";
    }
  }

  if (hostExpiryEnabled) {
    if (isNaN(hostExpiryWindow) || Number(hostExpiryWindow) <= 0) {
      errors.host_expiry_window =
        "Host expiry window must be a positive number";
    }
  }

  if (agentOptions) {
    const { error: yamlError, valid: yamlValid } = validateYaml(
      formData.agent_options
    );

    if (!yamlValid) {
      errors.agent_options = constructErrorString(yamlError);
    }
  }

  const valid = !size(errors);

  return { valid, errors };
};
