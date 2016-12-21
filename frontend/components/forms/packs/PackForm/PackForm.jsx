import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputField from 'components/forms/fields/InputField';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import validate from './validate';

const fieldNames = ['name', 'description', 'targets'];
const baseClass = 'pack-form';

class PackForm extends Component {
  static propTypes = {
    className: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      targets: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
    onFetchTargets: PropTypes.func,
    selectedTargetsCount: PropTypes.number,
  };

  render () {
    const {
      className,
      fields,
      handleSubmit,
      onFetchTargets,
      selectedTargetsCount,
    } = this.props;

    const packFormClass = classnames(baseClass, className);

    return (
      <form className={packFormClass} onSubmit={handleSubmit}>
        <h1>New Pack</h1>
        <InputField
          {...fields.name}
          placeholder="Query Pack Title"
          label="Query Pack Title"
          inputWrapperClass={`${baseClass}__pack-title`}
        />
        <InputField
          {...fields.description}
          inputWrapperClass={`${baseClass}__pack-description`}
          label="Query Pack Description"
          placeholder="Add a description of your query"
          type="textarea"
        />
        <div className={`${baseClass}__pack-targets`}>
          <SelectTargetsDropdown
            {...fields.targets}
            label="Select Pack Targets"
            onSelect={fields.targets.onChange}
            onFetchTargets={onFetchTargets}
            selectedTargets={fields.targets.value}
            targetsCount={selectedTargetsCount}
          />
        </div>
        <div className={`${baseClass}__pack-buttons`}>
          <Button
            text="Save Query pack"
            type="submit"
            variant="brand"
          />
        </div>
      </form>
    );
  }
}

export default Form(PackForm, {
  fields: fieldNames,
  validate,
});
