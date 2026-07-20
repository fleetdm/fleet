import React, {
  useState,
  useContext,
  useEffect,
  useCallback,
  useMemo,
  useRef,
} from "react";
import { size } from "lodash";
import classNames from "classnames";
import { useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPlatformSelector } from "hooks/usePlatformSelector";
import { IConfig } from "interfaces/config";
import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { ITeamConfig } from "interfaces/team";
import useDeepEffect from "hooks/useDeepEffect";

import configAPI from "services/entities/config";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI from "services/entities/teams";

import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { TargetLabelSelector } from "components/TargetLabelSelector";
import Icon from "components/Icon";

import PolicyAutomationsFields, {
  IPolicyAutomationsFieldsHandle,
} from "pages/policies/components/PolicyAutomationsFields";
import { usePolicyLabelTargets } from "pages/policies/hooks";
import { POLICY_TARGET_EMPTY_STATE_DESCRIPTION } from "pages/policies/constants";

const NAME_MAX_LENGTH = 255;

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
    errors.name = "Policy name must be present";
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
    setLastEditedQueryName,
    setLastEditedQueryPlatform,
    // TODO: Keep last edited query platform from resetting when cancelling out of modal and clicking save again
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
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

  const disableForm =
    isFetchingAutofillDescription || isFetchingAutofillResolution;
  const disableSave =
    !platformSelector.isAnyPlatformSelected ||
    disableForm ||
    (selectedTargetType === "Custom" && !hasCustomLabels);

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
    setLastEditedQueryPlatform(newPlatformString);

    const { valid: validName, errors: newErrors } = validatePolicyName(
      lastEditedQueryName
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (disableSave || !validName) {
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
              variant="inverse"
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
          inputOptions={{ maxLength: NAME_MAX_LENGTH }}
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
        {platformSelector.render()}
        {isPremiumTier && (
          <TargetLabelSelector
            {...selectorProps}
            className={`${baseClass}__target`}
            emptyStateDescription={POLICY_TARGET_EMPTY_STATE_DESCRIPTION}
            onAddLabel={() => router.push(PATHS.LABEL_NEW_DYNAMIC)}
            disableOptions={disableForm}
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
            />
          </div>
        ) : (
          <div className={`${baseClass}__add-automations`}>
            <Button
              variant="secondary"
              type="button"
              onClick={() => setShowAutomations(true)}
            >
              <Icon name="plus" /> Add automations
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
        <div className="modal-cta-wrap">
          <TooltipWrapper
            tipContent={
              <>
                Select the platforms this
                <br />
                policy will be checked on
                <br />
                to save the policy.
              </>
            }
            tooltipClass={`${baseClass}__button--modal-save-tooltip`}
            position="top"
            disableTooltip={!disableSave}
            underline={false}
            showArrow
            tipOffset={8}
          >
            <span className={`${baseClass}__button-wrap--modal-save`}>
              <Button
                type="submit"
                disabled={disableSave}
                className="save-policy-loading"
                isLoading={isUpdatingPolicy}
              >
                Save
              </Button>
            </span>
          </TooltipWrapper>
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
