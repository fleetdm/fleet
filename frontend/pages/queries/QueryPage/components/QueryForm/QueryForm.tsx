import React, { useState, useContext, useEffect, KeyboardEvent } from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import PATHS from "router/paths";

import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { size } from "lodash";
import { useDebouncedCallback } from "use-debounce/lib";
import classnames from "classnames";

import { addGravatarUrlToResource } from "fleet/helpers";
// @ts-ignore
import { listCompatiblePlatforms, parseSqlTables } from "utilities/sql_tools";

import queryAPI from "services/entities/queries";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { IQuery, IQueryFormData } from "interfaces/query";
import { IApiError } from "interfaces/errors";

import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner";
// @ts-ignore
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import NewQueryModal from "../NewQueryModal";
import PlatformCompatibility from "../PlatformCompatibility";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";
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
  backendValidators,
}: IQueryFormProps): JSX.Element => {
  const dispatch = useDispatch();

  const isEditMode = !!queryIdForEdit;
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isSaveModalOpen, setIsSaveModalOpen] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);
  const [compatiblePlatforms, setCompatiblePlatforms] = useState<string[]>([]);
  const [isEditingName, setIsEditingName] = useState<boolean>(false);
  const [isEditingDescription, setIsEditingDescription] = useState<boolean>(
    false
  );
  const [isSaveAsNewLoading, setIsSaveAsNewLoading] = useState<boolean>(false);

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
  } = useContext(AppContext);

  const debounceCompatiblePlatforms = useDebouncedCallback(
    (queryString: string) => {
      setCompatiblePlatforms(
        listCompatiblePlatforms(parseSqlTables(queryString))
      );
    },
    300,
    { leading: true }
  );

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
    if (queryIdForEdit === lastEditedQueryId) {
      debounceCompatiblePlatforms(lastEditedQueryBody);
    }

    debounceSQL(lastEditedQueryBody);
  }, [lastEditedQueryBody, lastEditedQueryId]);

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
          dispatch(push(PATHS.EDIT_QUERY(response.query)));
          dispatch(renderFlash("success", `Successfully added query.`));
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
                dispatch(push(PATHS.EDIT_QUERY(response.query)));
                dispatch(
                  renderFlash(
                    "success",
                    `Successfully added query as "Copy of ${lastEditedQueryName}".`
                  )
                );
              })
              .catch((createCopyError: { data: IApiError }) => {
                if (
                  createCopyError.data.errors[0].reason.includes(
                    "already exists"
                  )
                ) {
                  dispatch(
                    renderFlash(
                      "error",
                      `"Copy of ${lastEditedQueryName}" already exists. Please rename your query and try again.`
                    )
                  );
                }
                setIsSaveAsNewLoading(false);
              });
          } else {
            setIsSaveAsNewLoading(false);
            dispatch(
              renderFlash("error", "Could not create query. Please try again.")
            );
          }
        });
    }
  };

  const promptSaveQuery = () => (evt: React.MouseEvent<HTMLButtonElement>) => {
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
      if (!isEditMode) {
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
    if (
      isStoredQueryLoading ||
      queryIdForEdit !== lastEditedQueryId ||
      !compatiblePlatforms.length
    ) {
      return null;
    }

    return <PlatformCompatibility compatiblePlatforms={compatiblePlatforms} />;
  };

  const queryNameClasses = classnames("query-name-wrapper", {
    [`${baseClass}--editing`]: isEditingName,
  });

  const queryDescriptionClasses = classnames("query-description-wrapper", {
    [`${baseClass}--editing`]: isEditingDescription,
  });

  const renderName = () => {
    if (isEditMode) {
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
            <a className="edit-link" onClick={() => setIsEditingName(true)}>
              <img
                className={`edit-icon ${isEditingName && "hide"}`}
                alt="Edit name"
                src={PencilIcon}
              />
            </a>
          </div>
        </>
      );
    }

    return <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>;
  };

  const renderDescription = () => {
    if (isEditMode) {
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
            <a
              className="edit-link"
              onClick={() => setIsEditingDescription(true)}
            >
              <img
                className={`edit-icon ${isEditingDescription && "hide"}`}
                alt="Edit name"
                src={PencilIcon}
              />
            </a>
          </div>
        </>
      );
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
      <RevealButton
        isShowing={showQueryEditor}
        baseClass={baseClass}
        hideText="Hide SQL"
        showText="Show SQL"
        onClick={() => setShowQueryEditor(!showQueryEditor)}
      />
      {showQueryEditor && (
        <FleetAce
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly
        />
      )}
      <span className={`${baseClass}__platform-compatibility`}>
        {renderPlatformCompatibility()}
      </span>
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
        {isSaveAsNewLoading && (
          <div className={`${baseClass}__loading-overlay`}>
            <Spinner />
          </div>
        )}
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
        <span className={`${baseClass}__platform-compatibility`}>
          {renderPlatformCompatibility()}
        </span>
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
                  className={`${baseClass}__save-as-new`}
                  variant="text-link"
                  onClick={promptSaveAsNewQuery()}
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
          backendValidators={backendValidators}
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
