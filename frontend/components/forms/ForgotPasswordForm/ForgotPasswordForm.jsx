import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import validate from "./validate";

const baseClass = "forgot-password-form";
const fieldNames = ["email"];

class ForgotPasswordForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
  };

  render() {
    const { baseError, fields, handleSubmit } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass} autoComplete="off">
        {baseError && <div className="form__base-error">{baseError}</div>}
        <p>
          Enter your email below to receive an email with instructions to reset
          your password.
        </p>
        <InputFieldWithIcon
          {...fields.email}
          autofocus
          label="Email"
          placeholder="Email"
        />
        <div className="button-wrap">
          <Button className={`${baseClass}__submit-btn`} type="submit">
            Get instructions
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(ForgotPasswordForm, {
  fields: fieldNames,
  validate,
});
