import { size, startsWith } from "lodash";

const validate = (formData) => {
  const errors = {};
  const { org_name: orgName, org_logo_url: orgLogoUrl } = formData;

  if (!orgName) {
    errors.org_name = "Organization name must be present";
  }

  if (orgLogoUrl && !startsWith(orgLogoUrl, "https://")) {
    errors.org_logo_url = "Organization logo URL must start with https://";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
