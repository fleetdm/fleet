import React, { useState, useRef, useContext, useEffect } from "react";
import ContentEditable, { ContentEditableEvent } from "react-contenteditable";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { size } from "lodash";

import { IQueryFormFields, IQueryFormData, IQuery } from "interfaces/query";
import { IFormField } from "interfaces/form_field";
import { AppContext } from "context/app";

// @ts-ignore
import Form from "components/forms/Form"; // @ts-ignore
import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/loaders/Spinner";
import NewQueryModal from "./NewQueryModal";

import InfoIcon from "../../../../../assets/images/icon-info-purple-14x14@2x.png";

const baseClass = "query-form";

interface IQueryFormProps {
  baseError: string;
  fields: IQueryFormFields;
  storedQuery: IQuery;
  typedQueryBody: string;
  queryIdForEdit: number | null;
  hasSavePermissions: boolean;
  showOpenSchemaActionText: boolean;
  isStoredQueryLoading: boolean;
  isEditorUsingDefaultQuery: boolean;
  resetField: (fieldName: string) => void;
  onCreateQuery: (formData: IQueryFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IQueryFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
}

interface IRenderProps {
  queryValue: string;
  nameText?: string;
  descText?: string;
  queryError?: any;
  queryOnChange?: any;
  name?: IFormField;
  description?: IFormField;
  observer_can_run?: IFormField;
  observerCanRun?: boolean;
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
  typedQueryBody,
  queryIdForEdit,
  hasSavePermissions,
  showOpenSchemaActionText,
  isStoredQueryLoading,
  isEditorUsingDefaultQuery,
  resetField,
  onCreateQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryFormProps) => {
  const isEditMode = !!queryIdForEdit;
  const nameEditable = useRef(null);
  const descriptionEditable = useRef(null);

  const [errors, setErrors] = useState<{ [key: string]: any }>({});
  const [isSaveModalOpen, setIsSaveModalOpen] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);

  const {
    currentUser,
    isOnlyObserver,
    isGlobalObserver,
    isAnyTeamMaintainer,
    isGlobalMaintainer,
  } = useContext(AppContext);

  const hasTeamMaintainerPermissions = isEditMode
    ? isAnyTeamMaintainer &&
      storedQuery &&
      currentUser &&
      storedQuery.author_id === currentUser.id
    : isAnyTeamMaintainer;

  // Not ideal but we need to reset
  // form values if the query id changes
  // TODO: local states for all forms
  useEffect(() => {
    resetField("observer_can_run");
  }, [queryIdForEdit]);

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

