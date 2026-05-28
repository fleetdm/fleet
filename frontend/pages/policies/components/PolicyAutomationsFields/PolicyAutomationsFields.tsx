/* eslint-disable @typescript-eslint/no-use-before-define */

import React, {
  forwardRef,
  useImperativeHandle,
  useMemo,
  useState,
} from "react";
import { SingleValue } from "react-select-5";

import { IConfig } from "interfaces/config";
import { IPolicy } from "interfaces/policy";
import { ITeamConfig, API_NO_TEAM_ID } from "interfaces/team";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import TooltipWrapper from "components/TooltipWrapper";

import { getTicketOrWebhookInfo, getTicketOrWebhookLabel } from "./helpers";
import { IAutomationRow } from "./types";
import { useScripts, useSoftwareTitles } from "./hooks";
import { IPolicyAutomationUpdate } from "./hooks/useUpdatePolicyAutomations";

const baseClass = "policy-automations-fields";

/** Result returned to the parent on save, describing what (if anything)
 *  changed. The parent renders `error` and persists the update parts. */
export interface IPolicyAutomationsPayload {
  /** A validation message (e.g. a checked automation is missing its required
   *  selection), or null when the selection is valid. */
  error: string | null;
  /** false when nothing changed from the policy's stored automations. */
  isDirty: boolean;
  policyUpdate?: IPolicyAutomationUpdate;
  webhookOrTicketUpdate?: { enabled: boolean };
}

export interface IPolicyAutomationsFieldsHandle {
  /** Validates the current selection and returns the validation `error` (if
   *  any) plus the changed automation parts for the parent to persist. */
  getAutomationsPayload: () => IPolicyAutomationsPayload;
}

interface IPolicyAutomationsFieldsProps {
  policy: IPolicy;
  /** When true, only the webhook/ticket row is shown and the continuous-retry
   *  option is hidden (global / "All fleets" policies). */
  isGlobalPolicy: boolean;
  /** undefined for "All fleets", 0 for "Unassigned", positive for a fleet. */
  teamIdForApi: number | undefined;
  /** Config that owns the policy's automations (team config, or global config
   *  for inherited/global policies). */
  automationsConfig: IConfig | ITeamConfig | undefined;
  /** Global config — needed to read conditional access on the "Unassigned"
   *  view. */
  globalConfig: IConfig | undefined;
  /** Fleet display name, used in the "Not enabled for <fleet>" hints. */
  fleetName: string;
}

const PolicyAutomationsFields = forwardRef<
  IPolicyAutomationsFieldsHandle,
  IPolicyAutomationsFieldsProps
