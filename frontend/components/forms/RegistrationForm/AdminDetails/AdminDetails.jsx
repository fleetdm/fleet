import React, { Component } from "react";
import PropTypes from "prop-types";

import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import helpers from "./helpers";

const formFields = ["name", "password", "password_confirmation", "email"];
const { validate } = helpers;

class AdminDetails extends Component {
  static propTypes = {
    className: PropTypes.string,
    currentPage: PropTypes.bool,
    fields: PropTypes.shape({
      name: formFieldInterface.isRequired,
      email: formFieldInterface.isRequired,
      password: formFieldInterface.isRequired,
      password_confirmation: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  componentDidUpdate(prevProps) {
    if (
      this.props.currentPage &&
      this.props.currentPage !== prevProps.currentPage
    ) {
      // Component has a transition duration of 300ms set in
      // RegistrationForm/_styles.scss. We need to wait 300ms before
      // calling .focus() to preserve smooth transition.
      setTimeout(() => {
        this.firstInput.input.focus();
      }, 300);
    }
  }

  render() {
    const { className, currentPage, fields, handleSubmit } = this.props;
    const tabIndex = currentPage ? 0 : -1;

    return (
      <form onSubmit={handleSubmit} className={className} autoComplete="off">
        <p>Additional admins can be designated within the Fleet app.</p>
        <InputField
          {...fields.name}
          label="Full name"
          tabIndex={tabIndex}
          autofocus={currentPage}
          ref={(input) => {
            this.firstInput = input;
          }}
          inputOptions={{
            maxLength: "80",
          }}
        />
        <InputField {...fields.email} label="Email" tabIndex={tabIndex} />
        <InputField
          {...fields.password}
          label="Password"
          type="password"
          tabIndex={tabIndex}
          helpText="Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
        />
        <InputField
          {...fields.password_confirmation}
          type="password"
          tabIndex={tabIndex}
          label="Confirm password"
        />
        <Button type="submit" tabIndex={tabIndex} disabled={!currentPage}>
          Next
        </Button>
      </form>
    );
  }
}

export default Form(AdminDetails, {
  fields: formFields,
  validate,
});
