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
    className: PropTypes.string,
    currentPage: PropTypes.bool,
    fields: PropTypes.shape({
      org_name: formFieldInterface.isRequired,
      org_logo_url: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { className, currentPage, fields, handleSubmit } = this.props;
    const tabIndex = currentPage ? 1 : -1;

    return (
      <div className={className}>
        <div className="registration-fields">
          <InputFieldWithIcon
            {...fields.org_name}
            placeholder="Organization Name"
            tabIndex={tabIndex}
          />
          <InputFieldWithIcon
            {...fields.org_logo_url}
            placeholder="Organization Logo URL"
            tabIndex={tabIndex}
            hint="must start with https://"
          />
        </div>
        <Button
          onClick={handleSubmit}
          text="Submit"
          variant="gradient"
          tabIndex={tabIndex}
        />
      </div>
    );
  }
}

export default Form(OrgDetails, {
  fields: formFields,
  validate,
});
