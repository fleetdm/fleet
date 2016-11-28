import React, { Component, PropTypes } from 'react';
import { Link } from 'react-router';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
import paths from 'router/paths';
import validate from 'components/forms/LoginForm/validate';
import avatar from '../../../../assets/images/avatar.svg';

const baseClass = 'login-form';
const formFields = ['username', 'password'];

class LoginForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      password: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
    isHidden: PropTypes.bool,
  };

  render () {
    const { fields, handleSubmit, isHidden } = this.props;

    const loginFormClass = classnames(
      baseClass,
      { [`${baseClass}--hidden`]: isHidden }
    );

    return (
      <form onSubmit={handleSubmit} className={loginFormClass}>
        <div className={`${baseClass}__container`}>
          <img alt="Avatar" src={avatar} />
          <InputFieldWithIcon
            {...fields.username}
            autofocus
            iconName="username"
            placeholder="Username or Email"
          />
          <InputFieldWithIcon
            {...fields.password}
            iconName="password"
            placeholder="Password"
            type="password"
          />
          <div className={`${baseClass}__forgot-wrap`}>
            <Link className={`${baseClass}__forgot-link`} to={paths.FORGOT_PASSWORD}>Forgot Password?</Link>
          </div>
        </div>
        <Button
          className={`${baseClass}__submit-btn`}
          onClick={handleSubmit}
          text="Login"
          type="submit"
          variant="gradient"
        />
      </form>
    );
  }
}

export default Form(LoginForm, {
  fields: formFields,
  validate,
});
