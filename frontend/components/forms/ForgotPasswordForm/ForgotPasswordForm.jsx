import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import GradientButton from '../../buttons/GradientButton';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import validEmail from '../validators/valid_email';

class ForgotPasswordForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      formData: {
        email: null,
      },
    };
  }

  onInputFieldChange = (evt) => {
    const { value } = evt.target;

    this.setState({
      error: null,
      formData: {
        email: value,
      },
    });

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

    if (validEmail(email)) {
      return true;
    }

    this.setState({
      error: `${email} is not a valid email`,
    });

    return false;
  }

  render () {
    const { error, formData: { email } } = this.state;
    const { formStyles, inputStyles, submitButtonStyles } = componentStyles;
    const { onFormSubmit, onInputFieldChange } = this;
    const disabled = !email;

    return (
      <form onSubmit={onFormSubmit} style={formStyles}>
        <InputFieldWithIcon
          error={error}
          iconName="envelope"
          name="email"
          onChange={onInputFieldChange}
          placeholder="Email Address"
          style={inputStyles}
        />
        <GradientButton
          disabled={disabled}
          type="submit"
          style={submitButtonStyles}
          text="Reset Password"
        />
      </form>
    );
  }
}

export default radium(ForgotPasswordForm);
