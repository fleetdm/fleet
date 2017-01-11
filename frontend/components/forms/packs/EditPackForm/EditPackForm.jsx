import React, { Component, PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import InputField from 'components/forms/fields/InputField';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';

const fieldNames = ['description', 'name', 'targets'];

class EditPackForm extends Component {
  static propTypes = {
    className: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      targets: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
    onFetchTargets: PropTypes.func,
    targetsCount: PropTypes.number,
  };

  render () {
    const {
      className,
      fields,
      handleSubmit,
      onCancel,
      onFetchTargets,
      targetsCount,
    } = this.props;

    return (
      <form className={className} onSubmit={handleSubmit}>
        <InputField
          {...fields.name}
        />
        <InputField
          {...fields.description}
        />
        <SelectTargetsDropdown
          {...fields.targets}
          label="select pack targets"
          name="selected-pack-targets"
          onFetchTargets={onFetchTargets}
          onSelect={fields.targets.onChange}
          selectedTargets={fields.targets.value}
          targetsCount={targetsCount}
        />
        <Button onClick={onCancel} type="button" variant="inverse">CANCEL</Button>
        <Button type="submit" variant="brand">SAVE</Button>
      </form>
    );
  }
}

export default Form(EditPackForm, {
  fields: fieldNames,
});
