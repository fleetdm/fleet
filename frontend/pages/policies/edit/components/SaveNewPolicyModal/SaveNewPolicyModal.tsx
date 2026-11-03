import classNames from "classnames";
import { size } from "lodash";
import React, {
  useState,
  useContext,
  useEffect,
  useCallback,
  useMemo,
  useRef,
} from "react";
import { useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import Icon from "components/Icon";
import Modal from "components/Modal";
import { TargetLabelSelector } from "components/TargetLabelSelector";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import useDeepEffect from "hooks/useDeepEffect";
import { IPlatformSelector } from "hooks/usePlatformSelector";
import { IConfig } from "interfaces/config";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { ITeamConfig } from "interfaces/team";
import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
} from "pages/policies/components/PolicyAutomationsFields";
import { POLICY_TARGET_EMPTY_STATE_DESCRIPTION } from "pages/policies/constants";
import { usePolicyLabelTargets } from "pages/policies/hooks";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI from "services/entities/teams";
import { MAX_ENTITY_CHAR_LENGTH } from "utilities/constants";

export interface ISaveNewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  onCreatePolicy: (
    formData: IPolicyFormData,
    saveAutomations?: (newPolicy: IPolicy) => Promise<void>
  ) => void;
  setIsSaveNewPolicyModalOpen: (isOpen: boolean) => void;
  backendValidators: { [key: string]: string };
  platformSelector: IPlatformSelector;
  isUpdatingPolicy: boolean;
  aiFeaturesDisabled?: boolean;
  isFetchingAutofillDescription: boolean;
  isFetchingAutofillResolution: boolean;
  onClickAutofillDescription: () => Promise<void>;
  onClickAutofillResolution: () => Promise<void>;
  /** True when the new policy targets "All fleets" (global); only the
   *  webhook/ticket row is shown in the automations table. */
  isGlobalPolicy: boolean;
  /** undefined for global, 0 for "Unassigned", positive for a fleet. */
  policyTeamId: number | undefined;
  /** Config that owns the new policy's automations: global config for global
   *  policies, the team's config for team policies. */
  automationsConfig: IConfig | ITeamConfig | undefined;
  /** Global config — needed for the conditional access row on the
   *  "Unassigned" view. */
  globalConfig: IConfig | undefined;
  /** Display name of the fleet the new policy belongs to. */
  fleetName: string;
  router: InjectedRouter;
}

