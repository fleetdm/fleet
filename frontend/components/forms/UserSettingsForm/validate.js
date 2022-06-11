import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

const validate = (formData) => {
  const errors = {};
  const { email, name } = formData;

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
  } else if (!validEmail(email)) {
    errors.email = `${email} is not a valid email`;
  }

  if (!validatePresence(name)) {
    errors.name = "Full name field must be completed";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
