import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";
import validate from "./validate";

const fieldNames = ["name", "description", "targets"];
const baseClass = "pack-form";

class PackForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    className: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      targets: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
    onFetchTargets: PropTypes.func,
    selectedTargetsCount: PropTypes.number,
    isBasicTier: PropTypes.bool,
  };

  render() {
    const {
      baseError,
      className,
      fields,
      handleSubmit,
      onFetchTargets,
      selectedTargetsCount,
      isBasicTier,
    } = this.props;

    const packFormClass = classnames(baseClass, className);

    return (
      <form className={packFormClass} onSubmit={handleSubmit}>
        <h1>New pack</h1>
        {baseError && <div className="form__base-error">{baseError}</div>}
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
        <div className={`${baseClass}__pack-targets`}>
          <SelectTargetsDropdown
            {...fields.targets}
            label="Select pack targets"
            onSelect={fields.targets.onChange}
            onFetchTargets={onFetchTargets}
            selectedTargets={fields.targets.value}
            targetsCount={selectedTargetsCount}
            isBasicTier={isBasicTier}
          />
        </div>
        <div className={`${baseClass}__pack-buttons`}>
          <Button type="submit" variant="brand">
            Save query pack
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(PackForm, {
  fields: fieldNames,
  validate,
});
