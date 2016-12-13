import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputField from 'components/forms/fields/InputField';

const formFields = ['email', 'name', 'position', 'username'];

class UserSettingsForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      position: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
  };

  render () {
    const { fields, handleSubmit, onCancel } = this.props;

    return (
      <form onSubmit={handleSubmit}>
        <InputField
          {...fields.username}
          autofocus
          label="Username (required)"
        />
        <InputField
          {...fields.email}
          label="Email (required)"
        />
        <InputField
          {...fields.name}
          label="Full Name"
        />
        <InputField
          {...fields.position}
          label="Position"
        />
        <Button
          onClick={onCancel}
          text="CANCEL"
          type="button"
          variant="inverse"
        />
        <Button
          text="UPDATE"
          type="submit"
          variant="brand"
        />
      </form>
    );
  }
}

export default Form(UserSettingsForm, { fields: formFields });
