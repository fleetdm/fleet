import React, { useState, useRef } from "react";
import ContentEditable, { ContentEditableEvent } from "react-contenteditable";
import { IAceEditor } from "react-ace/lib/types";
import { size } from "lodash";

import { IQueryFormFields, IQueryFormData, IQuery } from "interfaces/query";

// @ts-ignore
import Form from "components/forms/Form"; // @ts-ignore
import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/loaders/Spinner";
import NewQueryModal from "./NewQueryModal";
import InfoIcon from "../../../../../assets/images/icon-info-purple-14x14@2x.png";

const baseClass = "query-form1";

interface IQueryFormProps {
  baseError: string;
  fields: IQueryFormFields;
  storedQuery: IQuery;
  isEditMode: boolean;
  hasSavePermissions: boolean;
  showOpenSchemaActionText: boolean;
  isStoredQueryLoading: boolean;
  onCreateQuery: (formData: IQueryFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IQueryFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
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
  isEditMode,
  hasSavePermissions,
  showOpenSchemaActionText,
  isStoredQueryLoading,
  onCreateQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryFormProps) => {
  const nameEditable = useRef(null);
  const descriptionEditable = useRef(null);
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

  const promptSaveQuery = (forceNew = false) => (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    let valid = true;
    const { description, name, query, observer_can_run } = fields;

    if (query.value) {
      const { valid: isValidated, errors: newErrors } = validateQuerySQL(
        query.value as string
      );

      valid = isValidated;
      setErrors({
        ...errors,
        ...newErrors,
      });
    }

    if (valid) {
      if (!isEditMode || forceNew) {
        setIsSaveModalOpen(true);
      } else {
        onUpdate({
          description: description.value,
          name: name.value,
          query: query.value,
          observer_can_run: observer_can_run.value,
        });
      }
    }
  };

  const renderLabelComponent = (): JSX.Element | null => {
    if (!showOpenSchemaActionText) {
      return null;
    }

    return (
      <Button variant="text-icon" onClick={onOpenSchemaSidebar}>
        <>
          <img alt="" src={InfoIcon} />
          Show schema
        </>
      </Button>
    );
  };

  if (isStoredQueryLoading) {
    return <Spinner />;
  }

  const { name, description, query, observer_can_run } = fields;
  const nameText = (name?.value || storedQuery.name) as string;
  const descText = (description?.value || storedQuery.description) as string;
  const observerCanRun = (observer_can_run?.value ||
    storedQuery.observer_can_run) as boolean;
  const modalProps = {
    baseClass,
    fields,
    queryValue: fields.query.value,
    onCreateQuery,
    setIsSaveModalOpen,
  };

  return (
    <>
      <form className={`${baseClass}__wrapper`}>
        {isEditMode ? (
          <ContentEditable
            className="query-name"
            innerRef={nameEditable}
            html={nameText}
            tagName="h1"
            onChange={(evt: ContentEditableEvent) =>
              name?.onChange(evt.target.value)
            }
          />
        ) : (
          <h1>New query</h1>
        )}
        {isEditMode && (
          <ContentEditable
            className="description"
            innerRef={descriptionEditable}
            html={descText}
            onChange={(evt: ContentEditableEvent) =>
              description?.onChange(evt.target.value)
            }
          />
        )}
        {baseError && <div className="form__base-error">{baseError}</div>}
        <FleetAce
          value={query.value || storedQuery.query}
          error={query.error || errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={query.onChange}
          handleSubmit={promptSaveQuery}
        />
        {isEditMode && (
          <>
            <Checkbox
              {...observer_can_run}
              value={observerCanRun}
              wrapperClassName={`query-observer-can-run-wrapper`}
            >
              Observers can run
            </Checkbox>
            <p>
              Users with the Observer role will be able to run this query on
              hosts where they have access.
            </p>
          </>
        )}
        {renderLiveQueryWarning()}
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
        >
          {hasSavePermissions && (
            <>
              {isEditMode && (
                <Button
                  className={`${baseClass}__save`}
                  variant="text-link"
                  onClick={promptSaveQuery(true)}
                  disabled={false}
                >
                  Save as new
                </Button>
              )}
              <Button
                className={`${baseClass}__save`}
                variant="brand"
                onClick={promptSaveQuery()}
                disabled={false}
              >
                Save
              </Button>
            </>
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
  fields: ["description", "name", "query", "observer_can_run"],
  validate: validateQuerySQL,
});
