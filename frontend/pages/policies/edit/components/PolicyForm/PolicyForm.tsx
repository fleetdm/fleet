/* eslint-disable jsx-a11y/no-noninteractive-element-to-interactive-role */
/* eslint-disable jsx-a11y/interactive-supports-focus */
import React, { useState, useContext, useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "react-query";

import { Ace } from "ace-builds";
import { useDebouncedCallback } from "use-debounce";
import { size } from "lodash";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import { PolicyContext } from "context/policy";
import usePlatformCompatibility from "hooks/usePlatformCompatibility";
import usePlatformSelector from "hooks/usePlatformSelector";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import { IPolicy, IPolicyFormData } from "interfaces/policy";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  APP_CONTEXT_NO_TEAM_ID,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import {
  DEFAULT_POLICIES,
  POLICY_TARGET_EMPTY_STATE_DESCRIPTION,
} from "pages/policies/constants";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import SQLEditor from "components/SQLEditor";
import { validateQuery, EMPTY_QUERY_ERR } from "components/forms/validators";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import Icon from "components/Icon/Icon";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import CustomLink from "components/CustomLink";
import { TargetLabelSelector } from "components/TargetLabelSelector";

import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
  IPolicyAutomationsPayload,
} from "pages/policies/components/PolicyAutomationsFields";
import { PatchAutomationCta } from "pages/policies/components";
import {
  useUpdatePolicyAutomations,
  usePolicyLabelTargets,
} from "pages/policies/hooks";

import SaveNewPolicyModal from "../SaveNewPolicyModal";

const baseClass = "policy-form";

const NAME_MAX_LENGTH = 255;

