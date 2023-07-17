import React, { useState, useContext, useEffect, KeyboardEvent } from "react";
import { InjectedRouter } from "react-router";
import { size } from "lodash";
import classnames from "classnames";
import { useDebouncedCallback } from "use-debounce";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";
import { addGravatarUrlToResource } from "utilities/helpers";
import usePlatformCompatibility from "hooks/usePlatformCompatibility";
import { IApiError } from "interfaces/errors";
import { IQuery, IQueryFormData } from "interfaces/query";
import queryAPI from "services/entities/queries";

import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import NewQueryModal from "../NewQueryModal";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";

const baseClass = "query-form";

interface IQueryFormProps {
  router: InjectedRouter;
  queryIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedQuery: IQuery | undefined;
  isStoredQueryLoading: boolean;
  isQuerySaving: boolean;
  isQueryUpdating: boolean;
  onCreateQuery: (formData: IQueryFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IQueryFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
  backendValidators: { [key: string]: string };
}

const validateQuerySQL = (query: string) => {
  const errors: { [key: string]: string } = {};
  const { error: queryError, valid: queryValid } = validateQuery(query);

  if (!queryValid) {
    errors.query = queryError;
  }

  const valid = !size(errors);
  return { valid, errors };
};

const QueryForm = ({
  router,
  queryIdForEdit,
  showOpenSchemaActionText,
  storedQuery,
  isStoredQueryLoading,
  isQuerySaving,
  isQueryUpdating,
  onCreateQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
  backendValidators,
}: IQueryFormProps): JSX.Element => {
  // Note: The QueryContext values should always be used for any mutable query data such as query name
  // The storedQuery prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryId,
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
    isObserverPlus,
    isAnyTeamObserverPlus,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const savedQueryMode = !!queryIdForEdit;
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isSaveModalOpen, setIsSaveModalOpen] = useState(false);
  const [showQueryEditor, setShowQueryEditor] = useState(
    isObserverPlus || isAnyTeamObserverPlus || false
  );
  const [isEditingName, setIsEditingName] = useState(false);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [isSaveAsNewLoading, setIsSaveAsNewLoading] = useState(false);

  const platformCompatibility = usePlatformCompatibility();
  const { setCompatiblePlatforms } = platformCompatibility;

  const debounceSQL = useDebouncedCallback((sql: string) => {
    let valid = true;
    const { valid: isValidated, errors: newErrors } = validateQuerySQL(sql);
    valid = isValidated;

    setErrors({
      ...newErrors,
    });
  }, 500);

  queryIdForEdit = queryIdForEdit || 0;

  useEffect(() => {
    if (!isStoredQueryLoading && queryIdForEdit === lastEditedQueryId) {
      setCompatiblePlatforms(lastEditedQueryBody);
    }

    debounceSQL(lastEditedQueryBody);
  }, [lastEditedQueryBody, lastEditedQueryId]);

  const hasTeamMaintainerPermissions = savedQueryMode
    ? isAnyTeamMaintainerOrTeamAdmin &&
      storedQuery &&
      currentUser &&
      storedQuery.author_id === currentUser.id
    : isAnyTeamMaintainerOrTeamAdmin;

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

