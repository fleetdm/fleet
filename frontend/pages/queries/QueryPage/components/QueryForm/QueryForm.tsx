import React, { useState, useContext, useEffect } from "react";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { size } from "lodash";
import { useDebouncedCallback } from "use-debounce/lib";

import { addGravatarUrlToResource } from "fleet/helpers";
// @ts-ignore
import { listCompatiblePlatforms, parseSqlTables } from "utilities/sql_tools";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { IQuery, IQueryFormData } from "interfaces/query";

import Avatar from "components/Avatar";
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
import QuestionIcon from "../../../../../../assets/images/icon-question-16x16@2x.png";
import PencilIcon from "../../../../../../assets/images/icon-pencil-14x14@2x.png";

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
  const [isEditingName, setIsEditingName] = useState<boolean>(false);
  const [isEditingDescription, setIsEditingDescription] = useState<boolean>(
    false
  );

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

  const debounceCompatiblePlatforms = useDebouncedCallback(
    (queryString: string) => {
      setCompatiblePlatforms(
        listCompatiblePlatforms(parseSqlTables(queryString))
      );
    },
    300
  );

  useEffect(() => {
    debounceCompatiblePlatforms(lastEditedQueryBody);

    let valid = true;
    const { valid: isValidated, errors: newErrors } = validateQuerySQL(
      lastEditedQueryBody
    );
    valid = isValidated;
    setErrors({
      ...newErrors,
    });
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

  const onChangeQuery = (sqlString: string) => {
    setLastEditedQueryBody(sqlString);
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
    const { valid: isValidated } = validateQuerySQL(lastEditedQueryBody);

    valid = isValidated;

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

  const renderAuthor = (): JSX.Element | null => {
    return storedQuery ? (
      <>
        <b>Author</b>
        <div>
          <Avatar
            user={addGravatarUrlToResource({
              email: storedQuery.author_email,
            })}
            size="xsmall"
          />
          <span>
            {storedQuery.author_name === currentUser?.name
              ? "You"
              : storedQuery.author_name}
          </span>
        </div>
      </>
    ) : null;
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
    const displayOrder = ["macOS", "Windows", "Linux"];

    const displayIncompatibilityText = () => {
      if (compatiblePlatforms[0] === "Invalid query") {
        return "No platforms (check your query for a possible syntax error)";
      } else if (compatiblePlatforms[0] === "None") {
        return "No platforms (check your query for invalid tables or tables that are supported on different platforms)";
      }
    };

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

    return (
      <span className={`${baseClass}__platform-compatibility`}>
        <b>Compatible with:</b>
        <span className={`tooltip`}>
          <span
            className={`tooltip__tooltip-icon`}
            data-tip
            data-for="query-compatibility-tooltip"
            data-tip-disable={false}
          >
            <img alt="question icon" src={QuestionIcon} />
          </span>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id="query-compatibility-tooltip"
            data-html
          >
            <span className={`tooltip__tooltip-text`}>
              Estimated compatiblity
              <br />
              based on the tables used
              <br />
              in the query
            </span>
          </ReactTooltip>
        </span>
        {displayIncompatibilityText() ||
          displayOrder.map((platform) => {
            const isCompatible =
              displayFormattedPlatforms.includes(platform) ||
              displayFormattedPlatforms[0] === "No tables in query AST"; // If query has no tables but is still syntatically valid sql, we treat it as compatible with all platforms
            return (
              <span
                key={`platform-compatibility__${platform}`}
                className="platform"
              >
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

  const renderName = () => {
    if (isEditMode) {
      if (isEditingName) {
        return (
          <InputField
            id="query-name"
            type="textarea"
            name="query-name"
            error={errors.name}
            value={lastEditedQueryName}
            placeholder="Add name here"
            inputClassName={`${baseClass}__query-name`}
            onChange={setLastEditedQueryName}
            inputOptions={{
              autoFocus: true,
              onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
                // sets cursor to end of inputfield
                const val = e.target.value;
                e.target.value = "";
                e.target.value = val;
              },
            }}
          />
        );
      }

      /* eslint-disable */
      // eslint complains about the button role
      // applied to H1 - this is needed to avoid
      // using a real button
      // prettier-ignore
      return (
        <h1
          role="button"
          className={`${baseClass}__query-name`}
          onClick={() => setIsEditingName(true)}
        >
          {lastEditedQueryName}
          <img alt="Edit name" src={PencilIcon} />
        </h1>
      );
      /* eslint-enable */
    }

    return <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>;
  };

  const renderDescription = () => {
    if (isEditMode) {
      if (isEditingDescription) {
        return (
          <InputField
            id="query-description"
            type="textarea"
            name="query-description"
            value={lastEditedQueryDescription}
            placeholder="Add description here."
            inputClassName={`${baseClass}__query-description`}
            onChange={setLastEditedQueryDescription}
            inputOptions={{
              autoFocus: true,
              onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
                // sets cursor to end of inputfield
                const val = e.target.value;
                e.target.value = "";
                e.target.value = val;
              },
            }}
          />
        );
      }

      /* eslint-disable */
      // eslint complains about the button role
      // applied to span - this is needed to avoid
      // using a real button
      // prettier-ignore
      return (
        <span
          role="button"
          className={`${baseClass}__query-description`}
          onClick={() => setIsEditingDescription(true)}
        >
          {lastEditedQueryDescription}
          <img alt="Edit description" src={PencilIcon} />
        </span>
      );
      /* eslint-enable */
    }

    return null;
  };

  const renderRunForObserver = (
    <form className={`${baseClass}__wrapper`}>
      <div className={`${baseClass}__title-bar`}>
        <div className="name-description">
          <h1 className={`${baseClass}__query-name no-hover`}>
            {lastEditedQueryName}
          </h1>
          <p className={`${baseClass}__query-description no-hover`}>
            {lastEditedQueryDescription}
          </p>
        </div>
        <div className="author">{renderAuthor()}</div>
      </div>
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
        <div className={`${baseClass}__title-bar`}>
          <div className="name-description">
            {renderName()}
            {renderDescription()}
          </div>
          <div className="author">{isEditMode && renderAuthor()}</div>
        </div>
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={onChangeQuery}
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
