import React, { Component } from "react";
import PropTypes from "prop-types";
import Checkbox from "components/forms/fields/Checkbox";
import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";

const baseClass = "edit-user-form";
const fieldNames = ["email", "name", "position", "username", "sso_enabled"];

class EditUserForm extends Component {
  static propTypes = {
    isCurrentUser: PropTypes.bool.isRequired,
    onCancel: PropTypes.func,
    handleSubmit: PropTypes.func,
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      position: formFieldInterface.isRequired,
      username: formFieldInterface.isRequired,
      sso_enabled: formFieldInterface.isRequired,
    }).isRequired,
  };

  render() {
    const { fields, handleSubmit, onCancel, isCurrentUser } = this.props;

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        <InputField
          {...fields.name}
          label="Name"
          labelClassName={`${baseClass}__label`}
          inputWrapperClass={`${baseClass}__input-wrap ${baseClass}__input-wrap--first`}
          inputClassName={`${baseClass}__input`}
        />
        <InputField
          {...fields.username}
          label="Username"
          labelClassName={`${baseClass}__label`}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input ${baseClass}__input--username`}
        />
        <InputField
          {...fields.position}
          label="Position"
          labelClassName={`${baseClass}__label`}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input`}
        />
        <InputField
          {...fields.email}
          inputWrapperClass={`${baseClass}__input-wrap`}
          label="Email"
          disabled={isCurrentUser}
          labelClassName={`${baseClass}__label`}
          inputClassName={`${baseClass}__input ${baseClass}__input--email`}
        />
        <Checkbox {...fields.sso_enabled}>Enable Single Sign On</Checkbox>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__form-btn ${baseClass}__form-btn--cancel button button--inverse`}
            onClick={onCancel}
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__form-btn ${baseClass}__form-btn--submit button button--brand`}
            type="submit"
          >
            Submit
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(EditUserForm, {
  fields: fieldNames,
});
