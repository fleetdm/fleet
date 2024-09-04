import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import validate from "components/forms/UserSettingsForm/validate";

const formFields = ["email", "name", "position", "username"];

const baseClass = "manage-user";

class UserSettingsForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      position: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    pendingEmail: PropTypes.string,
    onCancel: PropTypes.func.isRequired,
    smtpConfigured: PropTypes.bool,
  };

  renderEmailHelpText = () => {
    const { pendingEmail } = this.props;

    if (!pendingEmail) {
      return undefined;
    }

    return (
      <i className={`${baseClass}__email-help-text`}>
        Pending change to <b>{pendingEmail}</b>
      </i>
    );
  };

  render() {
    const { fields, handleSubmit, onCancel, smtpConfigured } = this.props;
    const { renderEmailHelpText } = this;

    return (
      <form onSubmit={handleSubmit} className={baseClass} autoComplete="off">
        <div
          className="smtp-not-configured"
          data-tip
          data-for="smtp-tooltip"
          data-tip-disable={smtpConfigured}
        >
          <InputField
            {...fields.email}
            autofocus
            label="Email (required)"
            helpText={renderEmailHelpText()}
            readOnly={!smtpConfigured}
            tooltip={
              <>
                Editing your email address requires that SMTP or SES is
                configured in order to send a validation email.
                <br />
                <br />
                Users with Admin role can configure SMTP in{" "}
                <strong>Settings &gt; Organization settings</strong>.
              </>
            }
          />
        </div>
        <InputField
          {...fields.name}
          label="Full name (required)"
          inputOptions={{
            maxLength: "80",
          }}
        />
        <InputField {...fields.position} label="Position" />
        <div className="button-wrap">
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

export default Form(UserSettingsForm, { fields: formFields, validate });
