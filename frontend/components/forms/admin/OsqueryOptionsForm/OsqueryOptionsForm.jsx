import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { size } from 'lodash';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import YamlAce from 'components/YamlAce';
import validateYaml from 'components/forms/validators/validate_yaml';
import constructErrorString from './helpers';

const baseClass = 'osquery-options-form';

const validate = (formData) => {
  const errors = {};
  const {
    error: yamlError,
    valid: yamlValid,
  } = validateYaml(formData.osquery_options);

  if (!yamlValid) {
    errors.osquery_options = constructErrorString(yamlError);
  }

  const valid = !size(errors);
  return { valid, errors };
};

class OsqueryOptionsForm extends Component {
  static propTypes = {
    formData: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    handleSubmit: PropTypes.func.isRequired,
    fields: PropTypes.shape({
      osquery_options: formFieldInterface.isRequired,
    }).isRequired,
  }

  render () {
    const { handleSubmit, fields } = this.props;

    return (
      <form onSubmit={handleSubmit} className={baseClass}>
        <YamlAce
          {...fields.osquery_options}
          error={fields.osquery_options.error}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          label="YAML"
        />
        <Button
          type="submit"
          variant="brand"
        >
          Update options
        </Button>
      </form>
    );
  }
}

export default Form(OsqueryOptionsForm, {
  fields: ['osquery_options'],
  validate,
});
