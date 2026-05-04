/* eslint-disable jsx-a11y/no-noninteractive-element-to-interactive-role */
/* eslint-disable jsx-a11y/interactive-supports-focus */
import React, { useState, useContext, useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "react-query";

import { Ace } from "ace-builds";
import { useDebouncedCallback } from "use-debounce";
import { size } from "lodash";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { PolicyContext } from "context/policy";
import usePlatformCompatibility from "hooks/usePlatformCompatibility";
import usePlatformSelector from "hooks/usePlatformSelector";
import PATHS from "router/paths";
import {
  getCustomTargetOptions,
  LabelScope,
} from "components/TargetLabelSelector/labelScopes";
import { getPathWithQueryParams } from "utilities/url";

import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { DEFAULT_POLICIES } from "pages/policies/constants";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import SQLEditor from "components/SQLEditor";
import {
  validateQuery,
  EMPTY_QUERY_ERR,
} from "components/forms/validators/validate_query";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import CustomLink from "components/CustomLink";
import TargetLabelSelector from "components/TargetLabelSelector";

import labelsAPI, {
  getCustomLabels,
  ILabelsSummaryResponse,
} from "services/entities/labels";

import teamPoliciesAPI from "services/entities/team_policies";

import SaveNewPolicyModal from "../SaveNewPolicyModal";
import PolicyAutomations from "../PolicyAutomations";

const baseClass = "policy-form";

interface IPolicyFormProps {
  router: InjectedRouter;
  teamIdForApi?: number;
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
  currentAutomatedPolicies: number[];
  onCancel?: () => void;
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
  router,
  teamIdForApi,
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
  currentAutomatedPolicies,
  onCancel,
}: IPolicyFormProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isSaveNewPolicyModalOpen, setIsSaveNewPolicyModalOpen] = useState(
    false
  );
  const [showQueryEditor, setShowQueryEditor] = useState(false);

  const [selectedTargetType, setSelectedTargetType] = useState("All hosts");
  const [selectedCustomTarget, setSelectedCustomTarget] = useState<LabelScope>(
    "labelsIncludeAny"
  );
  const [selectedLabels, setSelectedLabels] = useState({});

  const isPatchPolicy = storedPolicy?.type === "patch";
  const [isAddingAutomation, setIsAddingAutomation] = useState(false);

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
    lastEditedQueryLabelsIncludeAny,
    lastEditedQueryLabelsIncludeAll,
    lastEditedQueryLabelsExcludeAny,
    defaultPolicy,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

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

  const { renderFlash } = useContext(NotificationContext);
  const queryClient = useQueryClient();

  const {
    currentUser,
    currentTeam,
    isGlobalObserver,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isTeamTechnician,
    isGlobalTechnician,
    isOnGlobalTeam,
    isPremiumTier,
    config,
    isFreeTier,
  } = useContext(AppContext);

  const customTargetOptions = useMemo(
    () => getCustomTargetOptions({ entity: "policy", isPremiumTier }),
    [isPremiumTier]
  );

  const { data: { labels } = { labels: [] } } = useQuery<
    ILabelsSummaryResponse,
    Error
  >(
    ["custom_labels", currentTeam],
    () => labelsAPI.summary(currentTeam?.id, true),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // Wait for the current team to load from context before pulling labels, otherwise on a page load
      // directly on the policies new/edit page this gets called with currentTeam not set, then again
      // with the correct team value. If we don't trigger on currentTeam changes we'll just start with a
      // null team ID here and never populate with the correct team unless we navigate from another page
      // where team context is already set prior to navigation.
      enabled: isPremiumTier && !!currentTeam,
      staleTime: 10000,
      select: (res) => ({ labels: getCustomLabels(res.labels) }),
    }
  );

  const disabledLiveQuery = config?.server_settings.live_query_disabled;
  const aiFeaturesDisabled =
    config?.server_settings.ai_features_disabled || false;
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

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
    isFetchingAutofillDescription ||
    isFetchingAutofillResolution ||
    gitOpsModeEnabled;

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

  /* - Observer/Observer+ and Technicians cannot edit existing policies
     - Team users cannot edit inherited policies
    Reroute edit existing policy page (/:policyId/edit) to policy details page (/:policyId) */
  useEffect(() => {
    const isInheritedPolicy = isEditMode && storedPolicy?.team_id === null;

    const noEditPermissions =
      isTeamObserver ||
      isGlobalObserver ||
      isTeamTechnician ||
      isGlobalTechnician ||
      (!isOnGlobalTeam && isInheritedPolicy); // Team user viewing inherited policy

    if (
      !isStoredPolicyLoading && // Confirms teamId for storedQuery before RBAC reroute
      policyIdForEdit &&
      policyIdForEdit > 0 &&
      noEditPermissions
    ) {
      router.push(
        getPathWithQueryParams(PATHS.POLICY_DETAILS(policyIdForEdit), {
          fleet_id: teamIdForApi,
        })
      );
    }
  }, [
    policyIdForEdit,
    isEditMode,
    isStoredPolicyLoading,
    isTeamObserver,
    isGlobalObserver,
    isTeamTechnician,
    isGlobalTechnician,
    isOnGlobalTeam,
    storedPolicy?.team_id,
    router,
    teamIdForApi,
  ]);

  useEffect(() => {
    setSelectedTargetType(
      !lastEditedQueryLabelsIncludeAny.length &&
        !lastEditedQueryLabelsIncludeAll.length &&
        !lastEditedQueryLabelsExcludeAny.length
        ? "All hosts"
        : "Custom"
    );

    let customTarget: LabelScope | undefined;
    let activeLabels: typeof lastEditedQueryLabelsIncludeAny = [];
    if (lastEditedQueryLabelsExcludeAny.length) {
      customTarget = "labelsExcludeAny";
      activeLabels = lastEditedQueryLabelsExcludeAny;
    } else if (lastEditedQueryLabelsIncludeAll.length) {
      customTarget = "labelsIncludeAll";
      activeLabels = lastEditedQueryLabelsIncludeAll;
    } else if (lastEditedQueryLabelsIncludeAny.length) {
      customTarget = "labelsIncludeAny";
      activeLabels = lastEditedQueryLabelsIncludeAny;
    }
    if (customTarget) {
      setSelectedCustomTarget(customTarget);
    }

    setSelectedLabels(
      activeLabels.reduce((acc, label) => {
        return {
          ...acc,
          [label.name]: true,
        };
      }, {})
    );
  }, [
    lastEditedQueryLabelsIncludeAny,
    lastEditedQueryLabelsIncludeAll,
    lastEditedQueryLabelsExcludeAny,
  ]);

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

  const onLoad = (editor: Ace.Editor) => {
    editor.setOptions({
      enableLinking: true,
      enableMultiselect: false, // Disables command + click creating multiple cursors
    } as any);

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

  const onAddPatchAutomation = async () => {
    if (
      !storedPolicy?.patch_software?.software_title_id ||
      !storedPolicy?.team_id
    ) {
      return;
    }
    setIsAddingAutomation(true);
    try {
      await teamPoliciesAPI.update(policyIdForEdit as number, {
        team_id: storedPolicy.team_id,
        software_title_id: storedPolicy.patch_software.software_title_id,
      });
      queryClient.invalidateQueries(["policy", policyIdForEdit]);
      renderFlash("success", "Automation added.");
    } catch {
      renderFlash("error", "Couldn't set automation. Please try again.");
    } finally {
      setIsAddingAutomation(false);
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

    if (isEditMode && !isPatchPolicy && !isAnyPlatformSelected) {
      return setErrors({
        ...errors,
        name: "At least one platform must be selected",
      });
    }

    if (isPatchPolicy && isEditMode) {
      // Patch policies: only send editable fields, not query/platform
      const payload: IPolicyFormData = {
        name: lastEditedQueryName,
        description: lastEditedQueryDescription,
        resolution: lastEditedQueryResolution,
      };
      if (isPremiumTier) {
        payload.critical = lastEditedQueryCritical;
      }
      onUpdate(payload);
      return;
    }

    // Synchronously block empty queries. The button-level `disableSaveFormErrors`
    // relies on the debounced `errors.query`, so a fast click before the debounce
    // fires could otherwise submit an empty query. Mirrors the guard in
    // EditQueryForm's handleSaveQuery (see #38348).
    if (!lastEditedQueryBody?.trim()) {
      return setErrors({
        ...errors,
        query: EMPTY_QUERY_ERR,
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
        const customLabelNames =
          selectedTargetType === "Custom"
            ? Object.entries(selectedLabels)
                .filter(([, selected]) => selected)
                .map(([labelName]) => labelName)
            : [];
        payload.labels_include_any =
          selectedCustomTarget === "labelsIncludeAny" ? customLabelNames : [];
        payload.labels_include_all =
          selectedCustomTarget === "labelsIncludeAll" ? customLabelNames : [];
        payload.labels_exclude_any =
          selectedCustomTarget === "labelsExcludeAny" ? customLabelNames : [];
        payload.critical = lastEditedQueryCritical;
      }
      onUpdate(payload);
    }
  };

  const renderLabelComponent = (): JSX.Element | null => {
    return (
      <div className={`${baseClass}__sql-editor-label-actions`}>
        {showOpenSchemaActionText && (
          <Button variant="inverse" onClick={onOpenSchemaSidebar}>
            <>
              Schema
              <Icon name="info" />
            </>
          </Button>
        )}
        {!policyIdForEdit && (
          // only when creating a new policy
          <CustomLink
            text="Examples"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/policy-templates`}
            newTab
          />
        )}
      </div>
    );
  };

  const renderName = () => {
    if (isEditMode) {
      return (
        <InputField
          name="policy-name"
          label="Name"
          placeholder="Add name here"
          value={lastEditedQueryName}
          error={errors && errors.name}
          onChange={(value: string) => setLastEditedQueryName(value)}
          disabled={gitOpsModeEnabled}
        />
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
        <InputField
          name="policy-description"
          label="Description"
          placeholder="Add description here."
          value={lastEditedQueryDescription}
          type="textarea"
          helpText="How does this policy's failure put the organization at risk?"
          onChange={(value: string) => setLastEditedQueryDescription(value)}
          disabled={gitOpsModeEnabled}
        />
      );
    }

    return null;
  };

  const renderResolution = () => {
    if (isEditMode) {
      return (
        <InputField
          name="policy-resolution"
          label="Resolution"
          placeholder="Add resolution here."
          value={lastEditedQueryResolution}
          type="textarea"
          helpText="If this policy fails, what should the end user expect?"
          onChange={(value: string) => setLastEditedQueryResolution(value)}
          disabled={gitOpsModeEnabled}
        />
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
      <div className={`${baseClass}__critical-checkbox-wrapper`}>
        <Checkbox
          name="critical-policy"
          className="critical-policy"
          onChange={(value: boolean) => setLastEditedQueryCritical(value)}
          value={lastEditedQueryCritical}
          isLeftLabel
          disabled={gitOpsModeEnabled}
        >
          <TooltipWrapper
            tipContent={
              <p>
                If automations are turned on, this information is included. If
                Okta conditional access is configured, end users can never
                bypass critical policies.
              </p>
            }
          >
            Critical
          </TooltipWrapper>
        </Checkbox>
      </div>
    );
  };

  const renderPolicyFleetName = () => {
    if (isFreeTier || !currentTeam?.name) return null;

    return isEditMode ? (
      <p>
        Editing policy for <strong>{currentTeam?.name}</strong>.
      </p>
    ) : (
      <p>
        Creating a new policy for <strong>{currentTeam?.name}</strong>.
      </p>
    );
  };

  const renderPolicyForm = () => {
    // Save disabled for no platforms selected, policy name blank on existing policy,
    // invalid target selection, or empty query. Syntax errors do not disable Save.
    const disableSaveFormErrors =
      isAddingAutomation ||
      (isEditMode && !isPatchPolicy && !isAnyPlatformSelected) ||
      (lastEditedQueryName === "" && !!lastEditedQueryId) ||
      (selectedTargetType === "Custom" &&
        !Object.entries(selectedLabels).some(([, value]) => {
          return value;
        })) ||
      errors.query === EMPTY_QUERY_ERR;

    return (
      <>
        <form className={`${baseClass}__wrapper`} autoComplete="off">
          {isEditMode ? (
            <div className={`${baseClass}__page-header`}>
              <h1 className={`${baseClass}__page-title`}>Edit policy</h1>
              {renderPolicyFleetName()}
            </div>
          ) : (
            <div className={`${baseClass}__title-bar`}>
              <div className={`${baseClass}__policy-name-fleet-name`}>
                {renderName()}
                {renderPolicyFleetName()}
              </div>
            </div>
          )}
          {isEditMode && renderName()}
          {renderDescription()}
          {renderResolution()}
          {isEditMode && !isPatchPolicy && platformSelector.render()}
          {isEditMode && isPremiumTier && !isPatchPolicy && (
            <TargetLabelSelector
              selectedTargetType={selectedTargetType}
              selectedCustomTarget={selectedCustomTarget}
              customTargetOptions={customTargetOptions}
              onSelectCustomTarget={(val) =>
                setSelectedCustomTarget(val as LabelScope)
              }
              selectedLabels={selectedLabels}
              className={`${baseClass}__target`}
              onSelectTargetType={setSelectedTargetType}
              onSelectLabel={onSelectLabel}
              labels={labels || []}
              disableOptions={gitOpsModeEnabled}
              suppressTitle
            />
          )}
          {isEditMode && storedPolicy && (
            <PolicyAutomations
              storedPolicy={storedPolicy}
              currentAutomatedPolicies={currentAutomatedPolicies}
              onAddAutomation={onAddPatchAutomation}
              isAddingAutomation={isAddingAutomation}
              gitOpsModeEnabled={!!gitOpsModeEnabled}
            />
          )}
          {isEditMode &&
            isPremiumTier &&
            !isPatchPolicy &&
            renderCriticalPolicy()}
          <SQLEditor
            value={lastEditedQueryBody}
            error={errors.query}
            label="Query"
            labelActionComponent={
              isPatchPolicy ? (
                <TooltipWrapper
                  tipContent="Query is read-only for patch policies."
                  position="top"
                  underline={false}
                  showArrow
                  tipOffset={12}
                >
                  <Icon name="info" size="small" />
                </TooltipWrapper>
              ) : (
                renderLabelComponent()
              )
            }
            name="query editor"
            onLoad={onLoad}
            wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
            onChange={onChangePolicySql}
            handleSubmit={promptSavePolicy}
            wrapEnabled
            focus={!isEditMode}
            readOnly={isPatchPolicy}
          />
          {renderPlatformCompatibility()}
          {renderLiveQueryWarning()}
          <div className="button-wrap">
            {isEditMode && onCancel && (
              <Button variant="inverse" onClick={onCancel}>
                Cancel
              </Button>
            )}
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <TooltipWrapper
                  tipContent={
                    <>
                      Select the platforms this
                      <br />
                      policy will be checked on
                      <br />
                      to save or run the policy.
                    </>
                  }
                  tooltipClass={`${baseClass}__button-wrap--tooltip`}
                  position="top"
                  disableTooltip={!isEditMode || isAnyPlatformSelected}
                  underline={false}
                >
                  <span className={`${baseClass}__button-wrap--tooltip`}>
                    <Button
                      onClick={promptSavePolicy()}
                      disabled={disableSaveFormErrors || disableChildren}
                      className="save-loading"
                      isLoading={isUpdatingPolicy}
                    >
                      Save
                    </Button>
                  </span>
                </TooltipWrapper>
              )}
            />
            <TooltipWrapper
              tipContent={
                disabledLiveQuery ? (
                  <>
                    Live reports are disabled <br />
                    in organization settings.
                  </>
                ) : (
                  <>
                    Select the platforms this <br />
                    policy will be checked on <br />
                    to save or run the policy.
                  </>
                )
              }
              disableTooltip={
                (!isEditMode || isAnyPlatformSelected) && !disabledLiveQuery
              }
              underline={false}
              showArrow
              position="top"
            >
              <span className={`${baseClass}__button-wrap--tooltip`}>
                <Button
                  onClick={goToSelectTargets}
                  disabled={
                    isAddingAutomation ||
                    (isEditMode && !isAnyPlatformSelected) ||
                    disabledLiveQuery
                  }
                  variant="inverse"
                >
                  Run <Icon name="run" />
                </Button>
              </span>
            </TooltipWrapper>
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
            labels={labels}
          />
        )}
      </>
    );
  };

  if (isStoredPolicyLoading) {
    return <Spinner />;
  }

  return renderPolicyForm();
};

export default PolicyForm;