interface IPolicyFormProps {
  router: InjectedRouter;
  teamIdForApi?: number;
  policyIdForEdit: number | null;
  showOpenSchemaActionText: boolean;
  storedPolicy: IPolicy | undefined;
  isStoredPolicyLoading: boolean;
  isTeamObserver: boolean;
  isUpdatingPolicy: boolean;
  onCreatePolicy: (
    formData: IPolicyFormData,
    saveAutomations?: (newPolicy: IPolicy) => Promise<void>
  ) => void;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  // Returns a Promise so the form can sequence the automations save AFTER the
  // core update completes (see the patch-policy save flow in promptSavePolicy).
  onUpdate: (formData: IPolicyFormData) => Promise<unknown>;
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
  const { error: queryError, isValid: queryValid } = validateQuery(query);

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
}: IPolicyFormProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({}); // string | null | undefined or boolean | undefined
  const [isSaveNewPolicyModalOpen, setIsSaveNewPolicyModalOpen] = useState(
    false
  );

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
    lastEditedQueryLabelsExcludeAll,
    defaultPolicy,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const {
    selectorProps,
    selectedTargetType,
    hasCustomLabels,
    getLabelsPayload,
  } = usePolicyLabelTargets({
    includeAny: lastEditedQueryLabelsIncludeAny,
    includeAll: lastEditedQueryLabelsIncludeAll,
    excludeAny: lastEditedQueryLabelsExcludeAny,
    excludeAll: lastEditedQueryLabelsExcludeAll,
  });

  const queryClient = useQueryClient();

  const {
    currentTeam,
    isGlobalObserver,
    isTeamTechnician,
    isGlobalTechnician,
    isOnGlobalTeam,
    isPremiumTier,
    config,
    isFreeTier,
  } = useContext(AppContext);

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

  // For an existing policy, ownership comes from the policy's stored team_id.
  // For a new policy, it comes from the team the user is currently viewing,
  // since that's where the policy will be created.
  const newPolicyTeamId =
    currentTeam?.id !== undefined && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;
  const isGlobalPolicy = isEditMode
    ? storedPolicy?.team_id == null
    : newPolicyTeamId === undefined;
  const automationsTeamId = isEditMode
    ? storedPolicy?.team_id ?? undefined
    : newPolicyTeamId;

  const { data: automationsTeamData } = useQuery<ILoadTeamResponse, Error>(
    ["teams", automationsTeamId],
    () => teamsAPI.load(automationsTeamId as number),
    {
      enabled: !isGlobalPolicy && automationsTeamId !== undefined,
      staleTime: 5000,
    }
  );

  const automationsConfig =
    (isGlobalPolicy ? config : automationsTeamData?.team) ?? undefined;

  let automationsFleetName = "";
  if (isGlobalPolicy) {
    automationsFleetName = APP_CONTEXT_ALL_TEAMS_SUMMARY.name;
  } else if (automationsTeamId === APP_CONTEXT_NO_TEAM_ID) {
    automationsFleetName = APP_CONTEXT_NO_TEAM_SUMMARY.name;
  } else {
    automationsFleetName =
      automationsTeamData?.team?.name ?? currentTeam?.name ?? "";
  }

  const automationsRef = useRef<IPolicyAutomationsFieldsHandle>(null);

  const {
    mutate: saveAutomations,
    isLoading: isSavingAutomations,
  } = useUpdatePolicyAutomations({
    policy: storedPolicy,
    teamIdForApi: automationsTeamId,
    isGlobalPolicy,
    automationsConfig,
    onSuccess: () => {
      queryClient.invalidateQueries(["policy", policyIdForEdit]);
    },
    onError: () => notify.error("Could not update policy automations."),
  });

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
      storedPolicy?.team_id == null
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
      notify.success("Automation added.");
    } catch (e) {
      notify.error("Couldn't set automation. Please try again.", {
        response: e,
      });
    } finally {
      setIsAddingAutomation(false);
    }
  };

  const promptSavePolicy = () => async (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    if (isEditMode && !lastEditedQueryName) {
      setErrors({ ...errors, name: "Policy name must be present" });
      return;
    }

    if (isEditMode && !isPatchPolicy && !isAnyPlatformSelected) {
      setErrors({
        ...errors,
        name: "At least one platform must be selected",
      });
      return;
    }

    // Capture + validate automation changes up front so an invalid selection
    // blocks the whole save before anything is persisted.
    let automations: IPolicyAutomationsPayload | undefined;
    if (isEditMode) {
      automations = automationsRef.current?.getAutomationsPayload();
      if (automations && !automations.isValid) {
        return;
      }
    }

    // The core update (onUpdate) and the automations update both PATCH the policy.
    // We `await` the core update before firing the automations one so the
    // automations write is always the LAST write to the policy. This matters
    // for patch policies, where the backend re-links install_software to
    // patch_software whenever a patch policy is updated.
    const persistAutomations = () => {
      if (automations?.isDirty) {
        saveAutomations({
          policyUpdate: automations.policyUpdate,
          webhookOrTicketUpdate: automations.webhookOrTicketUpdate,
        });
      }
    };

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
      await onUpdate(payload);
      persistAutomations();
      return;
    }

    // Synchronously block empty queries. The button-level `disableSaveFormErrors`
    // relies on the debounced `errors.query`, so a fast click before the debounce
    // fires could otherwise submit an empty query. Mirrors the guard in
    // EditQueryForm's handleSaveQuery (see #38348).
    if (!lastEditedQueryBody?.trim()) {
      setErrors({ ...errors, query: EMPTY_QUERY_ERR });
      return;
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
        Object.assign(payload, getLabelsPayload());
        payload.critical = lastEditedQueryCritical;
      }
      await onUpdate(payload);
      persistAutomations();
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
          inputOptions={{ maxLength: NAME_MAX_LENGTH }}
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
    if (isFreeTier) return null;

    // In edit mode, the displayed Fleet must reflect the policy's actual
    // owner, not the URL/navigation context: a user can land here by clicking
    // an inherited (global) policy from a team's policy list, in which case
    // currentTeam reflects the team URL, not the policy's true Fleet.
    let fleetName: string | undefined;
    if (isEditMode) {
      if (storedPolicy?.team_id === null) {
        fleetName = APP_CONTEXT_ALL_TEAMS_SUMMARY.name;
      } else if (storedPolicy?.team_id === 0) {
        fleetName = APP_CONTEXT_NO_TEAM_SUMMARY.name;
      } else {
        fleetName = currentTeam?.name;
      }
    } else {
      fleetName = currentTeam?.name;
    }

    if (!fleetName) return null;

    return isEditMode ? (
      <p>
        Editing policy for <strong>{fleetName}</strong>.
      </p>
    ) : (
      <p>
        Creating a new policy for <strong>{fleetName}</strong>.
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
      (selectedTargetType === "Custom" && !hasCustomLabels) ||
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
              {...selectorProps}
              className={`${baseClass}__target`}
              emptyStateDescription={POLICY_TARGET_EMPTY_STATE_DESCRIPTION}
              onAddLabel={() => router.push(PATHS.LABEL_NEW_DYNAMIC)}
              disableOptions={gitOpsModeEnabled}
            />
          )}
          {isEditMode && !!storedPolicy && !!automationsConfig && (
            <div className="form-field">
              <div className="form-field__label">Automations</div>
              <PatchAutomationCta
                storedPolicy={storedPolicy}
                canEditPolicy={isEditMode}
                onAddAutomation={onAddPatchAutomation}
                isAddingAutomation={isAddingAutomation}
              />
              <PolicyAutomationsFields
                key={storedPolicy.updated_at}
                ref={automationsRef}
                policy={storedPolicy}
                isGlobalPolicy={isGlobalPolicy}
                teamIdForApi={automationsTeamId}
                automationsConfig={automationsConfig}
                globalConfig={config ?? undefined}
                fleetName={automationsFleetName}
              />
            </div>
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
                      isLoading={isUpdatingPolicy || isSavingAutomations}
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
                  Run policy <Icon name="run" />
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
            isGlobalPolicy={isGlobalPolicy}
            policyTeamId={automationsTeamId}
            automationsConfig={automationsConfig}
            globalConfig={config ?? undefined}
            fleetName={automationsFleetName}
            router={router}
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
