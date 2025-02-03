/* eslint-disable jsx-a11y/no-noninteractive-element-to-interactive-role */
/* eslint-disable jsx-a11y/interactive-supports-focus */
import React, { useState, useContext, useEffect, KeyboardEvent } from "react";
import { IAceEditor } from "react-ace/lib/types";
import ReactTooltip from "react-tooltip";
import { useDebouncedCallback } from "use-debounce";
import { size } from "lodash";
import classnames from "classnames";
import { COLORS } from "styles/var/colors";

import { addGravatarUrlToResource } from "utilities/helpers";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import usePlatformCompatibility from "hooks/usePlatformCompatibility";
import usePlatformSelector from "hooks/usePlatformSelector";

import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { DEFAULT_POLICIES } from "pages/policies/constants";

import Avatar from "components/Avatar";
import SQLEditor from "components/SQLEditor";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import SaveNewPolicyModal from "../SaveNewPolicyModal";

const baseClass = "policy-form";

interface IPolicyFormProps {
  policyIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedPolicy: IPolicy | undefined;
  isStoredPolicyLoading: boolean;
  isTeamObserver: boolean;
  isUpdatingPolicy: boolean;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IPolicyFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
  backendValidators: { [key: string]: string };
  isFetchingAutofillDescription: boolean;
  isFetchingAutofillResolution: boolean;
  onClickAutofillDescription: () => Promise<void>;
  onClickAutofillResolution: () => Promise<void>;
  resetAiAutofillData: () => void;
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

const PolicyForm = ({
  policyIdForEdit,
  showOpenSchemaActionText,
  storedPolicy,
  isStoredPolicyLoading,
  isTeamObserver,
  isUpdatingPolicy,
  onCreatePolicy,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
  backendValidators,
  isFetchingAutofillDescription,
  isFetchingAutofillResolution,
  onClickAutofillDescription,
  onClickAutofillResolution,
  resetAiAutofillData,
}: IPolicyFormProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isSaveNewPolicyModalOpen, setIsSaveNewPolicyModalOpen] = useState(
    false
  );
  const [showQueryEditor, setShowQueryEditor] = useState(false);
  const [isEditingName, setIsEditingName] = useState(false);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [isEditingResolution, setIsEditingResolution] = useState(false);

