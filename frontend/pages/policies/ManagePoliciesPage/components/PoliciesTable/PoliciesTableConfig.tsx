/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { millisecondsToHours, millisecondsToMinutes } from "date-fns";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { IPolicyStats, OtherAutomationType } from "interfaces/policy";
import PATHS from "router/paths";
import ENDPOINTS from "utilities/endpoints";

import { getPathWithQueryParams } from "utilities/url";
import sortUtils from "utilities/sort";
import { DEFAULT_EMPTY_CELL_VALUE, PolicyResponse } from "utilities/constants";

import CriticalPolicyBadge from "components/CriticalPolicyBadge";
import PillBadge from "components/PillBadge";
import { PATCH_TOOLTIP_CONTENT } from "components/SoftwareInstallPolicyBadges/SoftwareInstallPolicyBadges";
import { getConditionalSelectHeaderCheckboxProps } from "components/TableContainer/utilities/config_utils";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { getAutomationsForPolicy, IAutomationData } from "../../helpers";
import PassingColumnHeader from "../PassingColumnHeader";

interface IGetToggleAllRowsSelectedProps {
  checked: boolean;
  indeterminate: boolean;
  title: string;
  onChange: () => void;
  style: { cursor: string };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => IGetToggleAllRowsSelectedProps;
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IPolicyStats;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const AUTOMATION_ICON_RENDERERS: Record<
  IAutomationData["type"],
  (args: { name: string; iconUrl?: string }) => JSX.Element
> = {
  software: ({ name, iconUrl }) => (
    <span className="automations__software-icon">
      <SoftwareIcon name={name} url={iconUrl} size="small" />
    </span>
  ),
  script: ({ name }) => (
    <Graphic
      name={name.endsWith(".sh") ? "file-sh" : "file-ps1"}
      className="scale-40-24"
    />
  ),
  calendar: () => <Graphic name="calendar" />,
  conditional_access: () => <Graphic name="lock" />,
  other: () => <Icon name="settings" />,
};

interface IAutomationsCellProps {
  policy: IPolicyStats;
  selectedTeamId?: number | null;
  otherAutomationType?: OtherAutomationType;
  onOpenManageAutomationsModal?: (policy: IPolicyStats) => void;
}

const AutomationsCell = ({
  policy,
  selectedTeamId,
  otherAutomationType,
  onOpenManageAutomationsModal,
}: IAutomationsCellProps): JSX.Element => {
  const automations = getAutomationsForPolicy(policy, otherAutomationType);

  if (automations.length === 0) {
    return (
      <span className="automations__cell-content automations__cell-content--none">
        {DEFAULT_EMPTY_CELL_VALUE}
      </span>
    );
  }

  const handleClick = () => onOpenManageAutomationsModal?.(policy);
  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      handleClick();
    }
  };

  const renderAutomationIcon = ({
    type,
    name,
    softwareTitleId,
  }: IAutomationData) => {
    const iconUrl =
      type === "software" && softwareTitleId != null
        ? `/api${getPathWithQueryParams(
            ENDPOINTS.SOFTWARE_ICON(softwareTitleId),
            {
              fleet_id:
                selectedTeamId != null && selectedTeamId !== -1
                  ? selectedTeamId
                  : undefined,
            }
          )}`
        : undefined;
    return AUTOMATION_ICON_RENDERERS[type]({ name, iconUrl });
  };

  if (automations.length === 1) {
    const automation = automations[0];
    return (
      <div
        role="button"
        tabIndex={0}
        className="automations__cell-content"
        onClick={handleClick}
        onKeyDown={handleKeyDown}
        aria-label={`Edit automation: ${automation.name}`}
      >
        <TooltipTruncatedTextCell
          prefix={renderAutomationIcon(automation)}
          value={automation.name}
          className="automations__name"
        />
        <span className="automations__edit-button" aria-hidden="true">
          <Icon name="pencil" />
        </span>
      </div>
    );
  }

  return (
    <div
      role="button"
      tabIndex={0}
      className="automations__cell-content"
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      aria-label="Edit automations"
    >
      <TooltipWrapper
        className="automations__count"
        position="top"
        underline={false}
        fixedPositionStrategy
        tipOffset={8}
        tipContent={automations.map(({ name }) => name).join(", ")}
      >
        {automations.length} automations
      </TooltipWrapper>
      <span className="automations__edit-button" aria-hidden="true">
        <Icon name="pencil" />
      </span>
    </div>
  );
};

const getPolicyRefreshTime = (ms: number): string => {
  const seconds = ms / 1000;
  if (seconds < 60) {
    return `${seconds} seconds`;
  }
  if (seconds < 3600) {
    const minutes = millisecondsToMinutes(ms);
    return `${minutes} minute${minutes > 1 ? "s" : ""}`;
  }
  const hours = millisecondsToHours(ms);
  return `${hours} hour${hours > 1 ? "s" : ""}`;
};

