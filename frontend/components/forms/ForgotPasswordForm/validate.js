import { size } from "lodash";
import { isPresent, isValidEmail } from "components/forms/validators";

const validate = (formData) => {
  const errors = {};
  const { email } = formData;

  if (!isPresent(email)) {
    errors.email = "Email field must be completed";
  } else if (!isValidEmail(email)) {
    errors.email = "Email must be a valid email address";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
