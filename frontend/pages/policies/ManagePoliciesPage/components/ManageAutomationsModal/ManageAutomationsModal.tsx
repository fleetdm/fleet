/* eslint-disable @typescript-eslint/no-use-before-define */

import React, { useMemo, useState } from "react";
import { useQuery } from "react-query";
import { omit } from "lodash";
import { SingleValue } from "react-select-5";

import { IPolicyStats } from "interfaces/policy";
import { IConfig } from "interfaces/config";
import { ITeamConfig, API_NO_TEAM_ID } from "interfaces/team";
import {
  CommaSeparatedPlatformString,
  PLATFORM_DISPLAY_NAMES,
  QueryablePlatform,
} from "interfaces/platform";
import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import { getTicketOrWebhookLabel, getTicketOrWebhookState } from "./helpers";
import { IAutomationRow } from "./types";

const baseClass = "manage-automations-modal";

const PLATFORM_DISPLAY_ORDER: QueryablePlatform[] = [
  "darwin",
  "windows",
  "linux",
  "chrome",
];

const SOFTWARE_PAGE_SIZE = 1000;
const SCRIPTS_PAGE_SIZE = 1000;

interface IManageAutomationsModalProps {
  policy: IPolicyStats;
  fleetName: string;
  isGlobalPolicy: boolean;
  /** undefined for "All fleets", 0 for "No team", positive for a team. */
  teamIdForApi: number | undefined;
  automationsConfig: IConfig | ITeamConfig | undefined;
  globalConfig: IConfig | undefined;
  /** Policy IDs that have webhook/ticket (a.k.a. "other workflow")
   *  automations configured on this team. Membership is what drives the
   *  initial checked state of the webhook/ticket row. */
  webhookOrTicketPolicyIds: number[];
  onExit: () => void;
}

const ManageAutomationsModal = ({
  policy,
  fleetName,
  isGlobalPolicy,
  teamIdForApi,
  automationsConfig,
  globalConfig,
  webhookOrTicketPolicyIds,
  onExit,
}: IManageAutomationsModalProps): JSX.Element => {
  const ticketOrWebhookState = getTicketOrWebhookState(automationsConfig);

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

  const initialSendWebhook = webhookOrTicketPolicyIds.includes(policy.id);
  const initialInstallSoftware = !!policy.install_software;
  const initialRunScript = !!policy.run_script;
  const initialCalendar = policy.calendar_events_enabled;
  const initialConditionalAccess = policy.conditional_access_enabled;
  const initialContinuous = policy.continuous_automations_enabled ?? false;

  const [sendWebhook, setSendWebhook] = useState(initialSendWebhook);
  const [installSoftware, setInstallSoftware] = useState(
    initialInstallSoftware
  );
  const [runScript, setRunScript] = useState(initialRunScript);
  const [calendarEvent, setCalendarEvent] = useState(initialCalendar);
  const [conditionalAccess, setConditionalAccess] = useState(
    initialConditionalAccess
  );
  const [continuousEnabled, setContinuousEnabled] = useState(initialContinuous);

  const [softwareTitleId, setSoftwareTitleId] = useState<number | null>(
    policy.install_software?.software_title_id ?? null
  );
  const [scriptId, setScriptId] = useState<number | null>(
    policy.run_script?.id ?? null
  );

  const canFetchTeamScopedLists = !isGlobalPolicy && teamIdForApi !== undefined;
  const { data: softwareTitlesData } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: 0,
        perPage: SOFTWARE_PAGE_SIZE,
        query: "",
        orderDirection: "desc",
        orderKey: "hosts_count",
        teamId: teamIdForApi ?? 0,
        availableForInstall: true,
        platform: "darwin,windows,linux" as CommaSeparatedPlatformString,
      },
    ],
    ({ queryKey: [key] }) => softwareAPI.getSoftwareTitles(omit(key, "scope")),
    {
      enabled: canFetchTeamScopedLists && installSoftware,
      staleTime: 30_000,
    }
  );

  const { data: scriptsData } = useQuery<
    IScriptsResponse,
    Error,
    IScriptsResponse,
    [IListScriptsQueryKey]
  >(
    [
      {
        scope: "scripts",
        page: 0,
        per_page: SCRIPTS_PAGE_SIZE,
        fleet_id: teamIdForApi ?? 0,
      },
    ],
    ({ queryKey: [key] }) => scriptsAPI.getScripts(omit(key, "scope")),
    {
      enabled: canFetchTeamScopedLists && runScript,
      staleTime: 30_000,
    }
  );

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

  const policyPlatforms = (policy.platform ?? "")
    .split(",")
    .map((p) => p.trim())
    .filter((p): p is QueryablePlatform =>
      (PLATFORM_DISPLAY_ORDER as string[]).includes(p)
    );
  const displayedPlatforms = PLATFORM_DISPLAY_ORDER.filter((p) =>
    policyPlatforms.includes(p)
  );

  const isTicketWebhookEnabled = ticketOrWebhookState !== "disabled";
  const rows: IAutomationRow[] = [
    {
      key: "ticket_webhook",
      label: getTicketOrWebhookLabel(ticketOrWebhookState),
      checked: sendWebhook && isTicketWebhookEnabled,
      onToggle: setSendWebhook,
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
    <Modal
      title="Manage automations"
      onExit={onExit}
      className={baseClass}
      width="large"
    >
      <div className={`${baseClass}__body`}>
        <div className={`${baseClass}__header`}>
          Manage automations for the <b>{policy.name}</b> policy on{" "}
          <b>{fleetName}</b>.
        </div>

        {displayedPlatforms.length > 0 && (
          <section className={`${baseClass}__section`}>
            <h2 className={`${baseClass}__section-title`}>Platforms</h2>
            <div className={`${baseClass}__platforms`}>
              {displayedPlatforms.map((p) => (
                <span key={p} className={`${baseClass}__platform`}>
                  <Icon name={p} size="small" />
                  {PLATFORM_DISPLAY_NAMES[p]}
                </span>
              ))}
            </div>
          </section>
        )}

        <section className={`${baseClass}__section`}>
          <h2 className={`${baseClass}__section-title`}>Automations</h2>
          <table className={`${baseClass}__automations-table`}>
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
      </div>

      <div className="modal-cta-wrap">
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
        <Button onClick={onExit}>Save</Button>
      </div>
    </Modal>
  );
};

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

export default ManageAutomationsModal;
