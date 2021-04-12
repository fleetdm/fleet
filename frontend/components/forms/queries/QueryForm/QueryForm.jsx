import React, { Component } from "react";
import PropTypes from "prop-types";
import { size } from "lodash";

import DropdownButton from "components/buttons/DropdownButton";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import helpers from "components/forms/queries/QueryForm/helpers";
import InputField from "components/forms/fields/InputField";
import KolideAce from "components/KolideAce";
import queryInterface from "interfaces/query";
import validateQuery from "components/forms/validators/validate_query";

const baseClass = "query-form";

const validate = (formData) => {
  const errors = {};
  const { error: queryError, valid: queryValid } = validateQuery(
    formData.query
  );

  if (!queryValid) {
    errors.query = queryError;
  }

  if (!formData.name) {
    errors.name = "Title must be present";
  }

  const valid = !size(errors);

  return { valid, errors };
};

class QueryForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      query: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    formData: queryInterface,
    onOsqueryTableSelect: PropTypes.func.isRequired,
    onRunQuery: PropTypes.func.isRequired,
    onUpdate: PropTypes.func.isRequired,
    queryIsRunning: PropTypes.bool,
    title: PropTypes.string,
  };

  static defaultProps = {
    targetsCount: 0,
  };

  constructor(props) {
    super(props);

    this.state = { errors: {} };
  }

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

  onUpdate = (evt) => {
    evt.preventDefault();

    const { fields } = this.props;
    const { onUpdate: handleUpdate } = this.props;
    const formData = {
      description: fields.description.value,
      name: fields.name.value,
      query: fields.query.value,
    };

    const { valid, errors } = validate(formData);

    if (valid) {
      handleUpdate(formData);

      return false;
    }

    this.setState({
      errors: {
        ...this.state.errors,
        ...errors,
      },
    });

    return false;
  };

  renderButtons = () => {
    const { canSaveAsNew, canSaveChanges } = helpers;
    const { fields, formData, handleSubmit } = this.props;
    const { onUpdate } = this;

    const dropdownBtnOptions = [
      {
        disabled: !canSaveChanges(fields, formData),
        label: "Save Changes",
        onClick: onUpdate,
      },
      {
        disabled: !canSaveAsNew(fields, formData),
        label: "Save As New...",
        onClick: handleSubmit,
      },
    ];

    return (
      <div className={`${baseClass}__button-wrap`}>
        <DropdownButton
          className={`${baseClass}__save`}
          options={dropdownBtnOptions}
          variant="brand"
        >
          Save
        </DropdownButton>
      </div>
    );
  };

  render() {
    const {
      baseError,
      fields,
      handleSubmit,
      onRunQuery,
      queryIsRunning,
      title,
    } = this.props;
    const { errors } = this.state;
    const { onLoad, renderButtons } = this;

    return (
      <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
        <h1>{title}</h1>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputField
          {...fields.name}
          error={fields.name.error || errors.name}
          inputClassName={`${baseClass}__query-title`}
          label="Query title"
        />
        <KolideAce
          {...fields.query}
          error={fields.query.error || errors.query}
          label="SQL"
          name="query editor"
          onLoad={onLoad}
          readOnly={queryIsRunning}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          handleSubmit={onRunQuery}
        />
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__query-description`}
          label="Description"
          type="textarea"
        />
        {renderButtons()}
      </form>
    );
  }
}

export default Form(QueryForm, {
  fields: ["description", "name", "query"],
  validate,
});
