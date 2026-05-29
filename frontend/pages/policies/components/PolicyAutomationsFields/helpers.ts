import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { TicketOrWebhookState } from "./types";

export interface ITicketOrWebhookInfo {
  /** "webhook" or "ticket" when an "other workflow" automation is configured
   *  on the policy's fleet/global config; "disabled" otherwise. */
  state: TicketOrWebhookState;
  /** Policy IDs configured for the active webhook/ticket automation, or `[]`
   *  when disabled.
   *  NOTE: the backend stores membership for both webhooks and integrations
   *  in webhook_settings.failing_policies_webhook.policy_ids. */
  policyIds: number[];
}

export const getTicketOrWebhookInfo = (
  automationsConfig: IConfig | ITeamConfig | undefined
): ITicketOrWebhookInfo => {
  if (!automationsConfig) return { state: "disabled", policyIds: [] };

  const webhookEnabled =
    automationsConfig.webhook_settings?.failing_policies_webhook
      ?.enable_failing_policies_webhook ?? false;

  const integrations = automationsConfig.integrations;
  const ticketEnabled =
    !!integrations?.jira?.some((j) => j.enable_failing_policies) ||
    !!integrations?.zendesk?.some((z) => z.enable_failing_policies);

  let state: TicketOrWebhookState = "disabled";
  if (webhookEnabled) state = "webhook";
  else if (ticketEnabled) state = "ticket";

  const policyIds =
    state === "disabled"
      ? []
      : automationsConfig.webhook_settings?.failing_policies_webhook
          ?.policy_ids ?? [];

  return { state, policyIds };
};

export const getTicketOrWebhookLabel = (
  state: TicketOrWebhookState
): string => {
  if (state === "webhook") return "Send webhook";
  if (state === "ticket") return "Create ticket";
  return "Send webhook or create ticket";
};
