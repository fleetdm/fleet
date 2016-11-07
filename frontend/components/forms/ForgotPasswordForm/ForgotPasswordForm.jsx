import React, { Component, PropTypes } from 'react';

import Button from '../../buttons/Button';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import validatePresence from '../validators/validate_presence';
import validEmail from '../validators/valid_email';

const baseClass = 'forgot-password-form';

class ForgotPasswordForm extends Component {
  static propTypes = {
    clearErrors: PropTypes.func,
    error: PropTypes.string,
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      errors: {
        email: null,
      },
      formData: {
        email: '',
      },
    };
  }

  onInputFieldChange = (value) => {
    const { clearErrors, error: serverError } = this.props;

    this.setState({
      errors: {
        email: null,
      },
      formData: {
        email: value,
      },
    });

    if (serverError) {
      clearErrors();
    }

    return false;
  }

  onFormSubmit = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { onSubmit } = this.props;

    if (this.validate()) {
      return onSubmit(formData);
    }

    return false;
  }

  validate = () => {
    const { formData: { email } } = this.state;

    if (!validatePresence(email)) {
      this.setState({
        errors: {
          email: 'Email field must be completed',
        },
      });

      return false;
    }

    if (validEmail(email)) {
      return true;
    }

    this.setState({
      errors: {
        email: `${email} is not a valid email`,
      },
    });

    return false;
  }

  render () {
    const { error: serverError } = this.props;
    const { errors: clientErrors, formData } = this.state;
    const { onFormSubmit, onInputFieldChange } = this;

    return (
      <form onSubmit={onFormSubmit} className={baseClass}>
        <InputFieldWithIcon
          autofocus
          error={clientErrors.email || serverError}
          iconName="kolidecon-email"
          name="email"
          onChange={onInputFieldChange}
          placeholder="Email Address"
          value={formData.email}
        />
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__submit-btn`}
            type="submit"
            text="Reset Password"
            variant="gradient"
          />
        </div>
      </form>
    );
  }
}

export default ForgotPasswordForm;
