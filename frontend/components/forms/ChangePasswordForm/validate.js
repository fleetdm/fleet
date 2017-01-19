import { size } from 'lodash';
import validateEquality from 'components/forms/validators/validate_equality';

export default (formData) => {
  const errors = {};
  const {
    old_password: oldPassword,
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  if (!oldPassword) {
    errors.old_password = 'Password must be present';
  }

  if (!newPassword) {
    errors.new_password = 'New password must be present';
  }

  if (!newPasswordConfirmation) {
    errors.new_password_confirmation = 'New password confirmation must be present';
  }

  if (newPassword && newPasswordConfirmation &&
    !validateEquality(newPassword, newPasswordConfirmation)) {
    errors.new_password_confirmation = 'New password confirmation does not match new password';
  }

  const valid = !size(errors);

  return { valid, errors };
};
