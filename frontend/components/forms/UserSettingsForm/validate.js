import { size } from "lodash";
import { isPresent, isValidEmail } from "components/forms/validators";

const validate = (formData) => {
  const errors = {};
  const { email, name } = formData;

  if (!isPresent(email)) {
    errors.email = "Email field must be completed";
  } else if (!isValidEmail(email)) {
    errors.email = `${email} is not a valid email`;
  }

  if (!isPresent(name)) {
    errors.name = "Full name field must be completed";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
