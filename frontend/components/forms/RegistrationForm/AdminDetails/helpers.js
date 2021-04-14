import { size } from "lodash";
import validateEquality from "components/forms/validators/validate_equality";
import validEmail from "components/forms/validators/valid_email";
import validPassword from "components/forms/validators/valid_password";

const validate = (formData) => {
  const errors = {};
  const {
    email,
    password,
    password_confirmation: passwordConfirmation,
    username,
  } = formData;

  if (!validEmail(email)) {
    errors.email = "Email must be a valid email";
  }

  if (!email) {
    errors.email = "Email must be present";
  }

  if (!username) {
    errors.username = "Username must be present";
  }

  if (password && passwordConfirmation && !validPassword(password)) {
    errors.password =
      "Password must be at least 7 characters and contain at least 1 letter, 1 number, and 1 symbol";
  }

  if (
    password &&
    passwordConfirmation &&
    !validateEquality(password, passwordConfirmation)
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
