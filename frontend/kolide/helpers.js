import { flatMap, kebabCase, pick, size } from 'lodash';
import md5 from 'js-md5';

const ORG_INFO_ATTRS = ['org_name', 'org_logo_url'];
const ADMIN_ATTRS = ['email', 'name', 'password', 'password_confirmation', 'username'];

export const addGravatarUrlToResource = (resource) => {
  const { email } = resource;

  const emailHash = md5(email.toLowerCase());
  const gravatarURL = `https://www.gravatar.com/avatar/${emailHash}?d=blank&size=200`;

  return {
    ...resource,
    gravatarURL,
  };
};

const labelSlug = (label) => {
  const { display_text: displayText } = label;

  if (!displayText) return undefined;

  const lowerDisplayText = displayText.toLowerCase();

  return kebabCase(lowerDisplayText);
};

const filterTarget = (targetType) => {
  return (target) => {
    return target.target_type === targetType ? [target.id] : [];
  };
};

export const formatConfigDataForServer = (config) => {
  const orgInfoAttrs = pick(config, ['org_logo_url', 'org_name']);
  const serverSettingsAttrs = pick(config, ['kolide_server_url', 'osquery_enroll_secret']);
  const smtpSettingsAttrs = pick(config, [
    'authentication_method', 'authentication_type', 'domain', 'email_enabled', 'enable_ssl_tls',
    'enable_start_tls', 'password', 'port', 'sender_address', 'server', 'user_name', 'verify_ssl_certs',
  ]);

  const orgInfo = size(orgInfoAttrs) && { org_info: orgInfoAttrs };
  const serverSettings = size(serverSettingsAttrs) && { server_settings: serverSettingsAttrs };
  const smtpSettings = size(smtpSettingsAttrs) && { smtp_settings: smtpSettingsAttrs };

  return {
    ...orgInfo,
    ...serverSettings,
    ...smtpSettings,
  };
};

export const formatSelectedTargetsForApi = (selectedTargets, appendID = false) => {
  const targets = selectedTargets || [];
  const hosts = flatMap(targets, filterTarget('hosts'));
  const labels = flatMap(targets, filterTarget('labels'));

  if (appendID) {
    return { host_ids: hosts, label_ids: labels };
  }

  return { hosts, labels };
};

const setupData = (formData) => {
  const orgInfo = pick(formData, ORG_INFO_ATTRS);
  const adminInfo = pick(formData, ADMIN_ATTRS);

  return {
    kolide_server_url: formData.kolide_server_url,
    org_info: {
      ...orgInfo,
    },
    admin: {
      admin: true,
      ...adminInfo,
    },
  };
};

export default { addGravatarUrlToResource, formatConfigDataForServer, formatSelectedTargetsForApi, labelSlug, setupData };