>(
  (
    {
      policy,
      isGlobalPolicy,
      teamIdForApi,
      automationsConfig,
      globalConfig,
      fleetName,
    },
    ref
  ) => {
    const {
      state: ticketOrWebhookState,
      policyIds: webhookOrTicketPolicyIds,
    } = getTicketOrWebhookInfo(automationsConfig);
    const isTicketWebhookEnabled = ticketOrWebhookState !== "disabled";

    const isCalendarEnabledForTeam = !isGlobalPolicy
      ? (automationsConfig as ITeamConfig | undefined)?.integrations
          ?.google_calendar?.enable_calendar_events ?? false
      : false;

    const getIsConditionalAccessEnabledForTeam = () => {
      if (isGlobalPolicy) return false;
      if (teamIdForApi === API_NO_TEAM_ID) {
        return globalConfig?.integrations?.conditional_access_enabled ?? false;
      }
      return (
        (automationsConfig as ITeamConfig | undefined)?.integrations
          ?.conditional_access_enabled ?? false
      );
    };
    const isConditionalAccessEnabledForTeam = getIsConditionalAccessEnabledForTeam();

    const initialWebhookOrTicket = webhookOrTicketPolicyIds.includes(policy.id);
    const initialInstallSoftware = !!policy.install_software;
    const initialRunScript = !!policy.run_script;
    const initialCalendar = policy.calendar_events_enabled;
    const initialConditionalAccess = policy.conditional_access_enabled;
    const initialContinuous = policy.continuous_automations_enabled ?? false;

    const [webhookOrTicketEnabled, setWebhookOrTicketEnabled] = useState(
      initialWebhookOrTicket
    );
    const [installSoftware, setInstallSoftware] = useState(
      initialInstallSoftware
    );
    const [runScript, setRunScript] = useState(initialRunScript);
    const [calendarEvent, setCalendarEvent] = useState(initialCalendar);
    const [conditionalAccess, setConditionalAccess] = useState(
      initialConditionalAccess
    );
    const [continuousEnabled, setContinuousEnabled] = useState(
      initialContinuous
    );

    const [softwareTitleId, setSoftwareTitleId] = useState<number | null>(
      policy.install_software?.software_title_id ?? null
    );
    const [scriptId, setScriptId] = useState<number | null>(
      policy.run_script?.id ?? null
    );

    const canFetchTeamScopedLists =
      !isGlobalPolicy && teamIdForApi !== undefined;
    const { data: softwareTitlesData } = useSoftwareTitles({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && installSoftware,
    });
    const { data: scriptsData } = useScripts({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && runScript,
    });

    const softwareOptions: CustomOptionType[] = useMemo(
      () =>
        (softwareTitlesData?.software_titles ?? []).map((t) => ({
          label: t.name,
          value: String(t.id),
        })),
      [softwareTitlesData]
    );

    const scriptOptions: CustomOptionType[] = useMemo(
      () =>
        (scriptsData?.scripts ?? []).map((s) => ({
          label: s.name,
          value: String(s.id),
        })),
      [scriptsData]
    );

    useImperativeHandle(ref, () => ({
      getAutomationsPayload: () => {
        // Block enabling install/run without a selection — saving would
        // silently unset the automation. The caller renders the error.
        if (installSoftware && softwareTitleId === null) {
          return {
            error: "Please select software to install.",
            isDirty: false,
          };
        }
        if (runScript && scriptId === null) {
          return { error: "Please select a script to run.", isDirty: false };
        }

        const perPolicyDirty =
          !isGlobalPolicy &&
          (installSoftware !== initialInstallSoftware ||
            softwareTitleId !==
              (policy.install_software?.software_title_id ?? null) ||
            runScript !== initialRunScript ||
            scriptId !== (policy.run_script?.id ?? null) ||
            calendarEvent !== initialCalendar ||
            conditionalAccess !== initialConditionalAccess ||
            continuousEnabled !== initialContinuous);
        const webhookDirty = webhookOrTicketEnabled !== initialWebhookOrTicket;

        return {
          error: null,
          isDirty: perPolicyDirty || webhookDirty,
          policyUpdate: perPolicyDirty
            ? {
                software_title_id: installSoftware ? softwareTitleId : null,
                script_id: runScript ? scriptId : null,
                // When the team has the feature disabled, the row is locked
                // and the user can't toggle it — so we omit the field instead
                // of carrying the stale state through to the PATCH. That
                // preserves the policy's stored intent for if/when the team
                // admin re-enables the feature.
                ...(isCalendarEnabledForTeam && {
                  calendar_events_enabled: calendarEvent,
                }),
                ...(isConditionalAccessEnabledForTeam && {
                  conditional_access_enabled: conditionalAccess,
                }),
                continuous_automations_enabled: continuousEnabled,
              }
            : undefined,
          webhookOrTicketUpdate: webhookDirty
            ? { enabled: webhookOrTicketEnabled }
            : undefined,
        };
      },
    }));

    const rows: IAutomationRow[] = [
      {
        key: "ticket_webhook",
        label: getTicketOrWebhookLabel(ticketOrWebhookState),
        checked: webhookOrTicketEnabled && isTicketWebhookEnabled,
        onToggle: setWebhookOrTicketEnabled,
        isDisabled: !isTicketWebhookEnabled,
      },
    ];
    if (!isGlobalPolicy) {
      rows.push(
        {
          key: "install_software",
          label: "Install software",
          tooltip: (
            <AutomationRowTooltip
              text="The selected software will be installed when hosts fail the policy. Host counts will reset when new software is selected."
              learnMoreUrl="https://fleetdm.com/learn-more-about/policy-automation-install-software"
            />
          ),
          checked: installSoftware,
          onToggle: setInstallSoftware,
          isDisabled: false,
          picker: installSoftware ? (
            <DropdownWrapper
              name="software-title"
              className={`${baseClass}__row-picker`}
              value={
                softwareOptions.find(
                  (o) => o.value === String(softwareTitleId ?? "")
                ) ?? null
              }
              options={softwareOptions}
              placeholder="Select software"
              onChange={(opt: SingleValue<CustomOptionType>) =>
                setSoftwareTitleId(opt ? Number(opt.value) : null)
              }
            />
          ) : undefined,
        },
        {
          key: "run_script",
          label: "Run script",
          tooltip: (
            <AutomationRowTooltip
              text="The selected script will run when hosts fail the policy. Host counts will reset when new scripts are selected."
              learnMoreUrl="https://fleetdm.com/learn-more-about/policy-automation-run-script"
            />
          ),
          checked: runScript,
          onToggle: setRunScript,
          isDisabled: false,
          picker: runScript ? (
            <DropdownWrapper
              name="script"
              className={`${baseClass}__row-picker`}
              value={
                scriptOptions.find((o) => o.value === String(scriptId ?? "")) ??
                null
              }
              options={scriptOptions}
              placeholder="Select script"
              onChange={(opt: SingleValue<CustomOptionType>) =>
                setScriptId(opt ? Number(opt.value) : null)
              }
            />
          ) : undefined,
        },
        {
          key: "calendar_event",
          label: "Calendar event",
          tooltip: (
            <AutomationRowTooltip
              text="A calendar event will be created for end users if one of their hosts fail the policy."
              learnMoreUrl="https://www.fleetdm.com/learn-more-about/calendar-events"
            />
          ),
          checked: calendarEvent && isCalendarEnabledForTeam,
          onToggle: setCalendarEvent,
          isDisabled: !isCalendarEnabledForTeam,
        },
        {
          key: "conditional_access",
          label: "Conditional access",
          tooltip: (
            <AutomationRowTooltip
              text="Single sign-on will be blocked for end users whose hosts fail the policy."
              learnMoreUrl="https://fleetdm.com/learn-more-about/conditional-access"
            />
          ),
          checked: conditionalAccess && isConditionalAccessEnabledForTeam,
          onToggle: setConditionalAccess,
          isDisabled: !isConditionalAccessEnabledForTeam,
        }
      );
    }

    return (
      <>
        <section className={`${baseClass}__section`}>
          <h2 className={`${baseClass}__section-title`}>Automations</h2>
          <table className={`${baseClass}__table`}>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={row.key}
                  className={`${baseClass}__row${
                    row.isDisabled ? ` ${baseClass}__row--disabled` : ""
                  }`}
                >
                  <td className={`${baseClass}__row-label`}>
                    <Checkbox
                      name={row.key}
                      value={row.checked}
                      disabled={row.isDisabled}
                      onChange={row.onToggle}
                    >
                      {row.tooltip ? (
                        <TooltipWrapper tipContent={row.tooltip} clickable>
                          {row.label}
                        </TooltipWrapper>
                      ) : (
                        row.label
                      )}
                    </Checkbox>
                  </td>
                  <td className={`${baseClass}__row-trailing`}>
                    {row.isDisabled ? (
                      <span className={`${baseClass}__row-disabled-hint`}>
                        Not enabled for {fleetName}
                      </span>
                    ) : (
                      row.picker
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className={`${baseClass}__learn-more`}>
            <CustomLink
              url="https://fleetdm.com/learn-more-about/policy-automations"
              text="Learn more"
              newTab
            />{" "}
            about automation types and their supported platforms.
          </div>
        </section>

        {!isGlobalPolicy && (
          <section className={`${baseClass}__section`}>
            <Checkbox
              name="continuous-automations-enabled"
              value={continuousEnabled}
              onChange={setContinuousEnabled}
              helpText="If the automations do not resolve the policy, this could cause a retry loop."
            >
              <TooltipWrapper
                tipContent="Automations run on a host's first failure, and when a host's response changes from pass to fail. If enabled, script & software automations will also run on every subsequent failure."
                clickable={false}
              >
                Continuous
              </TooltipWrapper>{" "}
              software &amp; script automations
            </Checkbox>
          </section>
        )}
      </>
    );
  }
);

interface IAutomationRowTooltipProps {
  text: string;
  learnMoreUrl: string;
}

function AutomationRowTooltip({
  text,
  learnMoreUrl,
}: IAutomationRowTooltipProps): JSX.Element {
  return (
    <>
      {text}{" "}
      <CustomLink
        url={learnMoreUrl}
        text="Learn more"
        newTab
        variant="tooltip-link"
      />
    </>
  );
}

export default PolicyAutomationsFields;
