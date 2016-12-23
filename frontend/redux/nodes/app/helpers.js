export const frontendFormattedConfig = (config) => {
  const {
    org_info: orgInfo,
    server_settings: serverSettings,
    smtp_settings: smtpSettings,
  } = config;

  return {
    ...orgInfo,
    ...serverSettings,
    ...smtpSettings,
  };
};

export default { frontendFormattedConfig };
