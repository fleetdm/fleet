import { size } from "lodash";

const validate = (formData) => {
  const errors = {};
  const { server_url: fleetWebAddress } = formData;

  if (!fleetWebAddress) {
    errors.server_url = "Fleet web address must be completed";
  }

  // explicitly removed check for "https" scheme

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
