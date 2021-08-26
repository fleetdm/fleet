import React, { Component } from "react";
import PropTypes from "prop-types";
import { noop } from "lodash";

import Button from "components/buttons/Button";
import Dropdown from "components/forms/fields/Dropdown";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import helpers from "components/forms/queries/QueryForm/helpers";
import InputField from "components/forms/fields/InputField";
import FleetAce from "components/FleetAce";
import validate from "components/forms/LabelForm/validate";

const baseClass = "label-form";

const PLATFORM_STRINGS = {
  darwin: "macOS",
  windows: "MS Windows",
  ubuntu: "Ubuntu Linux",
  centos: "CentOS Linux",
};

class LabelForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      platform: formFieldInterface.isRequired,
      query: formFieldInterface.isRequired,
    }).isRequired,
    formData: PropTypes.shape({
      type: PropTypes.string,
      label_type: PropTypes.string,
      label_membership_type: PropTypes.string,
    }),
    handleSubmit: PropTypes.func.isRequired,
    isEdit: PropTypes.bool,
    onCancel: PropTypes.func.isRequired,
    onOsqueryTableSelect: PropTypes.func,
  };

  static defaultProps = {
    isEdit: false,
  };

  onLoad = (editor) => {
    editor.setOptions({
      enableLinking: true,
    });

    editor.on("linkClick", (data) => {
      const { type, value } = data.token;
      const { onOsqueryTableSelect } = this.props;

      if (type === "osquery-token") {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  render() {
    const {
      baseError,
      fields,
      handleSubmit,
      isEdit,
      onCancel,
      formData,
    } = this.props;
    const { onLoad } = this;
    const isBuiltin =
      formData &&
      (formData.label_type === "builtin" || formData.type === "status");
    const isManual = formData && formData.label_membership_type === "manual";
    const headerText = isEdit ? "Edit label" : "New label";
    const saveBtnText = isEdit ? "Update label" : "Save label";
    const aceHintText = isEdit
      ? "Label queries are immutable. To change the query, delete this label and create a new one."
      : "";

    const { platform } = fields;

    if (isBuiltin) {
      return (
        <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
          <h1>Built in labels cannot be edited</h1>
        </form>
      );
    }

    return (
      <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
        <h1>{headerText}</h1>
        {!isManual && (
          <FleetAce
            {...fields.query}
            label="SQL"
            onLoad={onLoad}
            readOnly={isEdit}
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            hint={aceHintText}
            handleSubmit={noop}
          />
        )}

        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputField
          {...fields.name}
          inputClassName={`${baseClass}__label-title`}
          label="Name"
        />
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__label-description`}
          label="Description"
          type="textarea"
        />
        {!isManual && !isEdit && (
          <div className="form-field form-field--dropdown">
            <label className="form-field__label" htmlFor="platform">
              Platform
            </label>
            <Dropdown {...fields.platform} options={helpers.platformOptions} />
          </div>
        )}
        {isEdit && platform && (
          <div className={`${baseClass}__label-platform`}>
            <p className="title">Platform</p>
            <p>
              {!platform.value
                ? "All platforms"
                : PLATFORM_STRINGS[platform.value]}
            </p>
            <p className="hint">
              Label platforms are immutable. To change the platform, delete this
              label and create a new one.
            </p>
          </div>
        )}
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__cancel-btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__save-btn`}
            type="submit"
            variant="brand"
          >
            {saveBtnText}
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(LabelForm, {
  fields: ["description", "name", "platform", "query"],
  validate,
});
