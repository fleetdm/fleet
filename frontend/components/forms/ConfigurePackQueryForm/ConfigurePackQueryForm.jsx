import React, { Component } from "react";
import PropTypes from "prop-types";
import { pull } from "lodash";

import KolideIcon from "components/icons/KolideIcon";
import Button from "components/buttons/Button";
import Dropdown from "components/forms/fields/Dropdown";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import InputField from "components/forms/fields/InputField";
import validate from "components/forms/ConfigurePackQueryForm/validate";

const baseClass = "configure-pack-query-form";
const fieldNames = [
  "query_id",
  "interval",
  "logging_type",
  "platform",
  "shard",
  "version",
];
const platformOptions = [
  { label: "All", value: "" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
  { label: "macOS", value: "darwin" },
];
const loggingTypeOptions = [
  { label: "Differential", value: "differential" },
  {
    label: "Differential (Ignore Removals)",
    value: "differential_ignore_removals",
  },
  { label: "Snapshot", value: "snapshot" },
];
const minOsqueryVersionOptions = [
  { label: "All", value: "" },
  { label: "4.7.0 +", value: "4.7.0" },
  { label: "4.6.0 +", value: "4.6.0" },
  { label: "4.5.1 +", value: "4.5.1" },
  { label: "4.5.0 +", value: "4.5.0" },
  { label: "4.4.0 +", value: "4.4.0" },
  { label: "4.3.0 +", value: "4.3.0" },
  { label: "4.2.0 +", value: "4.2.0" },
  { label: "4.1.2 +", value: "4.1.2" },
  { label: "4.1.1 +", value: "4.1.1" },
  { label: "4.1.0 +", value: "4.1.0" },
  { label: "4.0.2 +", value: "4.0.2" },
  { label: "4.0.1 +", value: "4.0.1" },
  { label: "4.0.0 +", value: "4.0.0" },
  { label: "3.4.0 +", value: "3.4.0" },
  { label: "3.3.2 +", value: "3.3.2" },
  { label: "3.3.1 +", value: "3.3.1" },
  { label: "3.2.6 +", value: "3.2.6" },
  { label: "2.2.1 +", value: "2.2.1" },
  { label: "2.2.0 +", value: "2.2.0" },
  { label: "2.1.2 +", value: "2.1.2" },
  { label: "2.1.1 +", value: "2.1.1" },
  { label: "2.0.0 +", value: "2.0.0" },
  { label: "1.8.2 +", value: "1.8.2" },
  { label: "1.8.1 +", value: "1.8.1" },
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

    return (
      <form className={baseClass} onSubmit={handleSubmit}>
        <h2 className={`${baseClass}__title`}>configuration</h2>
        <div className={`${baseClass}__fields`}>
          <InputField
            {...fields.interval}
            inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--interval`}
            placeholder="- - -"
            label="Interval"
            hint="Seconds"
            type="number"
          />
          <Dropdown
            {...fields.platform}
            options={platformOptions}
            placeholder="- - -"
            label="Platform"
            onChange={handlePlatformChoice}
            multi
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
          />
          <Dropdown
            {...fields.version}
            options={minOsqueryVersionOptions}
            placeholder="- - -"
            label={[
              "minimum ",
              <KolideIcon name="osquery" key="min-osquery-vers" />,
              " version",
            ]}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
          />
          <Dropdown
            {...fields.logging_type}
            options={loggingTypeOptions}
            placeholder="- - -"
            label="Logging"
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
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