  const renderRunForObserver = ({
    nameText,
    descText,
    queryValue,
    observerCanRun,
  }: IRenderProps) => (
    <form className={`${baseClass}__wrapper`}>
      <h1 className={`${baseClass}__query-name no-hover`}>{nameText}</h1>
      <p className={`${baseClass}__query-description no-hover`}>{descText}</p>
      <Button
        className={`${baseClass}__toggle-sql`}
        variant="text-link"
        onClick={() => setShowQueryEditor(!showQueryEditor)}
        disabled={false}
      >
        {showQueryEditor ? "Hide SQL" : "Show SQL"}
      </Button>
      {showQueryEditor && (
        <FleetAce
          value={queryValue}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
      {observerCanRun && (
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
        >
          <Button
            className={`${baseClass}__run`}
            variant="blue-green"
            onClick={goToSelectTargets}
          >
            Run query
          </Button>
        </div>
      )}
    </form>
  );

  const renderRunForMaintainer = ({
    nameText,
    descText,
    queryValue,
  }: IRenderProps) => (
    <form className={`${baseClass}__wrapper`}>
      <h1 className={`${baseClass}__query-name`}>{nameText}</h1>
      <p className={`${baseClass}__query-description`}>{descText}</p>
      <Button
        className={`${baseClass}__toggle-sql`}
        variant="text-link"
        onClick={() => setShowQueryEditor(!showQueryEditor)}
        disabled={false}
      >
        {showQueryEditor ? "Hide SQL" : "Show SQL"}
      </Button>
      {showQueryEditor && (
        <FleetAce
          value={queryValue}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
      <div
        className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
      >
        <Button
          className={`${baseClass}__run`}
          variant="blue-green"
          onClick={goToSelectTargets}
        >
          Run query
        </Button>
      </div>
    </form>
  );

  const renderCreateForTeamMaintainer = ({
    queryValue,
    queryOnChange,
    queryError,
  }: IRenderProps) => (
    <>
      <form className={`${baseClass}__wrapper`}>
        <h1 className={`${baseClass}__query-name`}>New query</h1>
        {baseError && <div className="form__base-error">{baseError}</div>}
        <FleetAce
          value={queryValue}
          error={queryError}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={queryOnChange}
          handleSubmit={promptSaveQuery}
        />
        {renderLiveQueryWarning()}
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
        >
          <Button
            className={`${baseClass}__run`}
            variant="blue-green"
            onClick={goToSelectTargets}
          >
            Run query
          </Button>
        </div>
      </form>
    </>
  );

  const renderForGlobalAdminOrMaintainer = ({
    nameText,
    descText,
    name,
    description,
    queryValue,
    queryOnChange,
    queryError,
    observer_can_run,
    observerCanRun,
  }: IRenderProps) => (
    <>
      <form className={`${baseClass}__wrapper`}>
        {isEditMode ? (
          <ContentEditable
            className={`${baseClass}__query-name`}
            innerRef={nameEditable}
            html={nameText || ""}
            tagName="h1"
            onChange={(evt: ContentEditableEvent) =>
              name?.onChange(evt.target.value)
            }
          />
        ) : (
          <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>
        )}
        {isEditMode && (
          <ContentEditable
            className={`${baseClass}__query-description`}
            innerRef={descriptionEditable}
            html={descText || "Add description here."}
            onChange={(evt: ContentEditableEvent) =>
              description?.onChange(evt.target.value)
            }
          />
        )}
        {baseError && <div className="form__base-error">{baseError}</div>}
        <FleetAce
          value={queryValue}
          error={queryError}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={queryOnChange}
          handleSubmit={promptSaveQuery}
        />
        {isEditMode && (
          <>
            <Checkbox
              {...observer_can_run}
              value={observerCanRun}
              wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
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
          {(hasSavePermissions || isAnyTeamMaintainer) && (
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
              <div className="query-form__button-wrap--save-query-button">
                <div
                  data-tip
                  data-for="save-query-button"
                  data-tip-disable={
                    !(isAnyTeamMaintainer && !hasTeamMaintainerPermissions)
                  }
                >
                  <Button
                    className={`${baseClass}__save`}
                    variant="brand"
                    onClick={promptSaveQuery()}
                    disabled={
                      isAnyTeamMaintainer && !hasTeamMaintainerPermissions
                    }
                  >
                    Save
                  </Button>
                </div>{" "}
                <ReactTooltip
                  className={`save-query-button-tooltip`}
                  place="bottom"
                  type="dark"
                  effect="solid"
                  backgroundColor="#3e4771"
                  id="save-query-button"
                  data-html
                >
                  <div
                    className={`tooltip`}
                    style={{ width: "152px", textAlign: "center" }}
                  >
                    You can only save changes to a query if you are the author.
                  </div>
                </ReactTooltip>
              </div>
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
      {isSaveModalOpen && (
        <NewQueryModal
          baseClass={baseClass}
          queryValue={queryValue}
          onCreateQuery={onCreateQuery}
          setIsSaveModalOpen={setIsSaveModalOpen}
        />
      )}
    </>
  );

  const { name, description, query, observer_can_run } = fields;
  const nameText = (name?.value || storedQuery.name) as string;
  const descText = (description?.value || storedQuery.description) as string;

  // `typedQueryBody` and `query?.value` will always be the same but
  // `typedQueryBody` keeps the value as user goes to other page steps
  // this makes sure to show what the user typed when they return to the editor
  const queryValue = (typedQueryBody ||
    query?.value ||
    storedQuery.query) as string;

  const queryError = query?.error || errors.query;
  const queryOnChange = query?.onChange;
  const observerCanRun = (typeof observer_can_run?.value !== "undefined"
    ? observer_can_run.value
    : storedQuery.observer_can_run) as boolean;

  if (isStoredQueryLoading) {
    return <Spinner />;
  }

  if (isOnlyObserver || isGlobalObserver) {
    return renderRunForObserver({
      nameText,
      descText,
      queryValue,
      observerCanRun,
    });
  }

  // if (!isEditMode && isAnyTeamMaintainer) {
  //   return renderCreateForTeamMaintainer({
  //     queryValue,
  //     queryOnChange,
  //     queryError,
  //   });
  // }

  // if (isAnyTeamMaintainer) {
  //   return renderRunForMaintainer({ nameText, descText, queryValue });
  // }

  return renderForGlobalAdminOrMaintainer({
    name,
    description,
    queryValue,
    queryError,
    observer_can_run,
    nameText,
    descText,
    observerCanRun,
    queryOnChange,
  });
};

export default Form(QueryForm, {
  fields: ["description", "name", "query", "observer_can_run"],
  validate: validateQuerySQL,
});
