import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { TicketOrWebhookState } from "./types";

/** Identifies whether the team has webhook automations or ticket integrations
 *  enabled for failing policies. */
export const getTicketOrWebhookState = (
  automationsConfig: IConfig | ITeamConfig | undefined
): TicketOrWebhookState => {
  if (!automationsConfig) return "disabled";

  const webhookEnabled =
    automationsConfig.webhook_settings?.failing_policies_webhook
      ?.enable_failing_policies_webhook ?? false;

  const integrations = automationsConfig.integrations;
  const ticketEnabled =
    !!integrations?.jira?.some((j) => j.enable_failing_policies) ||
    !!integrations?.zendesk?.some((z) => z.enable_failing_policies);

  if (webhookEnabled) return "webhook";
  if (ticketEnabled) return "ticket";
  return "disabled";
};

export const getTicketOrWebhookLabel = (
  state: TicketOrWebhookState
): string => {
  if (state === "webhook") return "Send webhook";
  if (state === "ticket") return "Create ticket";
  return "Send webhook or create ticket";
};