const getTooltip = (osqueryPolicyMs: number): JSX.Element => {
  return (
    <>
      Fleet is collecting policy results. Try again
      <br />
      in about {getPolicyRefreshTime(osqueryPolicyMs)} as the system catches up.
    </>
  );
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  options: {
    selectedTeamId?: number | null;
    hasPermissionAndPoliciesToDelete?: boolean;
    tableType?: string;
    otherAutomationType?: OtherAutomationType;
    onOpenManageAutomationsModal?: (policy: IPolicyStats) => void;
  },
  isPremiumTier?: boolean,
  isPrimoMode?: boolean
): IDataColumn[] => {
  const {
    selectedTeamId,
    hasPermissionAndPoliciesToDelete,
    otherAutomationType,
    onOpenManageAutomationsModal,
  } = options;
  const viewingTeamPolicies = selectedTeamId !== -1;

  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { critical, id, team_id, type } = cellProps.row.original;
        return (
          <LinkCell
            className="w250"
            tooltipTruncate
            value={cellProps.cell.value}
            suffix={
              <>
                {isPremiumTier && critical && <CriticalPolicyBadge />}
                {type === "patch" && (
                  <PillBadge tipContent={PATCH_TOOLTIP_CONTENT}>
                    Patch
                  </PillBadge>
                )}
                {viewingTeamPolicies && team_id === null && (
                  <PillBadge tipContent="This policy runs on all hosts.">
                    Inherited
                  </PillBadge>
                )}
              </>
            }
            path={getPathWithQueryParams(PATHS.POLICY_DETAILS(id), {
              // Inherited policies show team_id === null; preserve the
              // current team context so back nav returns to the same list
              // instead of "All teams".
              fleet_id:
                team_id ?? (selectedTeamId !== -1 ? selectedTeamId : null),
            })}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Automations",
      Header: "Automations",
      accessor: "automations",
      disableSortBy: true,
      Cell: (cellProps: ICellProps): JSX.Element => (
        <AutomationsCell
          policy={cellProps.row.original}
          selectedTeamId={selectedTeamId}
          otherAutomationType={otherAutomationType}
          onOpenManageAutomationsModal={onOpenManageAutomationsModal}
        />
      ),
    },
    {
      title: "Pass",
      Header: (cellProps) => (
        <HeaderCell
          value={<PassingColumnHeader isPassing />}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "passing_host_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { has_run, id, next_update_ms } = cellProps.row.original;

        if (has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} host${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
                policy_id: id,
                policy_response: PolicyResponse.PASSING,
                fleet_id: selectedTeamId,
              })}
            />
          );
        }
        return (
          <div className="policy-has-not-run">
            <TooltipWrapper
              tooltipClass="policy-has-not-run-tooltip"
              position="top"
              underline={false}
              fixedPositionStrategy
              tipOffset={8}
              tipContent={getTooltip(next_update_ms)}
            >
              ---
            </TooltipWrapper>
          </div>
        );
      },
    },
    {
      title: "Fail",
      Header: (cellProps) => (
        <HeaderCell
          value={<PassingColumnHeader isPassing={false} />}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "failing_host_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { has_run, id, next_update_ms } = cellProps.row.original;

        if (has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} host${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
                policy_id: id,
                policy_response: PolicyResponse.FAILING,
                fleet_id: selectedTeamId,
              })}
            />
          );
        }
        return (
          <div className="policy-has-not-run">
            <TooltipWrapper
              tooltipClass="policy-has-not-run-tooltip"
              position="top"
              underline={false}
              fixedPositionStrategy
              tipOffset={8}
              tipContent={getTooltip(next_update_ms)}
            >
              ---
            </TooltipWrapper>
          </div>
        );
      },
      sortType: "caseInsensitive",
    },
  ];

  if (hasPermissionAndPoliciesToDelete) {
    tableHeaders.unshift({
      id: "selection",
      // TODO: headerProps is `any` because local IHeaderProps is a simplified
      // subset of react-table's HeaderProps. Fixing requires refactoring
      // IDataColumn/IHeaderProps to align with react-table's actual types.
      Header: (headerProps: any) => {
        // When viewing team policies, the select all checkbox will ignore inherited policies
        const teamCheckboxProps = getConditionalSelectHeaderCheckboxProps({
          headerProps,
          checkIfRowIsSelectable: (row) =>
            // allow selecting inherited policies in primo mode
            isPrimoMode || row.original.team_id !== null,
        });

        // Regular table selection logic
        const {
          getToggleAllRowsSelectedProps,
          toggleAllRowsSelected,
        } = headerProps;
        const { checked, indeterminate } = getToggleAllRowsSelectedProps();

        const regularCheckboxProps = {
          value: checked,
          indeterminate,
          onChange: () => {
            toggleAllRowsSelected();
          },
        };

        const checkboxProps = viewingTeamPolicies
          ? teamCheckboxProps
          : regularCheckboxProps;
        return (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={8}
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <Checkbox
                disabled={disableChildren}
                enableEnterToCheck
                {...checkboxProps}
              />
            )}
          />
        );
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const inheritedPolicy = cellProps.row.original.team_id === null;
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };

        // When viewing team policies and a row is an inherited policy, do not render checkbox
        if (viewingTeamPolicies && inheritedPolicy && !isPrimoMode) {
          return <></>;
        }

        return (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={8}
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <Checkbox
                disabled={disableChildren}
                enableEnterToCheck
                {...checkboxProps}
              />
            )}
          />
        );
      },
      disableHidden: true,
    });
  }

  return tableHeaders;
};

