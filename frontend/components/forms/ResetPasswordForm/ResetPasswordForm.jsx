import React, { Component, PropTypes } from 'react';

import Button from '../../buttons/Button';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import validatePresence from '../validators/validate_presence';
import validateEquality from '../validators/validate_equality';

class ResetPasswordForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      errors: {
        new_password: null,
        new_password_confirmation: null,
      },
      formData: {
        new_password: '',
        new_password_confirmation: '',
      },
    };
  }

  onFormSubmit = (evt) => {
    evt.preventDefault();
    const { validate } = this;
    const { onSubmit } = this.props;
    const { formData } = this.state;

    if (validate()) {
      return onSubmit(formData);
    }

    return false;
  }

  onInputChange = (inputName) => {
    return (value) => {
      const { formData } = this.state;

      this.setState({
        errors: {
          new_password: null,
          new_password_confirmation: null,
        },
        formData: {
          ...formData,
          [inputName]: value,
        },
      });
    };
  }

  validate = () => {
    const {
      errors,
      formData: {
        new_password: newPassword,
        new_password_confirmation: newPasswordConfirmation,
      },
    } = this.state;

    if (!validatePresence(newPassword)) {
      this.setState({
        errors: {
          ...errors,
          new_password: 'New Password field must be completed',
        },
      });

      return false;
    }

    if (!validatePresence(newPasswordConfirmation)) {
      this.setState({
        errors: {
          ...errors,
          new_password_confirmation: 'New Password Confirmation field must be completed',
        },
      });

      return false;
    }

    if (!validateEquality(newPassword, newPasswordConfirmation)) {
      this.setState({
        errors: {
          ...errors,
          new_password_confirmation: 'Passwords Do Not Match',
        },
      });

      return false;
    }

    return true;
  }

  render () {
    const {
      errors,
      formData: {
        new_password: newPassword,
        new_password_confirmation: newPasswordConfirmation,
      },
    } = this.state;
    const { onFormSubmit, onInputChange } = this;
    const baseClass = 'reset-password-form';

    return (
      <form onSubmit={onFormSubmit} className={baseClass}>
        <InputFieldWithIcon
          autofocus
          error={errors.new_password}
          iconName="lock"
          name="new_password"
          onChange={onInputChange('new_password')}
          placeholder="New Password"
          className={`${baseClass}__input`}
          type="password"
          value={newPassword}
        />
        <InputFieldWithIcon
          error={errors.new_password_confirmation}
          iconName="lock"
          name="new_password_confirmation"
          onChange={onInputChange('new_password_confirmation')}
          placeholder="Confirm Password"
          className={`${baseClass}__input`}
          type="password"
          value={newPasswordConfirmation}
        />
        <Button
          onClick={onFormSubmit}
          className={`${baseClass}__btn`}
          text="Reset Password"
          type="submit"
          variant="gradient"
        />
      </form>
    );
  }
}

export default ResetPasswordForm;
