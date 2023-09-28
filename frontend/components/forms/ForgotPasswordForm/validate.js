import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";
import validateEmail from "components/forms/validators/valid_email";

const validate = (formData) => {
  const errors = {};
  const { email } = formData;

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
  } else if (!validateEmail(email)) {
    errors.email = "Email must be a valid email address";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