// The next update will match the next host count update, unless extra time is needed for hosts to send in their policy results.
const nextPolicyUpdateMs = (
  policyItemUpdatedAtMs: Date,
  nextHostCountUpdateMs: number,
  hostCountUpdateIntervalMs: number,
  osqueryPolicyMs: number
) => {
  let timeFromPolicyItemUpdateToNextHostCountUpdateMs =
    Date.now() - policyItemUpdatedAtMs.getTime() + nextHostCountUpdateMs;
  let additionalUpdateTimeMs = 0;
  while (timeFromPolicyItemUpdateToNextHostCountUpdateMs <= osqueryPolicyMs) {
    additionalUpdateTimeMs += hostCountUpdateIntervalMs;
    timeFromPolicyItemUpdateToNextHostCountUpdateMs += hostCountUpdateIntervalMs;
  }
  return nextHostCountUpdateMs + additionalUpdateTimeMs;
};

const generateDataSet = (
  policiesList: IPolicyStats[] = [],
  currentAutomatedPolicies?: number[],
  osquery_policy?: number
): IPolicyStats[] => {
  policiesList = policiesList.sort((a, b) =>
    sortUtils.caseInsensitiveAsc(a.name, b.name)
  );
  // To figure out if the policy has run for all the targeted hosts, we need to do the following calculation:
  // Each host asynchronously updates its own policy result every `osquery_policy` nanoseconds.
  // Then, the host count is updated by a cron job on the server every 1 hour (this is hardcoded on the server in `cron.go`).
  // So, we need to add `osquery_policy` to the time of the cron update.
  let policiesLastRun: Date;
  let osqueryPolicyMs = 0;
  const policiesThatHaveRunHostCountUpdatedAt =
    // host counts of all policies that have run are updated at the same time, and are therefore
    // identical, so we can use the first one. Those that haven't run will be `null`.
    policiesList.find((p) => !!p.host_count_updated_at)
      ?.host_count_updated_at || "";
  // If host_count_updated_at is not present, we assume the worst case.
  const hostCountUpdateIntervalMs = 60 * 60 * 1000; // 1 hour (from server's `cron.go`)
  const hostCountUpdatedAtDate = policiesThatHaveRunHostCountUpdatedAt
    ? new Date(policiesThatHaveRunHostCountUpdatedAt)
    : new Date(Date.now() - hostCountUpdateIntervalMs);
  if (osquery_policy) {
    // Convert from nanosecond to milliseconds
    osqueryPolicyMs = osquery_policy / 1000000;
    policiesLastRun = new Date(
      hostCountUpdatedAtDate.getTime() - osqueryPolicyMs
    );
  } else {
    // temporarily unused - will restore use with upcoming DB update
    policiesLastRun = hostCountUpdatedAtDate;
  }
  // Now we figure out when the next host count update will be.
  // The % (mod) is used below in case server was restarted and previously scheduled host count update was skipped.
  const nextHostCountUpdateMs =
    hostCountUpdateIntervalMs -
    (policiesThatHaveRunHostCountUpdatedAt
      ? (Date.now() - hostCountUpdatedAtDate.getTime()) %
        hostCountUpdateIntervalMs
      : 0);

  policiesList.forEach((policyItem) => {
    policyItem.webhook =
      currentAutomatedPolicies &&
      currentAutomatedPolicies.includes(policyItem.id)
        ? "On"
        : "Off";

    // Define policy has_run based on updated_at compared against last time policies ran.
    const policyItemUpdatedAt = new Date(policyItem.updated_at);
    // TODO: restore and update setting of policyItem.has_run based on upcoming custom
    // `policy_membership_updated_at`(ish) DB column/API response field
    // policyItem.has_run = isAfter(policiesLastRun, policyItemUpdatedAt);

    // all of the policiess `has_run` will be either true (cron has run, so host_count_updated_at
    // has a value that is the same for all such policies) or false (policy is new, wasn't included
    // in last cron run, host_count_updated_at is `null`)
    policyItem.has_run = !!policyItem.host_count_updated_at;
    if (!policyItem.has_run) {
      // Include time for next update for reference in tooltip, which is only present if policy has not run.
      policyItem.next_update_ms = nextPolicyUpdateMs(
        policyItemUpdatedAt,
        nextHostCountUpdateMs,
        hostCountUpdateIntervalMs,
        osqueryPolicyMs
      );
    }
  });

  return policiesList;
};

export { generateTableHeaders, generateDataSet, nextPolicyUpdateMs };
