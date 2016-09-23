import React, { Component, PropTypes } from 'react';
import { Link } from 'react-router';
import radium from 'radium';
import avatar from '../../../../assets/images/avatar.svg';
import componentStyles from './styles';
import GradientButton from '../../buttons/GradientButton';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import paths from '../../../router/paths';
import validatePresence from '../validators/validate_presence';

class LoginForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      errors: {
        username: null,
        password: null,
      },
      formData: {
        username: null,
        password: null,
      },
    };
  }

  onInputChange = (formField) => {
    return ({ target }) => {
      const { errors, formData } = this.state;
      const { value } = target;

      this.setState({
        errors: {
          ...errors,
          [formField]: null,
        },
        formData: {
          ...formData,
          [formField]: value,
        },
      });
    };
  }

  onFormSubmit = (evt) => {
    evt.preventDefault();
    const valid = this.validate();

    if (valid) {
      const { formData } = this.state;
      const { onSubmit } = this.props;

      return onSubmit(formData);
    }

    return false;
  }

  validate = () => {
    const {
      errors,
      formData: { username, password },
    } = this.state;

    if (!validatePresence(username)) {
      this.setState({
        errors: {
          ...errors,
          username: 'Username or email field must be completed',
        },
      });

      return false;
    }

    if (!validatePresence(password)) {
      this.setState({
        errors: {
          ...errors,
          password: 'Password field must be completed',
        },
      });

      return false;
    }

    return true;
  }

  validate = () => {
    const {
      errors,
      formData: { username, password },
    } = this.state;

    if (!validatePresence(username)) {
      this.setState({
        errors: {
          ...errors,
          username: 'Username or email field must be completed',
        },
      });

      return false;
    }

    if (!validatePresence(password)) {
      this.setState({
        errors: {
          ...errors,
          password: 'Password field must be completed',
        },
      });

      return false;
    }

    return true;
  }

  render () {
    const {
      containerStyles,
      forgotPasswordStyles,
      forgotPasswordWrapperStyles,
      formStyles,
      submitButtonStyles,
    } = componentStyles;
    const { errors } = this.state;
    const { onInputChange, onFormSubmit } = this;

    return (
      <form onSubmit={onFormSubmit} style={formStyles}>
        <div style={containerStyles}>
          <img alt="Avatar" src={avatar} />
          <InputFieldWithIcon
            autofocus
            error={errors.username}
            iconName="kolidecon-username"
            name="username"
            onChange={onInputChange('username')}
            placeholder="Username or Email"
          />
          <InputFieldWithIcon
            error={errors.password}
            iconName="kolidecon-password"
            name="password"
            onChange={onInputChange('password')}
            placeholder="Password"
            type="password"
          />
          <div style={forgotPasswordWrapperStyles}>
            <Link style={forgotPasswordStyles} to={paths.FORGOT_PASSWORD}>Forgot Password?</Link>
          </div>
        </div>
        <GradientButton
          onClick={onFormSubmit}
          style={submitButtonStyles}
          text="Login"
          type="submit"
        />
      </form>
    );
  }
}

export default radium(LoginForm);
