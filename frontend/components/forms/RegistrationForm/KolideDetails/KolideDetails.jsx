import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Button from 'components/buttons/Button';
import helpers from 'components/forms/RegistrationForm/KolideDetails/helpers';
import InputFieldWithIcon from 'components/forms/fields/InputFieldWithIcon';

const formFields = ['kolide_server_url'];
const { validate } = helpers;

class KolideDetails extends Component {
  static propTypes = {
    className: PropTypes.string,
    currentPage: PropTypes.bool,
    fields: PropTypes.shape({
      kolide_server_url: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  render () {
    const { className, currentPage, fields, handleSubmit } = this.props;
    const tabIndex = currentPage ? 1 : -1;

    return (
      <form onSubmit={handleSubmit} className={className}>
        <div className="registration-fields">
          <InputFieldWithIcon
            {...fields.kolide_server_url}
            placeholder="Fleet Web Address"
            tabIndex={tabIndex}
            hint={['Don’t include ', <code key="hint">/v1</code>, ' or any other path']}
          />
        </div>
        <Button type="submit" variant="gradient" tabIndex={tabIndex} disabled={!currentPage}>
          Submit
        </Button>
      </form>
    );
  }
}

export default Form(KolideDetails, {
  fields: formFields,
  validate,
});
