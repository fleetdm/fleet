import { size } from "lodash";
import validatePresence from "components/forms/validators/validate_presence";

const validate = (formData) => {
  const errors = {};
  const { password, email } = formData;

  if (!validatePresence(email)) {
    errors.email = "Email field must be completed";
  }

  if (!validatePresence(password)) {
    errors.password = "Password field must be completed";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
