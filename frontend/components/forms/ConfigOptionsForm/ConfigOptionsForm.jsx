import React, { Component, PropTypes } from 'react';
import { noop } from 'lodash';

import ConfigOptionForm from 'components/forms/ConfigOptionsForm/ConfigOptionForm';
import configOptionInterface from 'interfaces/config_option';
import dropdownOptionInterface from 'interfaces/dropdownOption';

const baseClass = 'config-options-form';

class ConfigOptionsForm extends Component {
  static propTypes = {
    completedOptions: PropTypes.arrayOf(configOptionInterface),
    configNameOptions: PropTypes.arrayOf(dropdownOptionInterface),
    errors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    onRemoveOption: PropTypes.func.isRequired,
    onFormUpdate: PropTypes.func.isRequired,
  };

  static defaultProps = {
    errors: {},
    onRemoveOption: noop,
    onFormUpdate: noop,
  };

  handleFormUpdate = (option) => {
    return (fieldName, value) => {
      const { onFormUpdate } = this.props;
      const newOption = { ...option, [fieldName]: value };

      return onFormUpdate(option, newOption);
    };
  }

  renderConfigOptionForm = (option, idx) => {
    const { configNameOptions, errors, onRemoveOption } = this.props;
    const { handleFormUpdate } = this;
    const configErrors = errors[option.id] || {};

    return (
      <li className={`${baseClass}__option`} key={`${idx}-config-form-option`}>
        <ConfigOptionForm
          configNameOptions={configNameOptions}
          formData={option}
          key={`config-option-form-${option.id}-${idx}`}
          onChangeFunc={handleFormUpdate(option)}
          onRemove={onRemoveOption}
          serverErrors={configErrors}
          baseClass={baseClass}
        />
      </li>
    );
  }

  render () {
    const { completedOptions } = this.props;
    const { renderConfigOptionForm } = this;

    return (
      <div className={baseClass}>
        <ul className={`${baseClass}__options`}>
          <li className={`${baseClass}__option-header`}>
            <span className={`${baseClass}__option-header-name`}>Option Name</span>
            <span className={`${baseClass}__option-header-value`}>Value</span>
          </li>

          {completedOptions.map((option, idx) => {
            return renderConfigOptionForm(option, idx);
          })}
        </ul>
      </div>
    );
  }
}

export default ConfigOptionsForm;
