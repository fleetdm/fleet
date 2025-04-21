import React, { Component } from "react";
import PropTypes from "prop-types";

import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import helpers from "./helpers";

const formFields = ["name", "password", "password_confirmation"];
const { validate } = helpers;

class ConfirmSSOInviteForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    className: PropTypes.string,
    fields: PropTypes.shape({
      name: formFieldInterface.isRequired,
      password: formFieldInterface.isRequired,
      password_confirmation: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render() {
    const { baseError, className, fields, handleSubmit } = this.props;

    return (
      <form className={className} autoComplete="off">
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputFieldWithIcon
          {...fields.name}
          autofocus
          placeholder="Full name"
          inputOptions={{
            maxLength: "80",
          }}
        />
        <Button onClick={handleSubmit} type="Submit">
          Submit
        </Button>
      </form>
    );
  }
}

export default Form(ConfirmSSOInviteForm, {
  fields: formFields,
  validate,
});
