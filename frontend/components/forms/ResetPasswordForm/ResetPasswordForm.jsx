import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import validate from "components/forms/ResetPasswordForm/validate";

const baseClass = "reset-password-form";
const formFields = ["new_password", "new_password_confirmation"];

class ResetPasswordForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    handleSubmit: PropTypes.func,
    fields: PropTypes.shape({
      new_password: formFieldInterface.isRequired,
      new_password_confirmation: formFieldInterface.isRequired,
    }),
  };

  render() {
    const { baseError, fields, handleSubmit } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputFieldWithIcon
          {...fields.new_password}
          autofocus
          label="New password"
          placeholder="New password"
          className={`${baseClass}__input`}
          type="password"
          hint={[
            "Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
          ]}
        />
        <InputFieldWithIcon
          {...fields.new_password_confirmation}
          label="Confirm password"
          placeholder="Confirm password"
          className={`${baseClass}__input`}
          type="password"
        />
        <div className={`${baseClass}__button-wrap`}>
          <Button
            variant="brand"
            onClick={handleSubmit}
            className={`${baseClass}__btn`}
            type="submit"
          >
            Reset password
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(ResetPasswordForm, {
  fields: formFields,
  validate,
});