  const onInputKeypress = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key.toLowerCase() === "enter" && !event.shiftKey) {
      event.preventDefault();
      event.currentTarget.blur();
      setIsEditingName(false);
      setIsEditingDescription(false);
    }
  };

  const promptSaveAsNewQuery = () => (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    if (savedQueryMode && !lastEditedQueryName) {
      return setErrors({
        ...errors,
        name: "Query name must be present",
      });
    }

    let valid = true;
    const { valid: isValidated } = validateQuerySQL(lastEditedQueryBody);

    valid = isValidated;

    if (valid) {
      setIsSaveAsNewLoading(true);

      queryAPI
        .create({
          name: lastEditedQueryName,
          description: lastEditedQueryDescription,
          query: lastEditedQueryBody,
          observer_can_run: lastEditedQueryObserverCanRun,
        })
        .then((response: { query: IQuery }) => {
          setIsSaveAsNewLoading(false);
          router.push(PATHS.EDIT_QUERY(response.query));
          renderFlash("success", `Successfully added query.`);
        })
        .catch((createError: { data: IApiError }) => {
          if (createError.data.errors[0].reason.includes("already exists")) {
            queryAPI
              .create({
                name: `Copy of ${lastEditedQueryName}`,
                description: lastEditedQueryDescription,
                query: lastEditedQueryBody,
                observer_can_run: lastEditedQueryObserverCanRun,
              })
              .then((response: { query: IQuery }) => {
                setIsSaveAsNewLoading(false);
                router.push(PATHS.EDIT_QUERY(response.query));
                renderFlash(
                  "success",
                  `Successfully added query as "Copy of ${lastEditedQueryName}".`
                );
              })
              .catch((createCopyError: { data: IApiError }) => {
                if (
                  createCopyError.data.errors[0].reason.includes(
                    "already exists"
                  )
                ) {
                  renderFlash(
                    "error",
                    `"Copy of ${lastEditedQueryName}" already exists. Please rename your query and try again.`
                  );
                }
                setIsSaveAsNewLoading(false);
              });
          } else {
            setIsSaveAsNewLoading(false);
            renderFlash("error", "Could not create query. Please try again.");
          }
        });
    }
  };

  const promptSaveQuery = () => (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (savedQueryMode && !lastEditedQueryName) {
      return setErrors({
        ...errors,
        name: "Query name must be present",
      });
    }

    let valid = true;
    const { valid: isValidated } = validateQuerySQL(lastEditedQueryBody);

    valid = isValidated;

    if (valid) {
      if (!savedQueryMode) {
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
    if (isStoredQueryLoading || queryIdForEdit !== lastEditedQueryId) {
      return null;
    }

    return platformCompatibility.render();
  };

  const queryNameClasses = classnames("query-name-wrapper", {
    [`${baseClass}--editing`]: isEditingName,
  });

  const queryDescriptionClasses = classnames("query-description-wrapper", {
    [`${baseClass}--editing`]: isEditingDescription,
  });

  const renderName = () => {
    if (savedQueryMode) {
      return (
        <>
          <div className={queryNameClasses}>
            <AutoSizeInputField
              name="query-name"
              placeholder="Add name here"
              value={lastEditedQueryName}
              inputClassName={`${baseClass}__query-name`}
              maxLength="160"
              hasError={errors && errors.name}
              onChange={setLastEditedQueryName}
              onFocus={() => setIsEditingName(true)}
              onBlur={() => setIsEditingName(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingName}
            />
            <Button
              variant="text-icon"
              className="edit-link"
              onClick={() => setIsEditingName(true)}
            >
              <Icon
                name="pencil"
                className={`edit-icon ${isEditingName ? "hide" : ""}`}
              />
            </Button>
          </div>
        </>
      );
    }

    return <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>;
  };

  const renderDescription = () => {
    if (savedQueryMode) {
      return (
        <>
          <div className={queryDescriptionClasses}>
            <AutoSizeInputField
              name="query-description"
              placeholder="Add description here."
              value={lastEditedQueryDescription}
              maxLength="250"
              inputClassName={`${baseClass}__query-description`}
              onChange={setLastEditedQueryDescription}
              onFocus={() => setIsEditingDescription(true)}
              onBlur={() => setIsEditingDescription(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingDescription}
            />
            <Button
              variant="text-icon"
              className="edit-link"
              onClick={() => setIsEditingDescription(true)}
            >
              <Icon
                name="pencil"
                className={`edit-icon ${isEditingDescription ? "hide" : ""}`}
              />
            </Button>
          </div>
        </>
      );
    }
    return null;
  };

  // Observers and observer+ of existing query
  const renderNonEditableForm = (
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
      {((!isObserverPlus && isGlobalObserver) || !isAnyTeamObserverPlus) && (
        <RevealButton
          isShowing={showQueryEditor}
          className={baseClass}
          hideText="Hide SQL"
          showText="Show SQL"
          onClick={() => setShowQueryEditor(!showQueryEditor)}
        />
      )}
      {showQueryEditor && (
        <FleetAce
          value={lastEditedQueryBody}
          name="query editor"
          label="Query"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly={
            (!isObserverPlus && !isAnyTeamObserverPlus) || savedQueryMode
          }
          labelActionComponent={isObserverPlus && renderLabelComponent()}
          wrapEnabled
        />
      )}
      <span className={`${baseClass}__platform-compatibility`}>
        {renderPlatformCompatibility()}
      </span>
      {renderLiveQueryWarning()}
      {(lastEditedQueryObserverCanRun ||
        isObserverPlus ||
        isAnyTeamObserverPlus) && (
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

  const hasSavePermissions = isGlobalAdmin || isGlobalMaintainer;

  // Global admin, any maintainer, any observer+ on new query
  const renderEditableQueryForm = (
    <>
      <form className={`${baseClass}__wrapper`} autoComplete="off">
        <div className={`${baseClass}__title-bar`}>
          <div className="name-description">
            {renderName()}
            {renderDescription()}
          </div>
          <div className="author">{savedQueryMode && renderAuthor()}</div>
        </div>
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={onChangeQuery}
          handleSubmit={promptSaveQuery}
          wrapEnabled
          focus={!savedQueryMode}
        />
        <span className={`${baseClass}__platform-compatibility`}>
          {renderPlatformCompatibility()}
        </span>
        {savedQueryMode && (
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
              Users with the observer role will be able to run this query on
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
              {savedQueryMode && (
                <Button
                  variant="text-link"
                  onClick={promptSaveAsNewQuery()}
                  disabled={false}
                  className="save-as-new-loading"
                  isLoading={isSaveAsNewLoading}
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
                    className="save-loading"
                    variant="brand"
                    onClick={promptSaveQuery()}
                    disabled={
                      isAnyTeamMaintainerOrTeamAdmin &&
                      !hasTeamMaintainerPermissions
                    }
                    isLoading={isQueryUpdating}
                  >
                    Save
                  </Button>
                </div>{" "}
                <ReactTooltip
                  className={`save-query-button-tooltip`}
                  place="bottom"
                  effect="solid"
                  backgroundColor="#3e4771"
                  id="save-query-button"
                  data-html
                >
                  <>
                    You can only save
                    <br /> changes to a query if you
                    <br /> are the author.
                  </>
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
          backendValidators={backendValidators}
          isLoading={isQuerySaving}
        />
      )}
    </>
  );

  if (isStoredQueryLoading) {
    return <Spinner />;
  }

  const noEditPermissions =
    (isGlobalObserver && !isObserverPlus) || // Global observer but not Observer+
    (isObserverPlus && queryIdForEdit !== 0) || // Global observer+ on existing query
    (isOnlyObserver && !isAnyTeamObserverPlus && !isGlobalObserver) || // Only team observer but not team Observer+
    (isAnyTeamObserverPlus && // Team Observer+ on existing query
      !isAnyTeamMaintainerOrTeamAdmin &&
      queryIdForEdit !== 0);

  // Render non-editable form only
  if (noEditPermissions) {
    return renderNonEditableForm;
  }

  // Render default editable form
  return renderEditableQueryForm;
};

export default QueryForm;
