import React, { Component } from "react";
import PropTypes from "prop-types";

import ReactTooltip from "react-tooltip";
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
    const { fields, handleSubmit, onCancel, smtpConfigured } = this.props;
    const { renderEmailHint } = this;

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
            hint={renderEmailHint()}
            disabled={!smtpConfigured}
          />
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="smtp-tooltip"
          backgroundColor="#3e4771"
          data-html
        >
          <span className={`${baseClass}__tooltip-text`}>
            Editing your email address requires that SMTP is <br />
            configured in order to send a validation email. <br />
            <br />
            Users with Admin role can configure SMTP in
            <br />
            <strong>Settings &gt; Organization settings</strong>.
          </span>
        </ReactTooltip>
        <InputField {...fields.name} label="Full name (required)" />
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

export default Form(UserSettingsForm, { fields: formFields, validate });
