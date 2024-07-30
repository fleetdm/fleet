import React, { Component } from "react";
import PropTypes from "prop-types";

import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import helpers from "./helpers";

const formFields = ["name", "password", "password_confirmation"];
const { validate } = helpers;

class ConfirmInviteForm extends Component {
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
          role="textbox"
          label="Full name"
          placeholder="Full name"
          inputOptions={{
            maxLength: "80",
          }}
        />
        <InputFieldWithIcon
          {...fields.password}
          label="Password"
          placeholder="Password"
          type="password"
          helpText="Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
        />
        <InputFieldWithIcon
          {...fields.password_confirmation}
          label="Confirm password"
          placeholder="Confirm password"
          type="password"
        />
        <Button
          onClick={handleSubmit}
          className="confirm-invite-button"
          type="Submit"
          variant="brand"
        >
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
