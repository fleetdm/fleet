import { size } from "lodash";

import validUrl from "components/forms/validators/valid_url";

import INVALID_SERVER_URL_MESSAGE from "utilities/error_messages";

const validate = (formData) => {
  const errors = {};
  const { server_url: fleetWebAddress } = formData;

  if (!fleetWebAddress) {
    errors.server_url = "Fleet web address must be completed";
  } else if (
    !validUrl({
      url: fleetWebAddress,
      protocols: ["http", "https"],
      allowLocalHost: true,
    })
  ) {
    errors.server_url = INVALID_SERVER_URL_MESSAGE;
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
