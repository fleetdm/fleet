import { kebabCase, pick } from 'lodash';

const ORG_INFO_ATTRS = ['org_name', 'org_logo_url'];
const ADMIN_ATTRS = ['email', 'name', 'password', 'password_confirmation', 'username'];

const labelSlug = (label) => {
  const { display_text: displayText } = label;

  if (!displayText) return undefined;

  const lowerDisplayText = displayText.toLowerCase();

  return kebabCase(lowerDisplayText);
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

export default { labelSlug, setupData };
