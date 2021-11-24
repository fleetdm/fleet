import React, { useCallback, useContext, useEffect, useState } from "react";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { isEmpty, omit, size } from "lodash";
import { useDebouncedCallback } from "use-debounce/lib";

import { addGravatarUrlToResource } from "fleet/helpers";
import {
  listCompatiblePlatforms,
  parseSqlTables,
  OsqueryPlatform,
  ParserResult,
} from "utilities/sql_tools";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { IQuery, IQueryFormData, QueryPlatform } from "interfaces/query";

import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import NewQueryModal from "../NewQueryModal";
import CloseIcon from "../../../../../../assets/images/icon-close-vibrant-blue-16x16@2x.png";
import CompatibleIcon from "../../../../../../assets/images/icon-compatible-green-16x16@2x.png";
import IncompatibleIcon from "../../../../../../assets/images/icon-incompatible-red-16x16@2x.png";
import InfoIcon from "../../../../../../assets/images/icon-info-purple-14x14@2x.png";
import PencilIcon from "../../../../../../assets/images/icon-pencil-14x14@2x.png";
import QuestionIcon from "../../../../../../assets/images/icon-question-16x16@2x.png";

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

const PLATFORM_DISPLAY_NAMES: Record<string, string> = {
  darwin: "macOS",
  windows: "Windows",
  linux: "Linux",
};
const PLATFORM_DISPLAY_ORDER = ["macOS", "Windows", "Linux"];
const SUPPORTED_PLATFORMS = ["darwin", "windows", "linux"];

