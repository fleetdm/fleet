import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";

const formFields = ["email", "name", "position", "username"];

const baseClass = "manage-user";

class UserSettingsForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      position: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    pendingEmail: PropTypes.string,
    onCancel: PropTypes.func.isRequired,
  };

  renderEmailHint = () => {
    const { pendingEmail } = this.props;

    if (!pendingEmail) {
      return undefined;
    }

    return (
      <i className={`${baseClass}__email-hint`}>
        Pending change to <b>{pendingEmail}</b>
      </i>
    );
  };

  render() {
    const { fields, handleSubmit, onCancel } = this.props;
    const { renderEmailHint } = this;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        <InputField
          {...fields.username}
          autofocus
          label="Username (required)"
        />
        <InputField
          {...fields.email}
          label="Email (required)"
          hint={renderEmailHint()}
        />
        <InputField {...fields.name} label="Full Name" />
        <InputField {...fields.position} label="Position" />
        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
          <Button type="submit" variant="brand">
            Update
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(UserSettingsForm, { fields: formFields });
