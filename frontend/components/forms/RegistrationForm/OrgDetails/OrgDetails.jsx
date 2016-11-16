import React, { Component, PropTypes } from 'react';

import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Button from 'components/buttons/Button';
import helpers from 'components/forms/RegistrationForm/OrgDetails/helpers';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';

const formFields = ['org_name', 'org_logo_url'];
const { validate } = helpers;

class OrgDetails extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      org_name: formFieldInterface.isRequired,
      org_logo_url: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { fields, handleSubmit } = this.props;

    return (
      <div>
        <InputFieldWithIcon
          {...fields.org_name}
          placeholder="Organization Name"
        />
        <InputFieldWithIcon
          {...fields.org_logo_url}
          placeholder="Organization Logo URL (must start with https://)"
        />
        <Button
          onClick={handleSubmit}
          text="Submit"
          variant="gradient"
        />
      </div>
    );
  }
}

export default Form(OrgDetails, {
  fields: formFields,
  validate,
});
