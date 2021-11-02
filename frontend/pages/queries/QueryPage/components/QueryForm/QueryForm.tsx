import React, { useState, useRef, useContext, useEffect } from "react";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { size } from "lodash";

// @ts-ignore
import { listCompatiblePlatforms, parseSqlTables } from "utilities/sql_tools";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { IQuery, IQueryFormData } from "interfaces/query";

import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import NewQueryModal from "../NewQueryModal";
import CompatibleIcon from "../../../../../../assets/images/icon-compatible-green-16x16@2x.png";
import IncompatibleIcon from "../../../../../../assets/images/icon-incompatible-red-16x16@2x.png";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";

const baseClass = "query-form";

interface IQueryFormProps {
  queryIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedQuery: IQuery | undefined;
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
  storedQuery,
  isStoredQueryLoading,
  onCreateQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryFormProps): JSX.Element => {
  const isEditMode = !!queryIdForEdit;
  const [errors, setErrors] = useState<{ [key: string]: any }>({});
  const [isSaveModalOpen, setIsSaveModalOpen] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);
  const [compatiblePlatforms, setCompatiblePlatforms] = useState<string[]>([]);

  // Note: The QueryContext values should always be used for any mutable query data such as query name
  // The storedQuery prop should only be used to access immutable metadata such as author id
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
    currentUser,
    isOnlyObserver,
    isGlobalObserver,
    isAnyTeamMaintainerOrTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
  } = useContext(AppContext);

  useEffect(() => {
    setCompatiblePlatforms(
      listCompatiblePlatforms(parseSqlTables(lastEditedQueryBody))
    );
  }, [lastEditedQueryBody]);

  const hasTeamMaintainerPermissions = isEditMode
    ? isAnyTeamMaintainerOrTeamAdmin &&
      storedQuery &&
      currentUser &&
      storedQuery.author_id === currentUser.id
    : isAnyTeamMaintainerOrTeamAdmin;

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

    if (isEditMode && !lastEditedQueryName) {
      return setErrors({
        ...errors,
        name: "Query name must be present",
      });
    }

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

        setErrors({});
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

  const renderPlatformCompatibility = () => {
    const displayFormattedPlatforms = compatiblePlatforms.map((string) => {
      switch (string) {
        case "darwin":
          return "macOS";
        case "windows":
          return "Windows";
        case "linux":
          return "Linux";
        default:
          return string;
      }
    });
    const displayOrder = ["macOS", "Windows", "Linux"];

    return (
      <span className={`${baseClass}__platform-compatibility`}>
        <b>Compatible with:</b>
        {displayOrder.map((platform) => {
          const isCompatible = displayFormattedPlatforms.includes(platform);
          return (
            <span key={`platform-compatibility__${platform}`}>
              {platform}{" "}
              <img
                alt={isCompatible ? "compatible" : "incompatible"}
                src={isCompatible ? CompatibleIcon : IncompatibleIcon}
              />
            </span>
          );
        })}
      </span>
    );
  };

  const renderRunForObserver = (
    <form className={`${baseClass}__wrapper`}>
      <h1 className={`${baseClass}__query-name no-hover`}>
        {lastEditedQueryName}
      </h1>
      <p className={`${baseClass}__query-description no-hover`}>
        {lastEditedQueryDescription}
      </p>
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

  const renderForGlobalAdminOrAnyMaintainer = (
    <>
      <form className={`${baseClass}__wrapper`} autoComplete="off">
        {isEditMode ? (
          <InputField
            id="query-name"
            type="text"
            name="query-name"
            error={errors.name}
            value={lastEditedQueryName}
            placeholder="Add name here"
            inputClassName={`${baseClass}__query-name`}
            onChange={setLastEditedQueryName}
          />
        ) : (
          <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>
        )}
        {isEditMode && (
          <InputField
            id="query-description"
            type="text"
            name="query-description"
            value={lastEditedQueryDescription}
            placeholder="Add description here."
            inputClassName={`${baseClass}__query-description`}
            onChange={setLastEditedQueryDescription}
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
          onChange={(sqlString: string) => {
            setLastEditedQueryBody(sqlString);
          }}
          handleSubmit={promptSaveQuery}
        />
        {renderPlatformCompatibility()}
        {isEditMode && (
          <>
            <Checkbox
              value={lastEditedQueryObserverCanRun}
              onChange={(value: boolean) =>
                setLastEditedQueryObserverCanRun(value)
              }
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
          {(hasSavePermissions || isAnyTeamMaintainerOrTeamAdmin) && (
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
                    !(
                      isAnyTeamMaintainerOrTeamAdmin &&
                      !hasTeamMaintainerPermissions
                    )
                  }
                >
                  <Button
                    className={`${baseClass}__save`}
                    variant="brand"
                    onClick={promptSaveQuery()}
                    disabled={
                      isAnyTeamMaintainerOrTeamAdmin &&
                      !hasTeamMaintainerPermissions
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

  return renderForGlobalAdminOrAnyMaintainer;
};

export default QueryForm;
