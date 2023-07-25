import { size } from "lodash";
import validateEquality from "components/forms/validators/validate_equality";
import validatePresence from "components/forms/validators/validate_presence";
import validPassword from "components/forms/validators/valid_password";

const validate = (formData) => {
  const errors = {};
  const {
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  const noMatch =
    newPassword &&
    newPasswordConfirmation &&
    !validateEquality(newPassword, newPasswordConfirmation);

  if (!validPassword(newPassword)) {
    errors.new_password = "Password must meet the criteria below";
  }

  if (!validatePresence(newPasswordConfirmation)) {
    errors.new_password_confirmation =
      "New password confirmation field must be completed";
  }

  if (!validatePresence(newPassword)) {
    errors.new_password = "New password field must be completed";
  }

  if (noMatch) {
    errors.new_password_confirmation = "Passwords do not match";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
