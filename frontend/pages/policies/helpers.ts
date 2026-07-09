import { IConfig } from "interfaces/config";
import { Platform, PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { TicketOrWebhookState } from "interfaces/policy";
import {
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  ISoftwarePackage,
  ISoftwareTitle,
} from "interfaces/software";
import { ITeamConfig } from "interfaces/team";
import { addedFromNow } from "utilities/date_format";
import { getExtensionFromFileName } from "utilities/file/fileUtils";
import { pluralize } from "utilities/strings/stringUtils";

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

/** Help-text shown under each option in the OUTER "Select software" dropdown
 * on the policy automations modal. Renders `platform (type) • <version>` for
 * VPP / App Store and single-package custom titles, or `platform (type) •
 * N versions` for multi-package custom titles. For the INNER "Select package"
 * dropdown that surfaces when a multi-package title is picked, see
 * `generateSoftwarePackageOptionHelpText`. */
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

  // Multi-package custom titles show a version count ("3 versions") in the
  // outer dropdown; the per-package picker below the outer dropdown carries
  // the actual version + upload date. Single-package titles keep the
  // existing "version string" treatment since there's nothing to count.
  const packageCount = title.packages?.length ?? 0;
  const versionOrCount =
    packageCount > 1
      ? `${packageCount} ${pluralize(packageCount, "version")}`
      : title.software_package?.version ?? "";
  const separator = platformString && versionOrCount ? " • " : "";

  return `${platformString}${separator}${versionOrCount}`;
};

/** Help-text shown under each option in the INNER "Select package" dropdown
 * that appears when a multi-package title is picked. Mirrors the Library
 * row's "version • Added X ago" secondary line. For the OUTER "Select
 * software" dropdown that lists titles, see `generateSoftwareOptionHelpText`. */
export const generateSoftwarePackageOptionHelpText = (
  pkg: ISoftwarePackage
): string => {
  const separator = pkg.version && pkg.uploaded_at ? " • " : "";
  const added = pkg.uploaded_at ? `Added ${addedFromNow(pkg.uploaded_at)}` : "";
  return `${pkg.version ?? ""}${separator}${added}`;
};

/** Returns the "first-added" package on a multi-package title, defined as the
 * smallest `installer_id`. The API returns `packages[]` in that order today,
 * but we `Math.min` defensively so the auto-select doesn't drift if the
 * response order ever changes. Returns `null` for titles with no packages
 * (e.g. VPP / App Store titles). */
export const findFirstAddedPackage = (
  packages: ISoftwarePackage[] | null | undefined
): ISoftwarePackage | null => {
  if (!packages || packages.length === 0) return null;
  return packages.reduce((first, pkg) =>
    pkg.installer_id < first.installer_id ? pkg : first
  );
};
