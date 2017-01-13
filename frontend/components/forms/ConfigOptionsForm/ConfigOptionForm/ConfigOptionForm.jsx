import React, { Component, PropTypes } from 'react';
import { uniq } from 'lodash';

import Button from 'components/buttons/Button';
import Dropdown from 'components/forms/fields/Dropdown';
import dropdownOptionInterface from 'interfaces/dropdownOption';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import Icon from 'components/icons/Icon';
import InputField from 'components/forms/fields/InputField';

const baseClass = 'config-option-form';
const fieldNames = ['name', 'value'];

class ConfigOptionForm extends Component {
  static propTypes = {
    configNameOptions: PropTypes.arrayOf(dropdownOptionInterface),
    fields: PropTypes.shape({
      name: formFieldInterface,
      value: formFieldInterface,
    }),
    formData: PropTypes.shape({
      read_only: PropTypes.bool,
    }).isRequired,
    onRemove: PropTypes.func.isRequired,
  };

  handleRemove = () => {
    const { formData, onRemove } = this.props;

    return onRemove(formData);
  }

  render () {
    const { configNameOptions, fields, formData } = this.props;
    const { handleRemove } = this;
    const { name, read_only: readOnly, value } = formData;
    const inputType = formData.type === 'int' ? 'number' : 'input';
    const options = uniq(configNameOptions.concat({ label: name, value: name, disabled: readOnly || false }));
    const disabled = readOnly || !!(name && value);

    return (
      <form className={baseClass}>
        <Button disabled={readOnly} onClick={handleRemove} variant="unstyled">
          <Icon name="x" onClick={handleRemove} />
        </Button>
        <Dropdown
          {...fields.name}
          className={`${baseClass}__field`}
          disabled={disabled}
          options={options}
        />
        <InputField
          {...fields.value}
          disabled={readOnly}
          inputClassName={`${baseClass}__field`}
          type={inputType}
        />
      </form>
    );
  }
}

export default Form(ConfigOptionForm, { fields: fieldNames });
