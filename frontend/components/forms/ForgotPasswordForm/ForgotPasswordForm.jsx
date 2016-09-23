import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import GradientButton from '../../buttons/GradientButton';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import validatePresence from '../validators/validate_presence';
import validEmail from '../validators/valid_email';

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
        email: null,
      },
    };
  }

  onInputFieldChange = (evt) => {
    const { clearErrors, error: serverError } = this.props;
    const { value } = evt.target;

    this.setState({
      errors: {
        email: null,
      },
      formData: {
        email: value,
      },
    });

    if (serverError) clearErrors();

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
    const { errors: clientErrors } = this.state;
    const { formStyles, submitButtonContainerStyles, submitButtonStyles } = componentStyles;
    const { onFormSubmit, onInputFieldChange } = this;

    return (
      <form onSubmit={onFormSubmit} style={formStyles}>
        <InputFieldWithIcon
          autofocus
          error={clientErrors.email || serverError}
          iconName="kolidecon-email"
          name="email"
          onChange={onInputFieldChange}
          placeholder="Email Address"
        />
        <div style={submitButtonContainerStyles}>
          <GradientButton
            type="submit"
            text="Reset Password"
            style={submitButtonStyles}
          />
        </div>
      </form>
    );
  }
}

export default radium(ForgotPasswordForm);
