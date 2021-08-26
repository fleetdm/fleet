import React, { useState } from "react";
import { size } from "lodash";
import { IAceEditor } from "react-ace/lib/types";

import { IQueryFormFields, IQueryFormData, IQuery } from "interfaces/query";

// @ts-ignore
import Form from "components/forms/Form"; // @ts-ignore
import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import NewQueryModal from "./NewQueryModal";

const baseClass = "query-form1";

interface IQueryFormProps {
  baseError: string;
  fields: IQueryFormFields;
  storedQuery: IQuery;
  onCreateQuery: (formData: IQueryFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: (value: any) => void;
  onUpdate: (formData: IQueryFormData) => void;
  title: string;
  hasSavePermissions: boolean;
}

const validateQuerySQL = (query: string) => {
  const errors: { [key: string]: any } = {};
  const { error: queryError, valid: queryValid } = validateQuery(query);

  if (!queryValid) {
    errors.query = queryError;
  }

  const valid = !size(errors);
  return { valid, errors };
};

const QueryForm = ({
  baseError,
  fields,
  storedQuery,
  onCreateQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  title,
  hasSavePermissions,
}: IQueryFormProps) => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({});
  const [isSaveModalOpen, setIsSaveModalOpen] = useState<boolean>(false);

  const onLoad = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
    });

    // @ts-expect-error
    // the string "linkClick" is not officially in the lib but we need it
    editor.on("linkClick", (data: EditorSession) => {
      const { type, value } = data.token;

      if (type === "osquery-token") {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const openSaveModal = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    const { query } = fields;
    const { valid, errors: newErrors } = validateQuerySQL(
      query.value as string
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    valid && setIsSaveModalOpen(true);
  };

  const modalProps = {
    baseClass,
    fields,
    queryValue: fields.query.value,
    onCreateQuery,
    setIsSaveModalOpen,
  };
  const {
    query: { error, onChange, value },
  } = fields;
  return (
    <>
      <form className={`${baseClass}__wrapper`}>
        <h1>{title}</h1>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <FleetAce
          value={value || storedQuery.query}
          error={error || errors.query}
          label="Query:"
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={onChange}
          handleSubmit={openSaveModal}
        />
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
        >
          {hasSavePermissions && (
            <Button
              className={`${baseClass}__save`}
              variant="brand"
              onClick={openSaveModal}
              disabled={false}
            >
              Save
            </Button>
          )}
          <Button
            className={`${baseClass}__run`}
            variant="blue-green"
            onClick={goToSelectTargets}
          >
            Run query
          </Button>
        </div>
      </form>
      {isSaveModalOpen && <NewQueryModal {...modalProps} />}
    </>
  );
};

export default Form(QueryForm, {
  fields: ["query"],
  validate: validateQuerySQL,
});
