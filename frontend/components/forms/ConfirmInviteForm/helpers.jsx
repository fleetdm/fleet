import { size } from "lodash";
import validateEquality from "components/forms/validators/validate_equality";

const validate = (formData) => {
  const errors = {};
  const {
    name,
    password,
    password_confirmation: passwordConfirmation,
  } = formData;

  if (!name) {
    errors.name = "Full name must be present";
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
