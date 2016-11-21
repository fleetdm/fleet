import { size } from 'lodash';
import validatePresence from 'components/forms/validators/validate_presence';
import validateEquality from 'components/forms/validators/validate_equality';

const validate = (formData) => {
  const errors = {};
  const {
    new_password: newPassword,
    new_password_confirmation: newPasswordConfirmation,
  } = formData;

  if (!validatePresence(newPasswordConfirmation)) {
    errors.new_password_confirmation = 'New Password Confirmation field must be completed';
  }

  if (!validatePresence(newPassword)) {
    errors.new_password = 'New Password field must be completed';
  }

  if (newPassword && newPasswordConfirmation && !validateEquality(newPassword, newPasswordConfirmation)) {
    errors.new_password_confirmation = 'Passwords Do Not Match';
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
