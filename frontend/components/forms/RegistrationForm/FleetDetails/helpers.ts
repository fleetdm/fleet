import { size, startsWith } from "lodash";
import {
  IRegistrationFormData,
  IRegistrationFormErrors,
} from "interfaces/registration_form_data";

const validate = (formData: IRegistrationFormData) => {
  const errors: IRegistrationFormErrors = {};
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
