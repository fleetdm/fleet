import { IConfig } from "interfaces/config";
import { Platform, PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { TicketOrWebhookState } from "interfaces/policy";
import {
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  ISoftwareTitle,
} from "interfaces/software";
import { ITeamConfig } from "interfaces/team";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

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

export const generateSoftwareOptionHelpText = (
  title: ISoftwareTitle
): string => {
  const isVppApp = title.source === "apps" && !!title.app_store_app;

  if (isVppApp) {
    const version = title.app_store_app?.version
      ? ` • ${title.app_store_app.version}`
      : "";
    return `macOS (App Store)${version}`;
  }

  const platform: Platform | null =
    INSTALLABLE_SOURCE_PLATFORM_CONVERSION[title.source] || null;
  const extension = getExtensionFromFileName(
    title.software_package?.name ?? ""
  );
  const platformString =
    platform && extension
      ? `${PLATFORM_DISPLAY_NAMES[platform]} (.${extension})`
      : "";
  const version = title.software_package?.version ?? "";
  const separator = platformString && version ? " • " : "";

  return `${platformString}${separator}${version}`;
};
