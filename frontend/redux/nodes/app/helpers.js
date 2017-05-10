export const frontendFormattedConfig = (config) => {
  const {
    org_info: orgInfo,
    server_settings: serverSettings,
    smtp_settings: smtpSettings,
    sso_settings: ssoSettings,

  } = config;

  return {
    ...orgInfo,
    ...serverSettings,
    ...smtpSettings,
    ...ssoSettings,
  };
};

export default { frontendFormattedConfig };
