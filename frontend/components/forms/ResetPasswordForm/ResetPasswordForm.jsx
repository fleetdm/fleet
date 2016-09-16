import React, { Component, PropTypes } from 'react';
import componentStyles from './styles';
import GradientButton from '../../buttons/GradientButton';
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
        new_password: null,
        new_password_confirmation: null,
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
    return ({ target }) => {
      const { formData } = this.state;
      const { value } = target;

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
    const { errors } = this.state;
    const {
      formStyles,
      inputStyles,
      submitButtonStyles,
    } = componentStyles;
    const { onFormSubmit, onInputChange } = this;

    return (
      <form onSubmit={onFormSubmit} style={formStyles}>
        <InputFieldWithIcon
          error={errors.new_password}
          iconName="lock"
          name="new_password"
          onChange={onInputChange('new_password')}
          placeholder="New Password"
          style={inputStyles}
          type="password"
        />
        <InputFieldWithIcon
          error={errors.new_password_confirmation}
          iconName="lock"
          name="new_password_confirmation"
          onChange={onInputChange('new_password_confirmation')}
          placeholder="Confirm Password"
          style={inputStyles}
          type="password"
        />
        <GradientButton
          onClick={onFormSubmit}
          style={submitButtonStyles}
          text="Reset Password"
          type="submit"
        />
      </form>
    );
  }
}

export default ResetPasswordForm;
