import { size, startsWith } from "lodash";

const validate = (formData) => {
  const errors = {};
  const { server_url: fleetWebAddress } = formData;

  if (!fleetWebAddress) {
    errors.server_url = "Fleet web address must be completed";
  }

  if (fleetWebAddress && !startsWith(fleetWebAddress, "https://")) {
    errors.server_url = "Fleet web address must start with https://";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
