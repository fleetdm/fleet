import React, { useState, useRef, useContext } from "react";
import ContentEditable, { ContentEditableEvent } from "react-contenteditable";
import { IAceEditor } from "react-ace/lib/types";
import { size } from "lodash";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { IQueryFormData } from "interfaces/query";

import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/loaders/Spinner";
import NewQueryModal from "../NewQueryModal";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";

const baseClass = "query-form";

interface IQueryFormProps {
  queryIdForEdit: number | null;
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
  queryIdForEdit,
  showOpenSchemaActionText,
  isStoredQueryLoading,
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
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
  } = useContext(QueryContext);

  const {
    isOnlyObserver,
    isGlobalObserver,
    isAnyTeamMaintainer,
    isGlobalAdmin,
    isGlobalMaintainer,
  } = useContext(AppContext);

  const hasSavePermissions = isGlobalAdmin || isGlobalMaintainer;

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
    const { valid: isValidated, errors: newErrors } = validateQuerySQL(
      lastEditedQueryBody
    );

    valid = isValidated;
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      if (!isEditMode || forceNew) {
        setIsSaveModalOpen(true);
      } else {
        onUpdate({
          name: lastEditedQueryName,
          description: lastEditedQueryDescription,
          query: lastEditedQueryBody,
          observer_can_run: lastEditedQueryObserverCanRun,
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

  const renderRunForObserver = (
    <form className={`${baseClass}__wrapper`}>
      <h1 className={`${baseClass}__query-name no-hover`}>{lastEditedQueryName}</h1>
      <p className={`${baseClass}__query-description no-hover`}>{lastEditedQueryDescription}</p>
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
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
      {lastEditedQueryObserverCanRun && (
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

  const renderRunForMaintainer = (
    <form className={`${baseClass}__wrapper`}>
      <h1 className={`${baseClass}__query-name`}>{lastEditedQueryName}</h1>
      <p className={`${baseClass}__query-description`}>{lastEditedQueryDescription}</p>
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
          value={lastEditedQueryBody}
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

  const renderCreateForTeamMaintainer = (
    <>
      <form className={`${baseClass}__wrapper`}>
        <h1 className={`${baseClass}__query-name`}>New query</h1>
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={(value: string) => setLastEditedQueryBody(value)}
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

  const renderForGlobalAdminOrMaintainer = (
    <>
      <form className={`${baseClass}__wrapper`}>
        {isEditMode ? (
          <ContentEditable
            className={`${baseClass}__query-name`}
            innerRef={nameEditable}
            html={lastEditedQueryName}
            tagName="h1"
            onChange={(evt: ContentEditableEvent) =>
              setLastEditedQueryName(evt.target.value)
            }
          />
        ) : (
          <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>
        )}
        {isEditMode && (
          <ContentEditable
            className={`${baseClass}__query-description`}
            innerRef={descriptionEditable}
            html={lastEditedQueryDescription || "Add description here."}
            onChange={(evt: ContentEditableEvent) =>
              setLastEditedQueryDescription(evt.target.value)
            }
          />
        )}
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={(value: string) => {
            setLastEditedQueryBody(value)
          }}
          handleSubmit={promptSaveQuery}
        />
        {isEditMode && (
          <>
            <Checkbox
              value={lastEditedQueryObserverCanRun}
              onChange={(value: boolean) => setLastEditedQueryObserverCanRun(value)}
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
      {isSaveModalOpen && (
        <NewQueryModal
          baseClass={baseClass}
          queryValue={lastEditedQueryBody}
          onCreateQuery={onCreateQuery}
          setIsSaveModalOpen={setIsSaveModalOpen}
        />
      )}
    </>
  );

  if (isStoredQueryLoading) {
    return <Spinner />;
  }

  if (isOnlyObserver || isGlobalObserver) {
    return renderRunForObserver;
  }

  if (!isEditMode && isAnyTeamMaintainer) {
    return renderCreateForTeamMaintainer;
  }

  if (isAnyTeamMaintainer) {
    return renderRunForMaintainer;
  }

  return renderForGlobalAdminOrMaintainer;
};

export default QueryForm;