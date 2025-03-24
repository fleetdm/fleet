import { size } from "lodash";

import validUrl from "components/forms/validators/valid_url";

const validate = (formData) => {
  const errors = {};
  const { server_url: fleetWebAddress } = formData;

  if (!fleetWebAddress) {
    errors.server_url = "Fleet web address must be completed";
  } else if (
    !validUrl({
      url: fleetWebAddress,
      protocols: ["http", "https"],
      allowAnyLocalHost: true,
    })
  ) {
    errors.server_url =
      "Fleet web address must be a valid https, http, or localhost URL";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
