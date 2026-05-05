import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";

const baseClass = "change-email-form";

class ChangeEmailForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      password: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
  };

  render() {
    const { fields, handleSubmit, onCancel } = this.props;

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        To update your email you must confirm your password.
        <InputField
          {...fields.password}
          autofocus
          label="Password"
          type="password"
        />
        <div className="modal-cta-wrap">
          <Button type="submit">Submit</Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(ChangeEmailForm, {
  fields: ["password"],
  validate: (formData) => {
    if (!formData.password) {
      return {
        valid: false,
        errors: { password: "Password must be present" },
      };
    }

    return { valid: true, errors: {} };
  },
});
