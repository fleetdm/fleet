import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import Checkbox from "components/forms/fields/Checkbox";
import userInterface from "interfaces/user";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

const baseClass = "invite-user-form";

class InviteUserForm extends Component {
  static propTypes = {
    serverErrors: PropTypes.shape({
      email: PropTypes.string,
      base: PropTypes.string,
    }),
    invitedBy: userInterface,
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
    canUseSSO: PropTypes.bool,
  };

  constructor(props) {
    super(props);

    this.state = {
      errors: {
        admin: null,
        email: null,
        name: null,
        sso_enabled: null,
      },
      formData: {
        admin: false,
        email: "",
        name: "",
        sso_enabled: false,
      },
    };
  }

  componentWillReceiveProps({ serverErrors }) {
    const { errors } = this.state;

    if (this.props.serverErrors !== serverErrors) {
      this.setState({
        errors: {
          ...errors,
          ...serverErrors,
        },
      });
    }
  }

  onInputChange = (formField) => {
    return (value) => {
      const { errors, formData } = this.state;

      this.setState({
        errors: {
          ...errors,
          [formField]: null,
        },
        formData: {
          ...formData,
          [formField]: value,
        },
      });

      return false;
    };
  };

  onCheckboxChange = (formField) => {
    return (evt) => {
      return this.onInputChange(formField)(evt);
    };
  };

  onFormSubmit = (evt) => {
    evt.preventDefault();
    const valid = this.validate();

    if (valid) {
      const {
        formData: { admin, email, name, sso_enabled: ssoEnabled },
      } = this.state;
      const { invitedBy, onSubmit } = this.props;
      return onSubmit({
        admin,
        email,
        invited_by: invitedBy.id,
        name,
        sso_enabled: ssoEnabled,
      });
    }

    return false;
  };

  validate = () => {
    const {
      errors,
      formData: { email },
    } = this.state;

    if (!validatePresence(email)) {
      this.setState({
        errors: {
          ...errors,
          email: "Email field must be completed",
        },
      });

      return false;
    }

    if (!validEmail(email)) {
      this.setState({
        errors: {
          ...errors,
          email: `${email} is not a valid email`,
        },
      });

      return false;
    }

    return true;
  };

  render() {
    const {
      errors,
      formData: { admin, email, name, ssoEnabled },
    } = this.state;
    const { onCancel, serverErrors } = this.props;
    const { onFormSubmit, onInputChange, onCheckboxChange } = this;
    const baseError = serverErrors.base;

    return (
      <form onSubmit={onFormSubmit} className={baseClass}>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputFieldWithIcon
          autofocus
          error={errors.name}
          name="name"
          onChange={onInputChange("name")}
          placeholder="Name"
          value={name}
        />
        <InputFieldWithIcon
          error={errors.email}
          name="email"
          onChange={onInputChange("email")}
          placeholder="Email"
          value={email}
        />
        <div className={`${baseClass}__radio`}>
          <p className={`${baseClass}__role`}>Admin</p>
          <Checkbox
            name="admin"
            onChange={onCheckboxChange("admin")}
            value={admin}
            wrapperClassName={`${baseClass}__invite-admin`}
          >
            Enable Admin
          </Checkbox>
        </div>
        <div className={`${baseClass}__radio`}>
          <p className={`${baseClass}__role`}>Single sign on</p>
          <Checkbox
            name="sso_enabled"
            onChange={onCheckboxChange("sso_enabled")}
            value={ssoEnabled}
            disabled={!this.props.canUseSSO}
            wrapperClassName={`${baseClass}__invite-admin`}
          >
            Enable Single Sign On
          </Checkbox>
        </div>

        <div className={`${baseClass}__btn-wrap`}>
          <Button className={`${baseClass}__btn`} type="submit" variant="brand">
            Invite
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            type="input"
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default InviteUserForm;
