import React, { Component, PropTypes } from 'react';
import { Link } from 'react-router';
import { noop } from 'lodash';
import radium from 'radium';

import avatar from '../../../../assets/images/avatar.svg';
import componentStyles from './styles';
import Button from '../../buttons/Button';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';
import paths from '../../../router/paths';
import validatePresence from '../validators/validate_presence';

const baseClass = 'login-form';

class LoginForm extends Component {
  static propTypes = {
    serverErrors: PropTypes.shape({
      username: PropTypes.string,
      password: PropTypes.string,
    }),
    onChange: PropTypes.func,
    onSubmit: PropTypes.func,
    isHidden: PropTypes.bool,
  };

  static defaultProps = {
    onChange: noop,
    serverErrors: {},
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
      const { onChange } = this.props;
      const { value } = target;

      onChange(value);

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
      hideForm,
    } = componentStyles;
    const { serverErrors } = this.props;
    const { errors } = this.state;
    const { onInputChange, onFormSubmit } = this;

    const byeFelicia = this.props.isHidden ? hideForm : {};

    return (
      <form onSubmit={onFormSubmit} style={[formStyles, byeFelicia]}>
        <div style={containerStyles}>
          <img alt="Avatar" src={avatar} />
          <InputFieldWithIcon
            autofocus
            error={errors.username || serverErrors.username}
            iconName="kolidecon-username"
            name="username"
            onChange={onInputChange('username')}
            placeholder="Username or Email"
          />
          <InputFieldWithIcon
            error={errors.password || serverErrors.password}
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
        <Button
          className={`${baseClass}__submit-btn`}
          onClick={onFormSubmit}
          text="Login"
          type="submit"
          variant="gradient"
        />
      </form>
    );
  }
}

export default radium(LoginForm);
