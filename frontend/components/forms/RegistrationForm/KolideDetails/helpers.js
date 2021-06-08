import { size, startsWith } from "lodash";

const validate = (formData) => {
  const errors = {};
  const { server_url: kolideWebAddress } = formData;

  if (!kolideWebAddress) {
    errors.server_url = "Kolide web address must be completed";
  }

  if (kolideWebAddress && !startsWith(kolideWebAddress, "https://")) {
    errors.server_url = "Kolide web address must start with https://";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
