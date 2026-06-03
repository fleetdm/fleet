import { useContext } from "react";
import { useMutation, useQueryClient } from "react-query";

import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI from "services/entities/teams";

/** The per-policy automation fields settable from the modal. */
export type IPolicyAutomationUpdate = Pick<
  IPolicyFormData,
  | "software_title_id"
  | "script_id"
  | "calendar_events_enabled"
  | "conditional_access_enabled"
  | "continuous_automations_enabled"
>;

export interface IUpdatePolicyAutomationsVars {
  /** Present only when per-policy automation fields changed. */
  policyUpdate?: IPolicyAutomationUpdate;
  /** Present only when webhook/ticket membership changed; `enabled` is the
   *  desired membership for this policy. */
  webhookOrTicketUpdate?: { enabled: boolean };
}

interface IUseUpdatePolicyAutomationsArgs {
  /** May be `undefined` during initial render / for a not-yet-loaded policy;
   *  the mutation guards against this and is only invokable once it's set. */
  policy: IPolicy | undefined;
  teamIdForApi: number | undefined;
  isGlobalPolicy: boolean;
  automationsConfig: IConfig | ITeamConfig | undefined;
  onSuccess?: () => void;
  onError?: () => void;
}

/** Saves a single policy's automations: the per-policy fields via the policy
 *  update endpoint, and webhook/ticket membership via the fleet/global config. */
const useUpdatePolicyAutomations = ({
  policy,
  teamIdForApi,
  isGlobalPolicy,
  automationsConfig,
  onSuccess,
  onError,
}: IUseUpdatePolicyAutomationsArgs) => {
  const queryClient = useQueryClient();
  const { setConfig } = useContext(AppContext);

  if (!isGlobalPolicy && teamIdForApi === undefined) {
    throw new Error("Missing fleet id for team-scoped policy automations.");
  }

  // Adds or removes this policy from the fleet/global webhook+ticket policy_ids
  // list (the backend stores membership for both webhooks and tickets there).
  const saveWebhookOrTicketMembership = async (
    policyId: number,
    enabled: boolean
  ) => {
    const existingWebhook =
      automationsConfig?.webhook_settings?.failing_policies_webhook ?? {};
    const currentIds = existingWebhook.policy_ids ?? [];
    const nextIds = enabled
      ? Array.from(new Set([...currentIds, policyId]))
      : currentIds.filter((id) => id !== policyId);

    const payload = {
      webhook_settings: {
        failing_policies_webhook: { ...existingWebhook, policy_ids: nextIds },
      },
    };

    if (isGlobalPolicy) {
      const updatedConfig = await configAPI.update(payload);
      queryClient.setQueryData(["config"], updatedConfig);
      setConfig(updatedConfig);
    } else {
      const updatedTeam = await teamsAPI.update(payload, teamIdForApi);
      queryClient.setQueryData(["teams", teamIdForApi], updatedTeam);
    }
  };

  return useMutation(
    ({ policyUpdate, webhookOrTicketUpdate }: IUpdatePolicyAutomationsVars) => {
      if (!policy) {
        return Promise.reject(
          new Error("Cannot update automations without a policy.")
        );
      }
      const { id: policyId } = policy;

      const requests: Promise<unknown>[] = [];
      if (policyUpdate) {
        requests.push(
          teamPoliciesAPI.update(policyId, {
            team_id: teamIdForApi,
            ...policyUpdate,
          })
        );
      }
      if (webhookOrTicketUpdate) {
        requests.push(
          saveWebhookOrTicketMembership(policyId, webhookOrTicketUpdate.enabled)
        );
      }
      return Promise.all(requests);
    },
    {
      onSuccess,
      onError: () => {
        if (isGlobalPolicy) {
          queryClient.invalidateQueries(["config"]);
        } else {
          queryClient.invalidateQueries(["teams", teamIdForApi]);
        }
        onError?.();
      },
    }
  );
};

export default useUpdatePolicyAutomations;
