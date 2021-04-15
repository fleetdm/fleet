import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

const validate = (formData) => {
  const { email } = formData;
  const errors = {};

  if (!validEmail(email)) {
    errors.email = `${email} is not a valid email`;
  }

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
