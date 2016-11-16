import React, { Component, PropTypes } from 'react';

import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Button from 'components/buttons/Button';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';
import helpers from './helpers';

const formFields = ['full_name', 'username', 'password', 'password_confirmation', 'email'];
const { validate } = helpers;

class AdminDetails extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
      full_name: formFieldInterface.isRequired,
      password: formFieldInterface.isRequired,
      password_confirmation: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { fields, handleSubmit } = this.props;

    return (
      <div>
        <InputFieldWithIcon
          {...fields.full_name}
          placeholder="Full Name"
        />
        <InputFieldWithIcon
          {...fields.username}
          iconName="kolidecon-username"
          placeholder="Username"
        />
        <InputFieldWithIcon
          {...fields.password}
          iconName="kolidecon-password"
          placeholder="Password"
          type="password"
        />
        <InputFieldWithIcon
          {...fields.password_confirmation}
          iconName="kolidecon-password"
          placeholder="Confirm Password"
          type="password"
        />
        <InputFieldWithIcon
          {...fields.email}
          iconName="kolidecon-email"
          placeholder="Email"
        />
        <Button
          onClick={handleSubmit}
          text="Submit"
          variant="gradient"
        />
      </div>
    );
  }
}

export default Form(AdminDetails, {
  fields: formFields,
  validate,
});
