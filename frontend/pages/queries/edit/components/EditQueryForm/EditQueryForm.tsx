import React, {
  useState,
  useContext,
  useEffect,
  KeyboardEvent,
  useCallback,
  useMemo,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { size } from "lodash";
import classnames from "classnames";
import { useDebouncedCallback } from "use-debounce";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";

import { COLORS } from "styles/var/colors";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";

import {
  addGravatarUrlToResource,
  getCustomDropdownOptions,
  secondsToDhms,
} from "utilities/helpers";
import {
  FREQUENCY_DROPDOWN_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  INVALID_PLATFORMS_REASON,
  INVALID_PLATFORMS_FLASH_MESSAGE,
  DEFAULT_USE_QUERY_OPTIONS,
} from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import usePlatformCompatibility from "hooks/usePlatformCompatibility";
import usePlatformSelector from "hooks/usePlatformSelector";

import { getErrorReason, IApiError } from "interfaces/errors";
import {
  ISchedulableQuery,
  ICreateQueryRequestBody,
  QueryLoggingOption,
} from "interfaces/schedulable_query";
import { CommaSeparatedPlatformString } from "interfaces/platform";

import queryAPI from "services/entities/queries";
import labelsAPI, {
  getCustomLabels,
  ILabelsSummaryResponse,
} from "services/entities/labels";

import Avatar from "components/Avatar";
import SQLEditor from "components/SQLEditor";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Slider from "components/forms/fields/Slider";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import LogDestinationIndicator from "components/LogDestinationIndicator";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TargetLabelSelector from "components/TargetLabelSelector";

import SaveNewQueryModal from "../SaveNewQueryModal";
import ConfirmSaveChangesModal from "../ConfirmSaveChangesModal";
import DiscardDataOption from "../DiscardDataOption";
import SaveAsNewQueryModal from "../SaveAsNewQueryModal";

const baseClass = "edit-query-form";

interface IEditQueryFormProps {
  router: InjectedRouter;
  queryIdForEdit: number | null;
  apiTeamIdForQuery?: number;
  currentTeamId?: number;
  teamNameForQuery?: string;
  showOpenSchemaActionText: boolean;
  storedQuery: ISchedulableQuery | undefined;
  isStoredQueryLoading: boolean;
  isQuerySaving: boolean;
  isQueryUpdating: boolean;
  onSubmitNewQuery: (formData: ICreateQueryRequestBody) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  onUpdate: (formData: ICreateQueryRequestBody) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
  backendValidators: { [key: string]: string };
  hostId?: number;
  queryReportsDisabled?: boolean;
  showConfirmSaveChangesModal: boolean;
  setShowConfirmSaveChangesModal: (bool: boolean) => void;
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

const EditQueryForm = ({
  router,
  queryIdForEdit,
  apiTeamIdForQuery,
  currentTeamId,
  teamNameForQuery,
  showOpenSchemaActionText,
  storedQuery,
  isStoredQueryLoading,
  isQuerySaving,
  isQueryUpdating,
  onSubmitNewQuery,
  onOsqueryTableSelect,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
  backendValidators,
  hostId,
  queryReportsDisabled,
  showConfirmSaveChangesModal,
  setShowConfirmSaveChangesModal,
}: IEditQueryFormProps): JSX.Element => {
  // Note: The QueryContext values should always be used for any mutable query data such as query name
  // The storedQuery prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryId,
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
    lastEditedQueryFrequency,
    lastEditedQueryAutomationsEnabled,
    lastEditedQueryPlatforms,
    lastEditedQueryMinOsqueryVersion,
    lastEditedQueryLoggingType,
    lastEditedQueryDiscardData,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryFrequency,
    setLastEditedQueryAutomationsEnabled,
    setLastEditedQueryMinOsqueryVersion,
    setLastEditedQueryLoggingType,
    setLastEditedQueryDiscardData,
    setEditingExistingQuery,
  } = useContext(QueryContext);
  const {
    currentUser,
    isOnlyObserver,
    isGlobalObserver,
    isTeamMaintainerOrTeamAdmin,
    isAnyTeamMaintainerOrTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isObserverPlus,
    isAnyTeamObserverPlus,
    config,
    isPremiumTier,
  } = useContext(AppContext);

  const savedQueryMode = !!queryIdForEdit;
  const disabledLiveQuery = config?.server_settings.live_query_disabled;
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  // handles saving a copy of an existing query as a new query
  const [showSaveAsNewQueryModal, setShowSaveAsNewQueryModal] = useState(false);
  const [isSaveAsNewLoading, setIsSaveAsNewLoading] = useState(false);

  // handles saving a brand new query
  const [showSaveNewQueryModal, setShowSaveNewQueryModal] = useState(false);
  const [showQueryEditor, setShowQueryEditor] = useState(
    isObserverPlus || isAnyTeamObserverPlus || false
  );
  const [isEditingName, setIsEditingName] = useState(false);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [queryWasChanged, setQueryWasChanged] = useState(false);
  const [selectedTargetType, setSelectedTargetType] = useState("");
  const [selectedLabels, setSelectedLabels] = useState({});

  const platformSelector = usePlatformSelector(
    lastEditedQueryPlatforms,
    baseClass,
    false,
    undefined,
    undefined
  );
  const updateQueryData = {
    name: lastEditedQueryName.trim(),
    description: lastEditedQueryDescription,
    query: lastEditedQueryBody,
    platform: platformSelector
      .getSelectedPlatforms()
      .join(",") as CommaSeparatedPlatformString,
    // name should already be trimmed at this point due to associated onBlurs, but this
    // doesn't hurt
    observer_can_run: lastEditedQueryObserverCanRun,
    interval: lastEditedQueryFrequency,
    automations_enabled: lastEditedQueryAutomationsEnabled,
    min_osquery_version: lastEditedQueryMinOsqueryVersion,
    logging: lastEditedQueryLoggingType,
    discard_data: lastEditedQueryDiscardData,
    labels_include_any:
      selectedTargetType === "Custom"
        ? Object.entries(selectedLabels)
            .filter(([, selected]) => selected)
            .map(([labelName]) => labelName)
        : [],
  };

  useEffect(() => {
    setSelectedTargetType(
      storedQuery?.labels_include_any?.length && isPremiumTier
        ? "Custom"
        : "All hosts"
    );
    setSelectedLabels(
      storedQuery?.labels_include_any?.reduce((acc, label) => {
        return {
          ...acc,
          [label.name]: true,
        };
      }, {}) || {}
    );
  }, [storedQuery]);

  const {
    data: { labels } = { labels: [] },
    isFetching: isFetchingLabels,
  } = useQuery<ILabelsSummaryResponse, Error>(
    ["custom_labels"],
    () => labelsAPI.summary(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      staleTime: 10000,
      select: (res) => ({ labels: getCustomLabels(res.labels) }),
    }
  );

  const platformCompatibility = usePlatformCompatibility();
  const { setCompatiblePlatforms } = platformCompatibility;

  const debounceSQL = useDebouncedCallback((sql: string) => {
    const { errors: newErrors } = validateQuerySQL(sql);

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
  }, [lastEditedQueryBody, lastEditedQueryId, isStoredQueryLoading]);

  const toggleSaveNewQueryModal = () => {
    setShowSaveNewQueryModal(!showSaveNewQueryModal);
  };

  const toggleConfirmSaveChangesModal = () => {
    setShowConfirmSaveChangesModal(!showConfirmSaveChangesModal);
  };

  const toggleSaveAsNewQueryModal = () => {
    setShowSaveAsNewQueryModal(!showSaveAsNewQueryModal);
  };

  const onLoad = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
      enableMultiselect: false, // Disables command + click creating multiple cursors
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
    setQueryWasChanged(true);
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
  const frequencyOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        FREQUENCY_DROPDOWN_OPTIONS,
        lastEditedQueryFrequency,
        // it's safe to assume that frequency is a number
        (frequency) => `Every ${secondsToDhms(frequency as number)}`
      ),
    // intentionally leave lastEditedQueryFrequency out of the dependencies, so that the custom
    // options are maintained even if the user changes the frequency in the UI
    []
  );

  const onSelectLabel = ({
    name: labelName,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedLabels({
      ...selectedLabels,
      [labelName]: value,
    });
  };

  const onChangeSelectFrequency = useCallback(
    (value: number) => {
      setLastEditedQueryFrequency(value);
    },
    [setLastEditedQueryFrequency]
  );

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  const onChangeMinOsqueryVersionOptions = useCallback(
    (value: string) => {
      setLastEditedQueryMinOsqueryVersion(value);
    },
    [setLastEditedQueryMinOsqueryVersion]
  );

  const onChangeSelectLoggingType = useCallback(
    (value: QueryLoggingOption) => {
      setLastEditedQueryLoggingType(value);
    },
    [setLastEditedQueryLoggingType]
  );

  const handleSaveQuery = () => (evt: React.MouseEvent<HTMLButtonElement>) => {
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
        platformSelector.setSelectedPlatforms(
          platformCompatibility.getCompatiblePlatforms()
        );
        setShowSaveNewQueryModal(true);
      } else {
        //
        onUpdate(updateQueryData);
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
          Schema
          <Icon name="info" size="small" />
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

  const editName = () => {
    if (!isEditingName) {
      setIsEditingName(true);
    }
  };

  const queryNameWrapperClass = "query-name-wrapper";
  const queryNameWrapperClasses = classnames(queryNameWrapperClass, {
    [`${baseClass}--editing`]: isEditingName,
  });

  const queryDescriptionWrapperClass = "query-description-wrapper";
  const queryDescriptionWrapperClasses = classnames(
    queryDescriptionWrapperClass,
    {
      [`${baseClass}--editing`]: isEditingDescription,
    }
  );

  const renderName = () => {
    if (savedQueryMode) {
      return (
        <GitOpsModeTooltipWrapper
          position="right"
          tipOffset={16}
          renderChildren={(disableChildren) => {
            const classes = classnames(queryNameWrapperClasses, {
              [`${queryNameWrapperClass}--disabled-by-gitops-mode`]: disableChildren,
            });
            return (
              <div
                className={classes}
                onFocus={() => setIsEditingName(true)}
                onBlur={() => setIsEditingName(false)}
                onClick={editName}
              >
                <AutoSizeInputField
                  name="query-name"
                  placeholder="Add name"
                  value={lastEditedQueryName}
                  inputClassName={`${baseClass}__query-name ${
                    !lastEditedQueryName ? "no-value" : ""
                  }`}
                  maxLength={160}
                  hasError={errors && errors.name}
                  onChange={setLastEditedQueryName}
                  onBlur={() => {
                    setLastEditedQueryName(lastEditedQueryName.trim());
                  }}
                  onKeyPress={onInputKeypress}
                  isFocused={isEditingName}
                  disableTabability={disableChildren}
                />
                <Icon
                  name="pencil"
                  className={`edit-icon ${isEditingName ? "hide" : ""}`}
                  size="small-medium"
                />
              </div>
            );
          }}
        />
      );
    }

    return <h1 className={`${baseClass}__query-name no-hover`}>New query</h1>;
  };

  const editDescription = () => {
    if (!isEditingDescription) {
      setIsEditingDescription(true);
    }
  };

  const renderDescription = () => {
    if (savedQueryMode) {
      return (
        <GitOpsModeTooltipWrapper
          position="right"
          tipOffset={16}
          renderChildren={(disableChildren) => {
            const classes = classnames(queryDescriptionWrapperClasses, {
              [`${queryDescriptionWrapperClass}--disabled-by-gitops-mode`]: disableChildren,
            });
            return (
              <div
                className={classes}
                onFocus={() => setIsEditingDescription(true)}
                onBlur={() => setIsEditingDescription(false)}
                onClick={editDescription}
              >
                <AutoSizeInputField
                  name="query-description"
                  placeholder="Add description"
                  value={lastEditedQueryDescription}
                  maxLength={250}
                  inputClassName={`${baseClass}__query-description ${
                    !lastEditedQueryDescription ? "no-value" : ""
                  }`}
                  onChange={setLastEditedQueryDescription}
                  onKeyPress={onInputKeypress}
                  isFocused={isEditingDescription}
                  disableTabability={disableChildren}
                />
                <Icon
                  name="pencil"
                  className={`edit-icon ${isEditingDescription ? "hide" : ""}`}
                  size="small-medium"
                />
              </div>
            );
          }}
        />
      );
    }
    return null;
  };

  // Observers and observer+ of existing query
  const renderNonEditableForm = (
    <form className={`${baseClass}`}>
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
        <SQLEditor
          value={lastEditedQueryBody}
          name="query editor"
          label="Query"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          readOnly={
            (!isObserverPlus && !isAnyTeamObserverPlus) || savedQueryMode
          }
          labelActionComponent={isObserverPlus && renderLabelComponent()}
          wrapEnabled
          data-testid="ace-editor"
        />
      )}
      {renderPlatformCompatibility()}
      {renderLiveQueryWarning()}
      {(lastEditedQueryObserverCanRun ||
        isObserverPlus ||
        isAnyTeamObserverPlus) && (
        <div className={`button-wrap ${baseClass}__button-wrap--new-query`}>
          <div
            data-tip
            data-for="live-query-button"
            // Tooltip shows when live queries are globally disabled
            data-tip-disable={!disabledLiveQuery}
          >
            <Button
              className={`${baseClass}__run`}
              variant="success"
              onClick={() => {
                router.push(
                  getPathWithQueryParams(PATHS.LIVE_QUERY(queryIdForEdit), {
                    host_id: hostId,
                    team_id: apiTeamIdForQuery,
                  })
                );
              }}
              disabled={disabledLiveQuery}
            >
              Live query
            </Button>
          </div>
          <ReactTooltip
            className={`live-query-button-tooltip`}
            place="top"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id="live-query-button"
            data-html
          >
            Live queries are disabled in organization settings
          </ReactTooltip>
        </div>
      )}
    </form>
  );

  const hasSavePermissions =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainerOrTeamAdmin;

  const currentlySavingQueryResults =
    storedQuery &&
    !storedQuery.discard_data &&
    !["differential", "differential_ignore_removals"].includes(
      storedQuery.logging
    );
  const changedSQL = storedQuery && lastEditedQueryBody !== storedQuery.query;
  const changedLoggingToDifferential = [
    "differential",
    "differential_ignore_removals",
  ].includes(lastEditedQueryLoggingType);

  // Note: The backend is not resetting the query reports with equivalent platform strings
  // so we are not showing a warning unless the platform combinations differ
  const formatPlatformEquivalences = (platforms?: string) => {
    // Remove white spaces allowed by API and format into a sorted string converted from a sorted array
    return platforms?.replace(/\s/g, "").split(",").sort().toString();
  };

  const changedPlatforms =
    storedQuery &&
    formatPlatformEquivalences(lastEditedQueryPlatforms) !==
      formatPlatformEquivalences(storedQuery?.platform);

  const changedMinOsqueryVersion =
    storedQuery &&
    lastEditedQueryMinOsqueryVersion !== storedQuery.min_osquery_version;

  const enabledDiscardData =
    storedQuery && lastEditedQueryDiscardData && !storedQuery.discard_data;

  const confirmChanges =
    currentlySavingQueryResults &&
    (changedSQL ||
      changedLoggingToDifferential ||
      enabledDiscardData ||
      changedPlatforms ||
      changedMinOsqueryVersion);

  const showChangedSQLCopy =
    changedSQL && !changedLoggingToDifferential && !enabledDiscardData;

  // Global admin, any maintainer, any observer+ on new query
  const renderEditableQueryForm = () => {
    // Save and save as new disabled for query name blank on existing query or sql errors
    const disableSaveFormErrors =
      (lastEditedQueryName === "" && !!lastEditedQueryId) ||
      !!size(errors) ||
      (savedQueryMode && !platformSelector.isAnyPlatformSelected) ||
      (selectedTargetType === "Custom" &&
        !Object.entries(selectedLabels).some(([, value]) => {
          return value;
        }));

    return (
      <>
        <form className={`${baseClass}`} autoComplete="off">
          <div className={`${baseClass}__title-bar`}>
            <div className="name-description">
              {renderName()}
              {renderDescription()}
            </div>
            <div className="author">{savedQueryMode && renderAuthor()}</div>
          </div>
          <SQLEditor
            value={lastEditedQueryBody}
            error={errors.query}
            label="Query"
            labelActionComponent={renderLabelComponent()}
            name="query editor"
            onLoad={onLoad}
            wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
            onChange={onChangeQuery}
            handleSubmit={
              confirmChanges ? toggleConfirmSaveChangesModal : handleSaveQuery
            }
            wrapEnabled
            focus={!savedQueryMode}
          />
          {renderPlatformCompatibility()}

          {savedQueryMode && (
            <div
              // including `form` class here keeps the children fields subject to the global form
              // children styles
              className={
                gitOpsModeEnabled ? "disabled-by-gitops-mode form" : "form"
              }
            >
              <Dropdown
                searchable={false}
                options={frequencyOptions}
                onChange={onChangeSelectFrequency}
                placeholder="Every day"
                value={lastEditedQueryFrequency}
                label="Interval"
                wrapperClassName={`${baseClass}__form-field form-field--frequency`}
                helpText="This is how often your query collects data."
              />
              <Slider
                onChange={() =>
                  setLastEditedQueryAutomationsEnabled(
                    !lastEditedQueryAutomationsEnabled
                  )
                }
                value={lastEditedQueryAutomationsEnabled}
                activeText={
                  <>
                    Automations on
                    {lastEditedQueryFrequency === 0 && (
                      <TooltipWrapper
                        tipContent={
                          <>
                            Automations and reporting will be paused <br />
                            for this query until an interval is set.
                          </>
                        }
                        position="right"
                        tipOffset={9}
                        showArrow
                        underline={false}
                      >
                        <Icon name="warning" />
                      </TooltipWrapper>
                    )}
                  </>
                }
                inactiveText="Automations off"
                helpText={
                  <>
                    Historical results will
                    {!lastEditedQueryAutomationsEnabled ? " not " : " "}be sent
                    to your log destination:{" "}
                    <b>
                      <LogDestinationIndicator
                        logDestination={config?.logging.result.plugin || ""}
                        filesystemDestination={
                          config?.logging.result.config?.result_log_file
                        }
                        excludeTooltip
                      />
                    </b>
                    .
                  </>
                }
              />
              <Checkbox
                value={lastEditedQueryObserverCanRun}
                onChange={(value: boolean) =>
                  setLastEditedQueryObserverCanRun(value)
                }
                helpText="Users with the observer role will be able to run this query on hosts where they have access."
              >
                Observers can run
              </Checkbox>
              {savedQueryMode && platformSelector.render()}
              {isPremiumTier && (
                <TargetLabelSelector
                  selectedTargetType={selectedTargetType}
                  selectedLabels={selectedLabels}
                  className={`${baseClass}__target`}
                  onSelectTargetType={setSelectedTargetType}
                  onSelectLabel={onSelectLabel}
                  labels={labels || []}
                  customHelpText={
                    <span className="form-field__help-text">
                      Query will target hosts that <b>have any</b> of these
                      labels:
                    </span>
                  }
                  suppressTitle
                />
              )}
              <RevealButton
                isShowing={showAdvancedOptions}
                className="advanced-options-toggle"
                hideText="Hide advanced options"
                showText="Show advanced options"
                caretPosition="after"
                onClick={toggleAdvancedOptions}
              />
              {showAdvancedOptions && (
                <>
                  <Dropdown
                    options={MIN_OSQUERY_VERSION_OPTIONS}
                    onChange={onChangeMinOsqueryVersionOptions}
                    placeholder="Select"
                    value={lastEditedQueryMinOsqueryVersion}
                    label="Minimum osquery version"
                    wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
                  />
                  <Dropdown
                    options={LOGGING_TYPE_OPTIONS}
                    onChange={onChangeSelectLoggingType}
                    placeholder="Select"
                    value={lastEditedQueryLoggingType}
                    label="Logging"
                    wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
                  />
                  {queryReportsDisabled !== undefined && (
                    <DiscardDataOption
                      selectedLoggingType={lastEditedQueryLoggingType}
                      discardData={lastEditedQueryDiscardData}
                      setDiscardData={setLastEditedQueryDiscardData}
                      queryReportsDisabled={queryReportsDisabled}
                    />
                  )}
                </>
              )}
            </div>
          )}
          {renderLiveQueryWarning()}
          <div className={`button-wrap ${baseClass}__button-wrap--new-query`}>
            {hasSavePermissions && (
              <>
                {savedQueryMode && (
                  <GitOpsModeTooltipWrapper
                    renderChildren={(disableChildren) => (
                      <Button
                        variant="text-link"
                        onClick={toggleSaveAsNewQueryModal}
                        disabled={disableSaveFormErrors || disableChildren}
                      >
                        Save as new
                      </Button>
                    )}
                  />
                )}
                <div className={`${baseClass}__button-wrap--save-query-button`}>
                  <GitOpsModeTooltipWrapper
                    tipOffset={8}
                    renderChildren={(disableChildren) => (
                      <Button
                        className="save-loading"
                        onClick={
                          confirmChanges
                            ? toggleConfirmSaveChangesModal
                            : handleSaveQuery()
                        }
                        disabled={disableSaveFormErrors || disableChildren}
                        isLoading={isQueryUpdating}
                      >
                        Save
                      </Button>
                    )}
                  />
                </div>
              </>
            )}
            <div
              data-tip
              data-for="live-query-button"
              // Tooltip shows when live queries are globally disabled
              data-tip-disable={!disabledLiveQuery}
            >
              <Button
                className={`${baseClass}__run`}
                variant="success"
                onClick={() => {
                  // calling `setEditingExistingQuery` here prevents
                  // inclusion of `query_id` in the subsequent `run` API call, which prevents counting
                  // this live run in performance impact. Since we DO want to count this run in those
                  // stats if the query is the same as the saved one, only set below IF the query
                  // has been changed.
                  // TODO - product: should host details > action > query > <select existing query>
                  // go to the host details page instead of the edit query page, where the user has
                  // the choice to edit the query or run it live directly?
                  if (queryWasChanged) {
                    setEditingExistingQuery(true); // Persists edited query data through live query flow
                  }
                  router.push(
                    getPathWithQueryParams(PATHS.LIVE_QUERY(queryIdForEdit), {
                      host_id: hostId,
                      team_id: currentTeamId,
                    })
                  );
                }}
                disabled={disabledLiveQuery}
              >
                Live query
              </Button>
            </div>
            <ReactTooltip
              className={`live-query-button-tooltip`}
              place="top"
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id="live-query-button"
              data-html
            >
              Live queries are disabled in organization settings
            </ReactTooltip>
          </div>
        </form>
        {showSaveNewQueryModal && (
          <SaveNewQueryModal
            queryValue={lastEditedQueryBody}
            apiTeamIdForQuery={apiTeamIdForQuery}
            saveQuery={onSubmitNewQuery}
            toggleSaveNewQueryModal={toggleSaveNewQueryModal}
            backendValidators={backendValidators}
            isLoading={isQuerySaving}
            queryReportsDisabled={queryReportsDisabled}
            platformSelector={platformSelector}
          />
        )}
        {showSaveAsNewQueryModal && (
          <SaveAsNewQueryModal
            router={router}
            initialQueryData={{
              ...updateQueryData,
              team_id: apiTeamIdForQuery,
            }}
          />
        )}
        {showConfirmSaveChangesModal && (
          <ConfirmSaveChangesModal
            onSaveChanges={handleSaveQuery()}
            isUpdating={isQueryUpdating}
            onClose={toggleConfirmSaveChangesModal}
            showChangedSQLCopy={showChangedSQLCopy}
          />
        )}
      </>
    );
  };

  if (isStoredQueryLoading || isFetchingLabels) {
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
  return renderEditableQueryForm();
};

export default EditQueryForm;
