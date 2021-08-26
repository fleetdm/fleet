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
    handleSubmit: PropTypes.func,
    fields: PropTypes.shape({
      new_password: formFieldInterface.isRequired,
      new_password_confirmation: formFieldInterface.isRequired,
    }),
  };

  render() {
    const { fields, handleSubmit } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        <InputFieldWithIcon
          {...fields.new_password}
          autofocus
          placeholder="New password"
          className={`${baseClass}__input`}
          type="password"
          hint={[
            "Must include 7 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
          ]}
        />
        <InputFieldWithIcon
          {...fields.new_password_confirmation}
          placeholder="Confirm password"
          className={`${baseClass}__input`}
          type="password"
        />
        <div className={`${baseClass}__button-wrap`}>
          <Button
            onClick={handleSubmit}
            className={`${baseClass}__btn button button--brand`}
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
