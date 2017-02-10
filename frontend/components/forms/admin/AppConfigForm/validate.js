import { size, some, trim } from 'lodash';

import APP_CONSTANTS from 'app_constants';
import validJwtToken from 'components/forms/validators/valid_jwt_token';

const { APP_SETTINGS } = APP_CONSTANTS;

export default (formData) => {
  const errors = {};
  const {
    authentication_type: authType,
    kolide_server_url: kolideServerUrl,
    license,
    org_name: orgName,
    password: smtpPassword,
    sender_address: smtpSenderAddress,
    server: smtpServer,
    port: smtpServerPort,
    user_name: smtpUserName,
  } = formData;

  if (!kolideServerUrl) {
    errors.kolide_server_url = 'Kolide Server URL must be present';
  }

  if (!license) {
    errors.license = 'License must be present';
  }

  if (license && !validJwtToken(trim(license))) {
    errors.license = 'License is not a valid JWT token';
  }

  if (!orgName) {
    errors.org_name = 'Organization Name must be present';
  }

  if (some([smtpSenderAddress, smtpServer, smtpUserName]) || (smtpPassword && smtpPassword !== APP_SETTINGS.FAKE_PASSWORD)) {
    if (!smtpSenderAddress) {
      errors.sender_address = 'SMTP Sender Address must be present';
    }

    if (!smtpServer) {
      errors.server = 'SMTP Server must be present';
    }

    if (!smtpServerPort) {
      errors.server = 'SMTP Server Port must be present';
    }

    if (authType !== 'authtype_none') {
      if (!smtpUserName) {
        errors.user_name = 'SMTP Username must be present';
      }

      if (!smtpPassword) {
        errors.password = 'SMTP Password must be present';
      }
    }
  }

  const valid = !size(errors);

  return { valid, errors };
};
