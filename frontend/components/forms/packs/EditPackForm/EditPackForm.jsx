import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";

const fieldNames = ["description", "name", "targets"];
const baseClass = "edit-pack-form";

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
    isBasicTier: PropTypes.bool,
  };

  render() {
    const {
      className,
      fields,
      handleSubmit,
      onCancel,
      onFetchTargets,
      targetsCount,
      isBasicTier,
    } = this.props;

    return (
      <form className={`${baseClass} ${className}`} onSubmit={handleSubmit}>
        <h1>Edit pack</h1>
        <InputField
          {...fields.name}
          placeholder="Query pack title"
          label="Query pack title"
          inputWrapperClass={`${baseClass}__pack-title`}
        />
        <InputField
          {...fields.description}
          inputWrapperClass={`${baseClass}__pack-description`}
          label="Description"
          placeholder="Add a description of your pack"
          type="textarea"
        />
        <SelectTargetsDropdown
          {...fields.targets}
          label="Select pack targets"
          name="selected-pack-targets"
          onFetchTargets={onFetchTargets}
          onSelect={fields.targets.onChange}
          selectedTargets={fields.targets.value}
          targetsCount={targetsCount}
          isBasicTier={isBasicTier}
        />
        <div className={`${baseClass}__pack-buttons`}>
          <Button onClick={onCancel} type="button" variant="inverse">
            Cancel
          </Button>
          <Button type="submit" variant="brand">
            Save
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(EditPackForm, {
  fields: fieldNames,
});
