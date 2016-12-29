import React, { Component, PropTypes } from 'react';

import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Button from 'components/buttons/Button';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
import helpers from './helpers';

const formFields = ['name', 'username', 'password', 'password_confirmation'];
const { validate } = helpers;

class ConfirmInviteForm extends Component {
  static propTypes = {
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
    const { className, fields, handleSubmit } = this.props;

    return (
      <form className={className}>
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

