import React, { Component } from "react";
import PropTypes from "prop-types";
import { pull } from "lodash";

import Button from "components/buttons/Button";
import Dropdown from "components/forms/fields/Dropdown";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import validate from "components/forms/ConfigurePackQueryForm/validate";
import {
  FREQUENCY_DROPDOWN_OPTIONS,
  PLATFORM_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
} from "utilities/constants";

const baseClass = "configure-pack-query-form";
const fieldNames = [
  "query_id",
  "interval",
  "logging_type",
  "platform",
  "shard",
  "version",
];

export class ConfigurePackQueryForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      interval: formFieldInterface.isRequired,
      logging_type: formFieldInterface.isRequired,
      platform: formFieldInterface.isRequired,
      version: formFieldInterface.isRequired,
      shard: formFieldInterface.isRequired,
    }).isRequired,
    formData: PropTypes.shape({
      id: PropTypes.number,
    }),
    handleSubmit: PropTypes.func,
    onCancel: PropTypes.func,
  };

  componentWillMount() {
    const { fields } = this.props;

    if (fields && fields.shard && !fields.shard.value) {
      fields.shard.value = "";
    }
  }

  onCancel = (evt) => {
    evt.preventDefault();

    const { formData, onCancel: handleCancel } = this.props;

    return handleCancel(formData);
  };

  handlePlatformChoice = (value) => {
    const {
      fields: { platform },
    } = this.props;
    const valArray = value.split(",");

    // Remove All if another OS is chosen
    if (valArray.indexOf("") === 0 && valArray.length > 1) {
      return platform.onChange(pull(valArray, "").join(","));
    }

    // Remove OS if All is chosen
    if (valArray.length > 1 && valArray.indexOf("") > -1) {
      return platform.onChange("");
    }

    return platform.onChange(value);
  };

  renderCancelButton = () => {
    const { formData } = this.props;
    const { onCancel } = this;

    if (!formData.id) {
      return false;
    }

    return (
      <Button
        className={`${baseClass}__cancel-btn`}
        onClick={onCancel}
        variant="inverse"
      >
        Cancel
      </Button>
    );
  };

  render() {
    const { fields, handleSubmit } = this.props;
    const { handlePlatformChoice, renderCancelButton } = this;

    // Uncontrolled form field defaults to snapshot if !fields.logging_type
    const loggingType = fields.logging_type.value || "snapshot";

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        <h2 className={`${baseClass}__title`}>Configuration</h2>
        <div className={`${baseClass}__fields`}>
          <Dropdown
            {...fields.logging_type}
            options={LOGGING_TYPE_OPTIONS}
            placeholder="- - -"
            label="Logging"
            value={loggingType}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
          />
          <InputField
            {...fields.interval}
            inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
            placeholder="- - -"
            label="Frequency (seconds)"
            // hint="Seconds"
            type="number"
          />
          <Dropdown
            {...fields.platform}
            options={PLATFORM_OPTIONS}
            placeholder="- - -"
            label="Platform"
            onChange={handlePlatformChoice}
            multi
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
          />
          <Dropdown
            {...fields.version}
            options={MIN_OSQUERY_VERSION_OPTIONS}
            placeholder="- - -"
            label="Minimum osquery version"
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
          />
          <InputField
            {...fields.shard}
            inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--shard`}
            placeholder="- - -"
            label="Shard"
            type="number"
          />
          <div className={`${baseClass}__btn-wrapper`}>
            {renderCancelButton()}
            <Button
              className={`${baseClass}__submit-btn`}
              type="submit"
              variant="brand"
            >
              Save
            </Button>
          </div>
        </div>
      </form>
    );
  }
}

export default Form(ConfigurePackQueryForm, {
  fields: fieldNames,
  validate,
});
