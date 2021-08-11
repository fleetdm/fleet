import React, { useState } from "react";
import { size } from "lodash";

import Checkbox from "components/forms/fields/Checkbox"; //@ts-ignore
import Form from "components/forms/Form"; //@ts-ignore
import InputField from "components/forms/fields/InputField"; //@ts-ignore
import FleetAce from "components/FleetAce"; //@ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IQuery, IQueryFormFields, IQueryFormData } from "interfaces/query";

import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";

const baseClass = "query-form1";

interface IQueryFormProps {
  baseError: string;
  fields: IQueryFormFields;
  handleSubmit: () => {};
  formData: IQuery;
  onOsqueryTableSelect: (tableName: string) => {};
  onRunQuery: () => {};
  onUpdate: (formData: IQueryFormData) => {};
  queryIsRunning: boolean;
  title: string;
  hasSavePermissions: boolean;
}

const validate = (formData: IQueryFormData) => {
  const errors: {[key: string]: any} = {};
  const { error: queryError, valid: queryValid } = validateQuery(
    formData.query
  );

  if (!queryValid) {
    errors.query = queryError;
  }

  if (!formData.name) {
    errors.name = "Query name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const QueryForm = ({
  baseError,
  fields,
  handleSubmit,
  formData,
  onOsqueryTableSelect,
  onRunQuery,
  onUpdate,
  queryIsRunning,
  title,
  hasSavePermissions,
}: IQueryFormProps) => {
  const [errors, setErrors] = useState<{[key: string]: any}>({});

  const onLoad = (editor: any) => {
    editor.setOptions({
      enableLinking: true,
    });

    editor.on("linkClick", (data: any) => {
      const { type, value } = data.token;

      if (type === "osquery-token") {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const handleUpdate = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    const formData = {
      description: fields.description.value,
      name: fields.name.value,
      query: fields.query.value,
      observer_can_run: fields.observer_can_run.value,
    };

    const { valid, errors: newErrors } = validate(formData);

    if (valid) {
      onUpdate(formData);

      return false;
    }

    setErrors({
      ...errors,
      ...newErrors,
    });

    return false;
  };

  const renderNewQueryButtons = () => {
    return (
      <div className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}>
        {hasSavePermissions && (
          <Button
            className={`${baseClass}__save`}
            variant="brand"
            onClick={handleSubmit}
            disabled={formData.query === fields.query.value}
          >
            Save
          </Button>
        )}
        <Button
          className={`${baseClass}__run`}
          variant="blue-green"
          onClick={onRunQuery}
        >
          Run query
        </Button>
      </div>
    )
  };

  return (
    <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
      <h1>{title}</h1>
      {baseError && <div className="form__base-error">{baseError}</div>}
      {/* {hasSavePermissions && (
        <InputField
          {...fields.name}
          error={fields.name.error || errors.name}
          inputClassName={`${baseClass}__query-name`}
          label="Query name"
        />
      )} */}
      <FleetAce
        {...fields.query}
        error={fields.query.error || errors.query}
        label="Query:"
        name="query editor"
        onLoad={onLoad}
        readOnly={queryIsRunning}
        wrapperClassName={`${baseClass}__text-editor-wrapper`}
        handleSubmit={onRunQuery}
      />
      {/* {hasSavePermissions && (
        <>
          <InputField
            {...fields.description}
            inputClassName={`${baseClass}__query-description`}
            label="Description"
            type="textarea"
          />
          <Checkbox
            {...fields.observer_can_run}
            value={!!fields.observer_can_run.value}
            wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
          >
            Observers can run
          </Checkbox>
          Users with the Observer role will be able to run this query on hosts
          where they have access.
        </>
      )} */}
      {renderNewQueryButtons()}
      <Modal title={"Save query"} onExit={() => {}} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__query-description`}
          label="Description"
          type="textarea"
        />
        <Checkbox
          {...fields.observer_can_run}
          value={!!fields.observer_can_run.value}
          wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
        >
          Observers can run
        </Checkbox>
        Users with the Observer role will be able to run this query on hosts
        where they have access.
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={() => {}}
          >
            Create
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={() => {}}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
    </form>
  );
};

export default Form(QueryForm, {
  fields: ["description", "name", "query", "observer_can_run"],
  validate,
});
