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
    name,
  } = formData;

  if (!email) {
    errors.email = "Email must be present";
  } else if (!validEmail(email)) {
    errors.email = "Email must be a valid email";
  }

  if (!name) {
    errors.name = "Full name must be present";
  }

  if (password && passwordConfirmation && !validPassword(password)) {
    errors.password = "Password must meet the criteria below";
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
