import React, { Component, PropTypes } from 'react';
import { Link } from 'react-router';
import { noop } from 'lodash';
import classnames from 'classnames';

import avatar from '../../../../assets/images/avatar.svg';
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
        username: '',
        password: '',
      },
    };
  }

  onInputChange = (formField) => {
    return (value) => {
      const { errors, formData } = this.state;
      const { onChange } = this.props;

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
    const { serverErrors, isHidden } = this.props;
    const { errors, formData: { username, password } } = this.state;
    const { onInputChange, onFormSubmit } = this;

    const loginFormClass = classnames(
      baseClass,
      { [`${baseClass}--hidden`]: isHidden }
    );

    return (
      <form onSubmit={onFormSubmit} className={loginFormClass}>
        <div className={`${baseClass}__container`}>
          <img alt="Avatar" src={avatar} />
          <InputFieldWithIcon
            autofocus
            error={errors.username || serverErrors.username}
            iconName="kolidecon-username"
            name="username"
            onChange={onInputChange('username')}
            placeholder="Username or Email"
            value={username}
          />
          <InputFieldWithIcon
            error={errors.password || serverErrors.password}
            iconName="kolidecon-password"
            name="password"
            onChange={onInputChange('password')}
            placeholder="Password"
            type="password"
            value={password}
          />
          <div className={`${baseClass}__forgot-wrap`}>
            <Link className={`${baseClass}__forgot-link`} to={paths.FORGOT_PASSWORD}>Forgot Password?</Link>
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

export default LoginForm;
