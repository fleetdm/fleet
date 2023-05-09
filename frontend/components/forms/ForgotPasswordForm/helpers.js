import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

const validate = (formData) => {
  const { email } = formData;
  const errors = {};

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
  } else if (!validEmail(email)) {
    errors.email = "Email must be a valid email address";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
