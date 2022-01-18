import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";

const validate = (formData) => {
  const errors = {};
  // this needs to use useState
  const { email, name } = formData;

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
    // add scroll to here
  }

  if (!validatePresence(name)) {
    errors.name = "Full name field must be completed";
    // add scroll to here
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
