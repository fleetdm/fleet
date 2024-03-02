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
import { OsqueryPlatform, SelectedPlatformString } from "interfaces/platform";
import { DEFAULT_POLICIES } from "pages/policies/constants";

import Avatar from "components/Avatar";
import FleetAce from "components/FleetAce";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
import AutoSizeInputField from "components/forms/fields/AutoSizeInputField";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import SaveNewPolicyModal from "../SaveNewPolicyModal";

const baseClass = "policy-form";

interface IPolicyFormProps {
  policyIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedPolicy: IPolicy | undefined;
  isStoredPolicyLoading: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isTeamObserver: boolean;
  isUpdatingPolicy: boolean;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onUpdate: (formData: IPolicyFormData) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
  backendValidators: { [key: string]: string };
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
  isTeamAdmin,
  isTeamMaintainer,
  isTeamObserver,
  isUpdatingPolicy,
  onCreatePolicy,
  onOsqueryTableSelect,
  goToSelectTargets,
  onUpdate,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
  backendValidators,
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
    policyTeamId,
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
    isGlobalObserver,
    isGlobalAdmin,
    isGlobalMaintainer,
    isObserverPlus,
    isOnGlobalTeam,
    isPremiumTier,
    isSandboxMode,
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

  const platformSelector = usePlatformSelector(
    lastEditedQueryPlatform,
    baseClass
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
    (isTeamAdmin && policyTeamId === storedPolicy?.team_id) || // team admin cannot save global policy
    (isTeamMaintainer && policyTeamId === storedPolicy?.team_id); // team maintainer cannot save global policy

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

  const onChangePolicy = (sqlString: string) => {
    setLastEditedQueryBody(sqlString);
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

    let selectedPlatforms: OsqueryPlatform[] = [];
    if (isEditMode || defaultPolicy) {
      selectedPlatforms = getSelectedPlatforms();
    } else {
      selectedPlatforms = getCompatiblePlatforms();
      setSelectedPlatforms(selectedPlatforms);
    }

    const newPlatformString = selectedPlatforms.join(
      ","
    ) as SelectedPlatformString;

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
      <Button variant="small-icon" onClick={onOpenSchemaSidebar}>
        <>
          <Icon name="info" size="small" />
          Show schema
        </>
      </Button>
    );
  };

  const policyNameClasses = classnames("policy-name-wrapper", {
    [`${baseClass}--editing`]: isEditingName,
  });

  const policyDescriptionClasses = classnames("policy-description-wrapper", {
    [`${baseClass}--editing`]: isEditingDescription,
  });

  const policyResolutionClasses = classnames("policy-resolution-wrapper", {
    [`${baseClass}--editing`]: isEditingResolution,
  });

  const renderName = () => {
    if (isEditMode) {
      return (
        <>
          <div className={policyNameClasses}>
            <AutoSizeInputField
              name="policy-name"
              placeholder="Add name here"
              value={lastEditedQueryName}
              hasError={errors && errors.name}
              inputClassName={`${baseClass}__policy-name`}
              maxLength="160"
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
          <div className={policyDescriptionClasses}>
            <AutoSizeInputField
              name="policy-description"
              placeholder="Add description here."
              value={lastEditedQueryDescription}
              inputClassName={`${baseClass}__policy-description`}
              maxLength="250"
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

  const renderResolution = () => {
    if (isEditMode) {
      return (
        <div className={`form-field ${baseClass}__policy-resolve`}>
          <div className="form-field__label">Resolve:</div>
          <div className={policyResolutionClasses}>
            <AutoSizeInputField
              name="policy-resolution"
              placeholder="Add resolution here."
              value={lastEditedQueryResolution}
              inputClassName={`${baseClass}__policy-resolution`}
              maxLength="500"
              onChange={setLastEditedQueryResolution}
              onFocus={() => setIsEditingResolution(true)}
              onBlur={() => setIsEditingResolution(false)}
              onKeyPress={onInputKeypress}
              isFocused={isEditingResolution}
            />
            <Button
              variant="text-icon"
              className="edit-link"
              onClick={() => setIsEditingResolution(true)}
            >
              <Icon
                name="pencil"
                className={`edit-icon ${isEditingResolution ? "hide" : ""}`}
              />
            </Button>
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
        {isSandboxMode && (
          <PremiumFeatureIconWithTooltip
            tooltipDelayHide={500}
            tooltipPositionOverrides={{ leftAdj: 84, topAdj: -4 }}
          />
        )}
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

  // Non-editable form used for Team Observers and Observer+ of their team policy and inherited policies
  // And Global Observers and Observer+ of all policies
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
        <FleetAce
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
          wrapEnabled
          readOnly
        />
      )}
      {renderLiveQueryWarning()}
      {isObserverPlus && ( // Observer+ can run existing policies
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

  // Admin or maintainer
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
          <FleetAce
            value={lastEditedQueryBody}
            error={errors.query}
            label="Query"
            labelActionComponent={renderLabelComponent()}
            name="query editor"
            onLoad={onLoad}
            wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
            onChange={onChangePolicy}
            handleSubmit={promptSavePolicy}
            wrapEnabled
            focus={!isEditMode}
          />
          <span className={`${baseClass}__platform-compatibility`}>
            {renderPlatformCompatibility()}
          </span>
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
          />
        )}
      </>
    );
  };

  if (isStoredPolicyLoading) {
    return <Spinner />;
  }

  const noEditPermissions =
    isTeamObserver ||
    isGlobalObserver ||
    (policyTeamId === 0 && !isOnGlobalTeam); // Team user viewing inherited policy

  // Render non-editable form only
  if (noEditPermissions) {
    return renderNonEditableForm;
  }

  // Render default editable form
  return renderEditableQueryForm();
};

export default PolicyForm;