const validatePolicyName = (name: string) => {
  const errors: { [key: string]: string } = {};

  if (!name) {
    errors.name = "Enter a policy name";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const SaveNewPolicyModal = ({
  baseClass,
  queryValue,
  onCreatePolicy,
  setIsSaveNewPolicyModalOpen,
  backendValidators,
  platformSelector,
  isUpdatingPolicy,
  aiFeaturesDisabled,
  isFetchingAutofillDescription,
  isFetchingAutofillResolution,
  onClickAutofillDescription,
  onClickAutofillResolution,
  isGlobalPolicy,
  policyTeamId,
  automationsConfig,
  globalConfig,
  fleetName,
  router,
}: ISaveNewPolicyModalProps): JSX.Element => {
  const { isPremiumTier, setConfig } = useContext(AppContext);
  const queryClient = useQueryClient();
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
    lastEditedQueryCritical,
    lastEditedQueryHidden,
    setLastEditedQueryName,
    setLastEditedQueryPlatform,
    // TODO: Keep last edited query platform from resetting when cancelling out of modal and clicking save again
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryHidden,
  } = useContext(PolicyContext);

  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  const {
    selectorProps,
    selectedTargetType,
    hasCustomLabels,
    getLabelsPayload,
  } = usePolicyLabelTargets();

  const [showAutomations, setShowAutomations] = useState(false);
  const automationsRef = useRef<IPolicyAutomationsFieldsHandle>(null);
  const [conditionalAccessOn, setConditionalAccessOn] = useState(false);

  useEffect(() => {
    if (conditionalAccessOn) {
      setLastEditedQueryHidden(false);
    }
  }, [conditionalAccessOn, setLastEditedQueryHidden]);

  const newPolicyStub = useMemo(
    () =>
      ({
        id: -1,
        team_id: policyTeamId ?? null,
        calendar_events_enabled: false,
        conditional_access_enabled: false,
        continuous_automations_enabled: false,
      } as IPolicy),
    [policyTeamId]
  );

  const [platformError, setPlatformError] = useState<string | null>(null);
  const [showLabelsError, setShowLabelsError] = useState(false);

  const disableForm =
    isFetchingAutofillDescription || isFetchingAutofillResolution;

  const labelsRequired = selectedTargetType === "Custom" && !hasCustomLabels;
  const labelsError =
    showLabelsError && labelsRequired ? "Select at least one label" : null;

  useEffect(() => {
    if (platformSelector.isAnyPlatformSelected) {
      setPlatformError(null);
    }
  }, [platformSelector.isAnyPlatformSelected]);

  useEffect(() => {
    if (!labelsRequired) {
      setShowLabelsError(false);
    }
  }, [labelsRequired]);

  useDeepEffect(() => {
    if (lastEditedQueryName) {
      setErrors({});
    }
  }, [lastEditedQueryName]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const handleSavePolicy = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const newPlatformString = platformSelector
      .getSelectedPlatforms()
      .join(",") as CommaSeparatedPlatformString;
    // Empty string in platformContext is a legacy "no restriction → check
    // all four" signal in usePlatformSelector. Persist null so clicking Save
    // with zero platforms doesn't flip to "all platforms" instead of
    // surfacing the inline error.
    setLastEditedQueryPlatform(newPlatformString || null);

    const { valid: validName, errors: newErrors } = validatePolicyName(
      lastEditedQueryName
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    const hasPlatform = platformSelector.isAnyPlatformSelected;
    setPlatformError(hasPlatform ? null : "Select at least one platform");
    setShowLabelsError(labelsRequired);

    if (disableForm || !validName || !hasPlatform || labelsRequired) {
      return;
    }

    const automations = showAutomations
      ? automationsRef.current?.getAutomationsPayload()
      : undefined;
    if (automations && !automations.isValid) {
      return;
    }

    const payload: IPolicyFormData = {
      description: lastEditedQueryDescription,
      name: lastEditedQueryName,
      query: queryValue,
      resolution: lastEditedQueryResolution,
      platform: newPlatformString,
      critical: lastEditedQueryCritical,
    };
    if (isPremiumTier) {
      Object.assign(payload, getLabelsPayload());
      payload.hidden = lastEditedQueryHidden;
    }

    // The create endpoint deliberately ignores automation fields (see the
    // pick in team_policies.ts:create) — they have to be PATCHed after the
    // policy exists. Build a saveAutomations closure to run after create.
    const saveAutomations = automations?.isDirty
      ? async (newPolicy: IPolicy) => {
          const requests: Promise<unknown>[] = [];

          if (automations.policyUpdate && !isGlobalPolicy) {
            requests.push(
              teamPoliciesAPI.update(newPolicy.id, {
                team_id: policyTeamId,
                ...automations.policyUpdate,
              })
            );
          }

          if (automations.webhookOrTicketUpdate?.enabled) {
            const existingWebhook =
              automationsConfig?.webhook_settings?.failing_policies_webhook ??
              {};
            const currentIds = existingWebhook.policy_ids ?? [];
            const nextIds = Array.from(new Set([...currentIds, newPolicy.id]));
            const webhookPayload = {
              webhook_settings: {
                failing_policies_webhook: {
                  ...existingWebhook,
                  policy_ids: nextIds,
                },
              },
            };
            if (isGlobalPolicy) {
              requests.push(
                configAPI.update(webhookPayload).then((updatedConfig) => {
                  queryClient.setQueryData(["config"], updatedConfig);
                  setConfig(updatedConfig);
                })
              );
            } else if (policyTeamId !== undefined) {
              requests.push(
                teamsAPI
                  .update(webhookPayload, policyTeamId)
                  .then((updatedTeam) => {
                    queryClient.setQueryData(
                      ["teams", policyTeamId],
                      updatedTeam
                    );
                  })
              );
            }
          }

          await Promise.all(requests);
        }
      : undefined;

    onCreatePolicy(payload, saveAutomations);
  };

  const renderAutofillButton = useCallback(
    (labelName: "Description" | "Resolution") => {
      const isFetchingButton =
        (labelName === "Description" && isFetchingAutofillDescription) ||
        (labelName === "Resolution" && isFetchingAutofillResolution);

      return (
        <TooltipWrapper
          tipContent={
            aiFeaturesDisabled ? (
              "AI features are disabled in organization settings"
            ) : (
              <>
                Policy queries (SQL) will be sent to a <br />
                large language model (LLM). Fleet <br />
                doesn&apos;t use this data to train models.
              </>
            )
          }
          position="top"
          disableTooltip={disableForm}
          underline={false}
        >
          <div className="autofill-tooltip-wrapper">
            <Button
              variant="subdued"
              disabled={aiFeaturesDisabled || disableForm}
              onClick={
                labelName === "Description"
                  ? onClickAutofillDescription
                  : onClickAutofillResolution
              }
              size="small"
            >
              {isFetchingButton ? (
                "Thinking..."
              ) : (
                <>
                  <Icon name="sparkles" /> Autofill
                </>
              )}
            </Button>
          </div>
        </TooltipWrapper>
      );
    },
    [isFetchingAutofillDescription, isFetchingAutofillResolution, disableForm]
  );

  const renderAutofillLabel = useCallback(
    (labelName: "Description" | "Resolution") => {
      const labelClassName = classNames(`${baseClass}__autofill-label`, {
        [`${baseClass}__label--${labelName}`]: !!labelName,
      });

      return (
        <div className={labelClassName}>
          {labelName}
          {renderAutofillButton(labelName)}
        </div>
      );
    },
    [renderAutofillButton]
  );

  return (
    <Modal
      title="Save policy"
      onExit={() => setIsSaveNewPolicyModalOpen(false)}
      width="large"
    >
      <form
        onSubmit={handleSavePolicy}
        className={`${baseClass}__save-modal-form`}
        autoComplete="off"
      >
        <InputField
          name="name"
          onChange={(value: string) => setLastEditedQueryName(value)}
          value={lastEditedQueryName}
          error={errors.name}
          inputClassName={`${baseClass}__policy-save-modal-name`}
          label="Name"
          autofocus
          disabled={disableForm}
          inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
        />
        <InputField
          name="description"
          onChange={(value: string) => setLastEditedQueryDescription(value)}
          value={lastEditedQueryDescription}
          inputClassName={`${baseClass}__policy-save-modal-description`}
          label={renderAutofillLabel("Description")}
          helpText="How does this policy's failure put the organization at risk?"
          type="textarea"
          disabled={disableForm}
        />
        <InputField
          name="resolution"
          onChange={(value: string) => setLastEditedQueryResolution(value)}
          value={lastEditedQueryResolution}
          inputClassName={`${baseClass}__policy-save-modal-resolution`}
          label={renderAutofillLabel("Resolution")}
          type="textarea"
          helpText="If this policy fails, what should the end user expect?"
          disabled={disableForm}
        />
        {platformSelector.render(platformError)}
        {isPremiumTier && (
          <TargetLabelSelector
            {...selectorProps}
            className={`${baseClass}__target`}
            emptyStateDescription={POLICY_TARGET_EMPTY_STATE_DESCRIPTION}
            onAddLabel={() => router.push(PATHS.LABEL_NEW_DYNAMIC)}
            disableOptions={disableForm}
            labelsError={labelsError}
          />
        )}
        {showAutomations ? (
          <div className="form-field">
            <div className="form-field__label">Automations</div>
            <PolicyAutomationsFields
              ref={automationsRef}
              policy={newPolicyStub}
              isGlobalPolicy={isGlobalPolicy}
              teamIdForApi={policyTeamId}
              automationsConfig={automationsConfig}
              globalConfig={globalConfig}
              fleetName={fleetName}
              selectedPlatforms={platformSelector.getSelectedPlatforms()}
              onConditionalAccessChange={setConditionalAccessOn}
            />
          </div>
        ) : (
          <div className={`${baseClass}__add-automations`}>
            <Button
              variant="secondary"
              type="button"
              onClick={() => setShowAutomations(true)}
              icon="plus"
            >
              Add automations
            </Button>
          </div>
        )}
        {isPremiumTier && (
          <div className="critical-checkbox-wrapper">
            <Checkbox
              name="critical-policy"
              onChange={(value: boolean) => setLastEditedQueryCritical(value)}
              value={lastEditedQueryCritical}
              disabled={disableForm}
            >
              <TooltipWrapper
                tipContent={
                  <p>
                    If automations are turned on, this information is included.
                    If Okta conditional access is configured, end users can
                    never bypass critical policies.
                  </p>
                }
              >
                Critical
              </TooltipWrapper>
            </Checkbox>
          </div>
        )}
        {isPremiumTier && (
          <div className="hidden-checkbox-wrapper">
            <Checkbox
              name="hidden-policy"
              onChange={(value: boolean) => setLastEditedQueryHidden(value)}
              value={lastEditedQueryHidden}
              disabled={disableForm || conditionalAccessOn}
            >
              <TooltipWrapper
                tipContent={
                  conditionalAccessOn
                    ? "This setting is not compatible with the conditional access automation."
                    : "Does not require action from the end user and is hidden in Fleet Desktop."
                }
              >
                Hide from end user
              </TooltipWrapper>
            </Checkbox>
          </div>
        )}
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            className="save-policy-loading"
            isLoading={isUpdatingPolicy}
            disabled={disableForm}
          >
            Save
          </Button>
          <Button
            className={`${baseClass}__button--modal-cancel`}
            type="button"
            onClick={() => setIsSaveNewPolicyModalOpen(false)}
            variant="secondary"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default SaveNewPolicyModal;
