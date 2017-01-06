import { size, some } from 'lodash';

export default (formData) => {
  const errors = {};
  const {
    authentication_type: authType,
    kolide_server_url: kolideServerUrl,
    org_name: orgName,
    password: smtpPassword,
    sender_address: smtpSenderAddress,
    server: smtpServer,
    user_name: smtpUserName,
  } = formData;

  if (!kolideServerUrl) {
    errors.kolide_server_url = 'Kolide Server URL must be present';
  }

  if (!orgName) {
    errors.org_name = 'Organization Name must be present';
  }

  if (some([smtpSenderAddress, smtpPassword, smtpServer, smtpUserName])) {
    if (!smtpSenderAddress) {
      errors.sender_address = 'SMTP Sender Address must be present';
    }

    if (!smtpServer) {
      errors.server = 'SMTP Server must be present';
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
