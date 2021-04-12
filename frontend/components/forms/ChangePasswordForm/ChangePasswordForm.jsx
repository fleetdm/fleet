import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import validate from "components/forms/ChangePasswordForm/validate";

const formFields = [
  "old_password",
  "new_password",
  "new_password_confirmation",
];
const baseClass = "change-password-form";

class ChangePasswordForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      old_password: formFieldInterface.isRequired,
      new_password: formFieldInterface.isRequired,
      new_password_confirmation: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
  };

  render() {
    const { fields, handleSubmit, onCancel } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        <InputField
          {...fields.old_password}
          autofocus
          label="Original password"
          type="password"
        />
        <InputField
          {...fields.new_password}
          label="New password"
          type="password"
        />
        <InputField
          {...fields.new_password_confirmation}
          label="New password confirmation"
          type="password"
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button type="submit" variant="brand" className={`${baseClass}__btn`}>
            Change password
          </Button>
          <Button
            onClick={onCancel}
            variant="inverse"
            className={`${baseClass}__btn`}
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(ChangePasswordForm, { fields: formFields, validate });
