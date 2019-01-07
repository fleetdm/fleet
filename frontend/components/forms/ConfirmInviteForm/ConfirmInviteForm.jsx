import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Button from 'components/buttons/Button';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
import helpers from './helpers';

const formFields = ['name', 'username', 'password', 'password_confirmation'];
const { validate } = helpers;

class ConfirmInviteForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    className: PropTypes.string,
    fields: PropTypes.shape({
      name: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
      password: formFieldInterface.isRequired,
      password_confirmation: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { baseError, className, fields, handleSubmit } = this.props;

    return (
      <form className={className}>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <div className="fields">
          <InputFieldWithIcon
            {...fields.name}
            autofocus
            placeholder="Full Name"
          />
          <InputFieldWithIcon
            {...fields.username}
            iconName="username"
            placeholder="Username"
          />
          <InputFieldWithIcon
            {...fields.password}
            iconName="password"
            placeholder="Password"
            type="password"
          />
          <InputFieldWithIcon
            {...fields.password_confirmation}
            iconName="password"
            placeholder="Confirm Password"
            type="password"
          />
        </div>
        <Button onClick={handleSubmit} type="Submit" variant="gradient">
          Submit
        </Button>
      </form>
    );
  }
}

export default Form(ConfirmInviteForm, {
  fields: formFields,
  validate,
});

