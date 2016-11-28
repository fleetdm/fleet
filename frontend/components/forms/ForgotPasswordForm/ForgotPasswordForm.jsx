import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import helpers from 'components/forms/ForgotPasswordForm/helpers';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';

const baseClass = 'forgot-password-form';
const fieldNames = ['email'];
const { validate } = helpers;

class ForgotPasswordForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      email: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
  };

  render () {
    const { fields, handleSubmit } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        <InputFieldWithIcon
          {...fields.email}
          autofocus
          iconName="email"
          placeholder="Email Address"
        />
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__submit-btn`}
            type="submit"
            text="Reset Password"
            variant="gradient"
          />
        </div>
      </form>
    );
  }
}

export default Form(ForgotPasswordForm, {
  fields: fieldNames,
  validate,
});
