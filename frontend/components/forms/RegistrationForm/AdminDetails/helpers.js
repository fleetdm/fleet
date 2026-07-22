import { size } from "lodash";
import {
  isEqual,
  isValidEmail,
  validatePassword,
} from "components/forms/validators";

const validate = (formData) => {
  const errors = {};
  const {
    email,
    password,
    password_confirmation: passwordConfirmation,
    name,
  } = formData;

  if (!email) {
    errors.email = "Email must be present";
  } else if (!isValidEmail(email)) {
    errors.email = "Email must be a valid email";
  }

  if (!name) {
    errors.name = "Full name must be present";
  }

  const { isValid, error } = validatePassword(password);
  if (password && passwordConfirmation && !isValid) {
    errors.password = error;
  }

  if (
    password &&
    passwordConfirmation &&
    !isEqual(password, passwordConfirmation)
  ) {
    errors.password_confirmation =
      "Password confirmation does not match password";
  }

  if (!password) {
    errors.password = "Password must be present";
  }

  if (!passwordConfirmation) {
    errors.password_confirmation = "Password confirmation must be present";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