const formatParsedPlatformsForDisplay = (
  parsedPlatforms: ParserResult[]
): Array<ParserResult | string> => {
  // Map platform to display name if specified (e.g., 'darwin' becomes 'macOS'); otherwise preserve
  // the original value from the parser
  return parsedPlatforms.map(
    (string) => PLATFORM_DISPLAY_NAMES[string] || string
  );
};

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
  console.log("rendering QueryForm");
  console.log("isEditMode: ", isEditMode);

  // Note: The QueryContext values should always be used for any mutable query data such as query name
  // The storedQuery prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
    lastEditedQueryPlatform,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryPlatform,
  } = useContext(QueryContext);

  const {
    currentUser,
    isOnlyObserver,
    isGlobalObserver,
    isAnyTeamMaintainerOrTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
  } = useContext(AppContext);
  // console.log("last edited platform: ", lastEditedQueryPlatform);

  const [errors, setErrors] = useState<{ [key: string]: any }>({});
  const [isEditingName, setIsEditingName] = useState<boolean>(false);
  const [isEditingDescription, setIsEditingDescription] = useState<boolean>(
    false
  );
  const [isSaveModalOpen, setIsSaveModalOpen] = useState<boolean>(false);
  const [showQueryEditor, setShowQueryEditor] = useState<boolean>(false);

  const [isOverridePlatforms, setIsOverridePlatforms] = useState<boolean>(
    isEditMode
  );
  const [isDarwinCompatible, setIsDarwinCompatible] = useState<boolean>();
  const [isWindowsCompatible, setIsWindowsCompatible] = useState<boolean>();
  const [isLinuxCompatible, setIsLinuxCompatible] = useState<boolean>();

  const [parsedPlatforms, setParsedPlatforms] = useState<ParserResult[]>([]);

  const debounceParsePlatforms = useDebouncedCallback(
    (queryString: string) => {
      const newPlatforms = listCompatiblePlatforms(parseSqlTables(queryString));
      console.log("setting parsedPlatforms: ", newPlatforms);
      setParsedPlatforms(newPlatforms);
    },
    300,
    { leading: true }
  );

  // Watch for changes to lastEditedQueryPlatform and set checkbox values but only if they have NOT
  // already been defined. After the storedQuery is loaded, this effect should do nothing. Subsequent
  // changes to checkbox values are handled elsewhere.
  useEffect(() => {
    // TODO: Because QueryPage is not fetching the storedQuery on initial page load before setting
    // default value for the lastEditedQuery, we can't depend on lastEditedQueryPlatform
    // when this component initially loads, which causes issues on refresh. This means we have a
    // potentially circular dependencies with this effect and the effect below.
    if (isEditMode && storedQuery) {
      console.log(
        "triggered page load jank effect because lastEditedQueryPlatform changed: ",
        lastEditedQueryPlatform
      );
      const areCheckboxesUndefined =
        isDarwinCompatible ?? isWindowsCompatible ?? isLinuxCompatible ?? true;
      console.log("areCheckboxesUndefined: ", areCheckboxesUndefined);
      if (areCheckboxesUndefined) {
        console.log("setting checkbox values: ", lastEditedQueryPlatform);
        setIsWindowsCompatible(!!lastEditedQueryPlatform?.includes("windows"));
        setIsDarwinCompatible(!!lastEditedQueryPlatform?.includes("darwin"));
        setIsLinuxCompatible(!!lastEditedQueryPlatform?.includes("linux"));
      } else {
        console.log("effect does nothing");
      }
    }
  }, [lastEditedQueryPlatform]);

  // Watch for changes to override checkbox values and update lastEditedQueryPlatform but only if
  // checkboxes have already been defined (which should only be the case after storedQuery is loaded)
  useEffect(() => {
    console.log(
      `triggered checkbox change effect; new checkbox values: darwin = ${isDarwinCompatible}, windows = ${isWindowsCompatible}, linux = ${isLinuxCompatible}`
    );
    const areCheckboxesUndefined =
      isDarwinCompatible ?? isWindowsCompatible ?? isLinuxCompatible ?? true;
    console.log("areCheckboxesUndefined: ", areCheckboxesUndefined);
    if (!areCheckboxesUndefined) {
      const platforms = [];
      isDarwinCompatible && platforms.push("darwin");
      isWindowsCompatible && platforms.push("windows");
      isLinuxCompatible && platforms.push("linux");
      console.log("setting new lastEditedQueryPlatform: ", platforms.join(","));
      setLastEditedQueryPlatform(platforms.join(",") as QueryPlatform);
    } else {
      console.log("effect does nothing");
    }
  }, [isWindowsCompatible, isDarwinCompatible, isLinuxCompatible]);

  // Watch for changes in lastEditedQueryBody and update parsedPlatforms if override is not active
  useEffect(() => {
    if (!isOverridePlatforms) {
      console.log("triggered query body changed effect");
      debounceParsePlatforms(lastEditedQueryBody);
    }
  }, [lastEditedQueryBody]);

  // Watch for changes in parsedPlatforms and update lastEditedQueryPlatform if override is not active
  useEffect(() => {
    if (!isOverridePlatforms) {
      console.log("triggered parsedPlatforms changed effect");
      // Filter out any unsupported values that may have been returned by the parser (e.g., freebsd)
      const newPlatformValue = parsedPlatforms
        .filter((p) => SUPPORTED_PLATFORMS.includes(p))
        .join(",") as QueryPlatform;
      console.log("setting lastEditedQueryPlatform: ", newPlatformValue);
      setLastEditedQueryPlatform(newPlatformValue);
    }
  }, [parsedPlatforms]);

  const switchOverridePlatforms = useCallback(() => {
    const switchToOverride = !isOverridePlatforms;
    console.log(
      `switching to ${switchToOverride ? "override" : "parser"} mode`
    );

    if (switchToOverride) {
      // Set the checkbox values based on lastEditedQueryPlatform
      const platforms = lastEditedQueryPlatform?.split(",") || [];
      setIsDarwinCompatible(platforms.includes("darwin"));
      setIsWindowsCompatible(platforms.includes("windows"));
      setIsLinuxCompatible(platforms.includes("linux"));
    } else {
      // Parse the lastEditedQueryBody
      debounceParsePlatforms(lastEditedQueryBody);
    }

    // Clear any platform errors
    console.log(
      `clearing errors.platform; ${
        errors.platform
          ? `removing error: ${errors.platform}`
          : "no errors to remove"
      }`
    );
    errors.platform && setErrors(omit(errors, "platform"));

    setIsOverridePlatforms(switchToOverride);
  }, [
    errors,
    lastEditedQueryBody,
    lastEditedQueryPlatform,
    isOverridePlatforms,
  ]);

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

    const { errors: newErrors } = validateQuerySQL(lastEditedQueryBody);

    if (isEditMode && !lastEditedQueryName) {
      newErrors.name = "Query name must be present";
    }

    // Check that at least one supported platform has been selected
    const platform = lastEditedQueryPlatform
      ?.split(",")
      .filter((p) => SUPPORTED_PLATFORMS.includes(p))
      .join(",") as QueryPlatform;
    if (!platform) {
      console.log(
        "add new error: Please select a platform to save this policy"
      );
      newErrors.platform = "Please select a platform to save this policy";
      !isOverridePlatforms && switchOverridePlatforms();
    }
    const reparsedPlatforms = listCompatiblePlatforms(
      parseSqlTables(lastEditedQueryBody)
    );
    console.log(
      "platform parser is not being re-run prior to save but here is what it would return for the query being saved: ",
      reparsedPlatforms
    );
    const parserErrors: ParserResult[] = [
      "none",
      "invalid query syntax",
      "no tables in query AST",
    ];
    if (parserErrors.includes(reparsedPlatforms[0])) {
      console.log(
        "saving was allowed to continue despite parser result; isOverridePlatforms: ",
        isOverridePlatforms
      );
    }

    if (isEmpty(newErrors)) {
      if (!isEditMode || forceNew) {
        setIsSaveModalOpen(true);
      } else {
        onUpdate({
          name: lastEditedQueryName,
          description: lastEditedQueryDescription,
          query: lastEditedQueryBody,
          observer_can_run: lastEditedQueryObserverCanRun,
          platform,
        });

        setErrors({});
      }
    } else {
      setErrors(newErrors);
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

  const renderPlatforms = () => {
    // Override mode is active so compatibility is based on checkbox values
    if (isOverridePlatforms) {
      return (
        <span>
          <form>
            <Checkbox
              value={isDarwinCompatible}
              onChange={(value: boolean) => setIsDarwinCompatible(value)}
              wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
            >
              macOS
            </Checkbox>
            <Checkbox
              value={isWindowsCompatible}
              onChange={(value: boolean) => setIsWindowsCompatible(value)}
              wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
            >
              Windows
            </Checkbox>
            <Checkbox
              value={isLinuxCompatible}
              onChange={(value: boolean) => setIsLinuxCompatible(value)}
              wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
            >
              Linux
            </Checkbox>
          </form>
        </span>
      );
    }

    // Override mode is not active so compatibility is based on parsed values
    const platforms = formatParsedPlatformsForDisplay(parsedPlatforms);

    if (platforms[0] === "invalid query syntax") {
      return (
        <span className="platform">
          No platforms (check your query for a possible syntax error)
        </span>
      );
    } else if (platforms[0] === "none") {
      return (
        <span className="platform">
          No platforms (check your query for invalid tables or tables that are
          supported on different platforms)
        </span>
      );
    }

    const isCompatible = (p: string) =>
      platforms[0] === "all" || platforms.includes(p);

    return PLATFORM_DISPLAY_ORDER.map((platform) => {
      return (
        <span key={`platform-compatibility__${platform}`} className="platform">
          <img
            alt={isCompatible(platform) ? "compatible" : "incompatible"}
            src={isCompatible(platform) ? CompatibleIcon : IncompatibleIcon}
          />
          {platform}
        </span>
      );
    });
  };

  const renderPlatformCompatibilityBlock = () => {
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
              Estimated compatiblity based on the tables <br />
              used in the query. Edit the compatibility <br />
              to override the platforms this policy is <br />
              checked on.
            </span>
          </ReactTooltip>
        </span>
        {renderPlatforms()}
        <Button variant="unstyled" onClick={switchOverridePlatforms}>
          <img
            alt="edit compatible platforms"
            src={isOverridePlatforms ? CloseIcon : PencilIcon}
          />
        </Button>
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
        <div className={`${baseClass}__platform-error`}>{errors.platform}</div>
        <FleetAce
          value={lastEditedQueryBody}
          error={errors.query}
          label="Query:"
          labelActionComponent={renderLabelComponent()}
          name="query editor"
          onLoad={onLoad}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={(sqlString: string) => setLastEditedQueryBody(sqlString)}
          handleSubmit={promptSaveQuery}
        />
        {renderPlatformCompatibilityBlock()}
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
