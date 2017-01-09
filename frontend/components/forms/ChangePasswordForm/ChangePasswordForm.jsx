import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputField from 'components/forms/fields/InputField';
import validate from 'components/forms/ChangePasswordForm/validate';

const formFields = ['password', 'new_password', 'new_password_confirmation'];

class ChangePasswordForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      password: formFieldInterface.isRequired,
      new_password: formFieldInterface.isRequired,
      new_password_confirmation: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
  };

  render () {
    const { fields, handleSubmit, onCancel } = this.props;

    return (
      <form onSubmit={handleSubmit}>
        <InputField
          {...fields.password}
          autofocus
          label="Original Password"
          type="password"
        />
        <InputField
          {...fields.new_password}
          label="New Password"
          type="password"
        />
        <InputField
          {...fields.new_password_confirmation}
          label="New Password Confirmation"
          type="password"
        />
        <Button onClick={onCancel} variant="inverse">CANCEL</Button>
        <Button type="submit" variant="brand">CHANGE PASSWORD</Button>
      </form>
    );
  }
}

export default Form(ChangePasswordForm, { fields: formFields, validate });

