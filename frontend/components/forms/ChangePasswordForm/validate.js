import { size } from "lodash";
import { isEqual, validatePassword } from "components/forms/validators";

export default (formData) => {
  const errors = {};
  const {
    old_password: oldPassword,
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  const { isValid, error } = validatePassword(newPassword);
  if (newPassword && newPasswordConfirmation && !isValid) {
    errors.new_password = error;
  }

  if (!oldPassword) {
    errors.old_password = "Password must be present";
  }

  if (!newPassword) {
    errors.new_password = "New password must be present";
  }

  if (!newPasswordConfirmation) {
    errors.new_password_confirmation =
      "New password confirmation must be present";
  }

  if (
    newPassword &&
    newPasswordConfirmation &&
    !isEqual(newPassword, newPasswordConfirmation)
  ) {
    errors.new_password_confirmation =
      "New password confirmation does not match new password";
  }

  const valid = !size(errors);

  return { valid, errors };
};