  // Note: The PolicyContext values should always be used for any mutable policy data such as query name
  // The storedPolicy prop should only be used to access immutable metadata such as author id
  const {
    lastEditedQueryId,
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryResolution,
    lastEditedQueryCritical,
    lastEditedQueryPlatform,
    defaultPolicy,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const {
    currentUser,
    currentTeam,
    isGlobalObserver,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isOnGlobalTeam,
    isPremiumTier,
    config,
  } = useContext(AppContext);

  const debounceSQL = useDebouncedCallback((sql: string) => {
    const { errors: newErrors } = validateQuerySQL(sql);

    setErrors({
      ...newErrors,
    });
  }, 500);

  const platformCompatibility = usePlatformCompatibility();
  const {
    getCompatiblePlatforms,
    setCompatiblePlatforms,
  } = platformCompatibility;

  const platformSelectorDisabled =
    isFetchingAutofillDescription || isFetchingAutofillResolution;

  const platformSelector = usePlatformSelector(
    lastEditedQueryPlatform,
    baseClass,
    platformSelectorDisabled,
    storedPolicy?.install_software,
    currentTeam?.id
  );

  const {
    getSelectedPlatforms,
    setSelectedPlatforms,
    isAnyPlatformSelected,
  } = platformSelector;

  policyIdForEdit = policyIdForEdit || 0;

  const isEditMode = !!policyIdForEdit && !isTeamObserver && !isGlobalObserver;

  const isNewTemplatePolicy =
    !policyIdForEdit &&
    DEFAULT_POLICIES.find((p) => p.name === lastEditedQueryName);

  const disabledLiveQuery = config?.server_settings.live_query_disabled;
  const aiFeaturesDisabled =
    config?.server_settings.ai_features_disabled || false;

  useEffect(() => {
    if (isNewTemplatePolicy) {
      setCompatiblePlatforms(lastEditedQueryBody);
    }
  }, []);

  useEffect(() => {
    debounceSQL(lastEditedQueryBody);
    if (
      (policyIdForEdit && policyIdForEdit !== lastEditedQueryId) ||
      (isNewTemplatePolicy && !lastEditedQueryBody)
    ) {
      return;
    }
    setCompatiblePlatforms(lastEditedQueryBody);
  }, [lastEditedQueryBody, lastEditedQueryId]);

  const hasSavePermissions =
    !isEditMode || // save a new policy
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isTeamMaintainerOrTeamAdmin;

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

  const onChangePolicySql = (sqlString: string) => {
    setLastEditedQueryBody(sqlString);
    resetAiAutofillData(); // Allows retry of AI autofill API if the SQL has changed
  };

  const onInputKeypress = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key.toLowerCase() === "enter" && !event.shiftKey) {
      event.preventDefault();
      event.currentTarget.blur();
      setIsEditingName(false);
      setIsEditingDescription(false);
      setIsEditingResolution(false);
    }
  };

  const promptSavePolicy = () => (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (isEditMode && !lastEditedQueryName) {
      return setErrors({
        ...errors,
        name: "Policy name must be present",
      });
    }

    if (isEditMode && !isAnyPlatformSelected) {
      return setErrors({
        ...errors,
        name: "At least one platform must be selected",
      });
    }

    let selectedPlatforms = getSelectedPlatforms();
    if (selectedPlatforms.length === 0 && !isEditMode && !defaultPolicy) {
      // If no platforms are selected, default to all compatible platforms
      selectedPlatforms = getCompatiblePlatforms();
      setSelectedPlatforms(selectedPlatforms);
    }

    const newPlatformString = selectedPlatforms.join(
      ","
    ) as CommaSeparatedPlatformString;

    if (!defaultPolicy) {
      setLastEditedQueryPlatform(newPlatformString);
    }

    if (!isEditMode) {
      setIsSaveNewPolicyModalOpen(true);
    } else {
      const payload: IPolicyFormData = {
        name: lastEditedQueryName,
        description: lastEditedQueryDescription,
        query: lastEditedQueryBody,
        resolution: lastEditedQueryResolution,
        platform: newPlatformString,
      };
      if (isPremiumTier) {
        payload.critical = lastEditedQueryCritical;
      }
      onUpdate(payload);
    }

    setIsEditingName(false);
    setIsEditingDescription(false);
    setIsEditingResolution(false);
  };

  const renderAuthor = (): JSX.Element | null => {
    return storedPolicy ? (
      <>
        <b>Author</b>
        <div>
          <Avatar
            user={addGravatarUrlToResource({
              email: storedPolicy.author_email,
            })}
            size="xsmall"
          />
          <span>
            {storedPolicy.author_name === currentUser?.name
              ? "You"
              : storedPolicy.author_name}
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
          <Icon name="info" size="small" />
          Show schema
        </>
      </Button>
    );
  };

  const editName = () => {
    if (!isEditingName) {
      setIsEditingName(true);
    }
  };

  const editDescription = () => {
    if (!isEditingDescription) {
      setIsEditingDescription(true);
    }
  };

  const editResolution = () => {
    if (!isEditingResolution) {
      setIsEditingResolution(true);
    }
  };

  const policyNameWrapperClasses = classnames("policy-name-wrapper", {
    [`${baseClass}--editing`]: isEditingName,
  });

  const policyDescriptionWrapperClasses = classnames(
    "policy-description-wrapper",
    {
      [`${baseClass}--editing`]: isEditingDescription,
    }
  );

  const policyResolutionWrapperClasses = classnames(
    "policy-resolution-wrapper",
    {
      [`${baseClass}--editing`]: isEditingResolution,
    }
  );

  const renderName = () => {
    if (isEditMode) {
      return (
        <>
          <div
            className={policyNameWrapperClasses}
            onFocus={() => setIsEditingName(true)}
            onBlur={() => setIsEditingName(false)}
            onClick={editName}
          >
            <AutoSizeInputField
              name="policy-name"
              placeholder="Add name here"
              value={lastEditedQueryName}
              hasError={errors && errors.name}
              inputClassName={`${baseClass}__policy-name ${
                !lastEditedQueryName ? "no-value" : ""
              }
              `}
              maxLength={160}
              onChange={setLastEditedQueryName}
              onKeyPress={onInputKeypress}
              isFocused={isEditingName}
            />
            <Icon
              name="pencil"
              className={`edit-icon ${isEditingName ? "hide" : ""}`}
              size="small-medium"
            />
          </div>
        </>
      );
    }

    return (
      <h1
        className={`${baseClass}__policy-name ${baseClass}__policy-name--new no-hover`}
      >
        New policy
      </h1>
    );
  };

  const renderDescription = () => {
    if (isEditMode) {
      return (
        <>
          <div
            className={policyDescriptionWrapperClasses}
            onFocus={() => setIsEditingDescription(true)}
            onBlur={() => setIsEditingDescription(false)}
            onClick={editDescription}
          >
            <AutoSizeInputField
              name="policy-description"
              placeholder="Add description here."
              value={lastEditedQueryDescription}
              inputClassName={`${baseClass}__policy-description ${
                !lastEditedQueryDescription ? "no-value" : ""
              }`}
              maxLength={250}
              onChange={setLastEditedQueryDescription}
              onKeyPress={onInputKeypress}
              isFocused={isEditingDescription}
            />
            <Icon
              name="pencil"
              className={`edit-icon ${isEditingDescription ? "hide" : ""}`}
              size="small-medium"
            />
          </div>
        </>
      );
    }

    return null;
  };

  const renderResolution = () => {
    if (isEditMode) {
      return (
        <div className={`form-field ${baseClass}__policy-resolve`}>
          <div className="form-field__label">Resolve:</div>
          <div
            className={policyResolutionWrapperClasses}
            onFocus={() => setIsEditingResolution(true)}
            onBlur={() => setIsEditingResolution(false)}
            onClick={editResolution}
          >
            <AutoSizeInputField
              name="policy-resolution"
              placeholder="Add resolution here."
              value={lastEditedQueryResolution}
              inputClassName={`${baseClass}__policy-resolution ${
                !lastEditedQueryResolution ? "no-value" : ""
              }`}
              maxLength={500}
              onChange={setLastEditedQueryResolution}
              onKeyPress={onInputKeypress}
              isFocused={isEditingResolution}
            />
            <Icon
              name="pencil"
              className={`edit-icon ${isEditingResolution ? "hide" : ""}`}
              size="small-medium"
            />
          </div>
        </div>
      );
    }

    return null;
  };

  const renderPlatformCompatibility = () => {
    if (
      isEditMode &&
      (isStoredPolicyLoading || policyIdForEdit !== lastEditedQueryId)
    ) {
      return null;
    }

    return platformCompatibility.render();
  };

  const renderCriticalPolicy = () => {
    return (
      <div className="critical-checkbox-wrapper">
        <Checkbox
          name="critical-policy"
          className="critical-policy"
          onChange={(value: boolean) => setLastEditedQueryCritical(value)}
          value={lastEditedQueryCritical}
          isLeftLabel
        >
          <TooltipWrapper
            tipContent={
              <p>
                If automations are turned on, this
                <br /> information is included.
              </p>
            }
          >
            Critical:
          </TooltipWrapper>
        </Checkbox>
      </div>
    );
  };

  // Non-editable form used for:
  // Team observers and team observer+ viewing any of their team's policies and any inherited policies
  // Team admins and team maintainers viewing any inherited policy
  // And Global observers and global observer+ viewing any team's policies and any inherited policies
  const renderNonEditableForm = (
    <form className={`${baseClass}__wrapper`}>
      <div className={`${baseClass}__title-bar`}>
        <div className="name-description-resolve">
          <h1 className={`${baseClass}__policy-name no-hover`}>
            {lastEditedQueryName}
          </h1>
          <p className={`${baseClass}__policy-description no-hover`}>
            {lastEditedQueryDescription}
          </p>
          <p className="resolve-title">
            <strong>Resolve:</strong>
          </p>
          <p className={`${baseClass}__policy-resolution no-hover`}>
            {lastEditedQueryResolution}
          </p>
        </div>
        <div className="author">{renderAuthor()}</div>
      </div>
      <RevealButton
        isShowing={showQueryEditor}
        className={baseClass}
        hideText="Hide SQL"
        showText="Show SQL"
        onClick={() => setShowQueryEditor(!showQueryEditor)}
      />
      {showQueryEditor && (
        <SQLEditor
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
          wrapEnabled
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
      {(isObserverPlus || isTeamMaintainerOrTeamAdmin) && ( // Team admin, team maintainer and any Observer+ can run a policy
        <div className="button-wrap">
          <Button
            className={`${baseClass}__run`}
            variant="blue-green"
            onClick={goToSelectTargets}
            disabled={isEditMode && !isAnyPlatformSelected}
          >
            Run
          </Button>
        </div>
      )}
    </form>
  );

  // Editable form is used for:
  // Global admins and global maintainers
  // Team admins and team maintainers viewing any of their team's policies
  const renderEditableQueryForm = () => {
    // Save disabled for no platforms selected, query name blank on existing query, or sql errors
    const disableSaveFormErrors =
      (isEditMode && !isAnyPlatformSelected) ||
      (lastEditedQueryName === "" && !!lastEditedQueryId) ||
      !!size(errors);

    return (
      <>
        <form className={`${baseClass}__wrapper`} autoComplete="off">
          <div className={`${baseClass}__title-bar`}>
            <div className="name-description-resolve">
              {renderName()}
              {renderDescription()}
              {renderResolution()}
            </div>
            <div className="author">{isEditMode && renderAuthor()}</div>
          </div>
          <SQLEditor
            value={lastEditedQueryBody}
            error={errors.query}
            label="Query"
            labelActionComponent={renderLabelComponent()}
            name="query editor"
            onLoad={onLoad}
            wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
            onChange={onChangePolicySql}
            handleSubmit={promptSavePolicy}
            wrapEnabled
            focus={!isEditMode}
          />
          {renderPlatformCompatibility()}
          {(isEditMode || defaultPolicy) && platformSelector.render()}
          {isEditMode && isPremiumTier && renderCriticalPolicy()}
          {renderLiveQueryWarning()}
          <div className="button-wrap">
            {hasSavePermissions && (
              <>
                <span
                  className={`${baseClass}__button-wrap--tooltip`}
                  data-tip
                  data-for="save-policy-button"
                  data-tip-disable={!isEditMode || isAnyPlatformSelected}
                >
                  <Button
                    variant="brand"
                    onClick={promptSavePolicy()}
                    disabled={disableSaveFormErrors}
                    className="save-loading"
                    isLoading={isUpdatingPolicy}
                  >
                    Save
                  </Button>
                </span>
                <ReactTooltip
                  className={`${baseClass}__button-wrap--tooltip`}
                  place="bottom"
                  effect="solid"
                  id="save-policy-button"
                  backgroundColor={COLORS["tooltip-bg"]}
                >
                  Select the platform(s) this
                  <br />
                  policy will be checked on
                  <br />
                  to save or run the policy.
                </ReactTooltip>
              </>
            )}
            <span
              className={`${baseClass}__button-wrap--tooltip`}
              data-tip
              data-for="run-policy-button"
              data-tip-disable={
                (!isEditMode || isAnyPlatformSelected) && !disabledLiveQuery
              }
            >
              <Button
                className={`${baseClass}__run`}
                variant="blue-green"
                onClick={goToSelectTargets}
                disabled={
                  (isEditMode && !isAnyPlatformSelected) || disabledLiveQuery
                }
              >
                Run
              </Button>
            </span>
            <ReactTooltip
              className={`${baseClass}__button-wrap--tooltip`}
              place="bottom"
              effect="solid"
              id="run-policy-button"
              backgroundColor={COLORS["tooltip-bg"]}
              data-html
            >
              {disabledLiveQuery ? (
                <>Live queries are disabled in organization settings</>
              ) : (
                <>
                  Select the platform(s) this <br />
                  policy will be checked on <br />
                  to save or run the policy.
                </>
              )}
            </ReactTooltip>
          </div>
        </form>
        {isSaveNewPolicyModalOpen && (
          <SaveNewPolicyModal
            baseClass={baseClass}
            queryValue={lastEditedQueryBody}
            onCreatePolicy={onCreatePolicy}
            setIsSaveNewPolicyModalOpen={setIsSaveNewPolicyModalOpen}
            backendValidators={backendValidators}
            platformSelector={platformSelector}
            isUpdatingPolicy={isUpdatingPolicy}
            aiFeaturesDisabled={aiFeaturesDisabled}
            isFetchingAutofillDescription={isFetchingAutofillDescription}
            isFetchingAutofillResolution={isFetchingAutofillResolution}
            onClickAutofillDescription={onClickAutofillDescription}
            onClickAutofillResolution={onClickAutofillResolution}
          />
        )}
      </>
    );
  };

  if (isStoredPolicyLoading) {
    return <Spinner />;
  }

  const isInheritedPolicy = !!policyIdForEdit && storedPolicy?.team_id === null;

  const noEditPermissions =
    isTeamObserver ||
    isGlobalObserver ||
    (!isOnGlobalTeam && isInheritedPolicy); // Team user viewing inherited policy

  // Render non-editable form only
  if (noEditPermissions) {
    return renderNonEditableForm;
  }

  // Render default editable form
  return renderEditableQueryForm();
};

export default PolicyForm;
