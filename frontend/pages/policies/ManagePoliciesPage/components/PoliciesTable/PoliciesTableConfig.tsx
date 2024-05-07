/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import {
  formatDistanceToNowStrict,
  isAfter,
  millisecondsToHours,
  millisecondsToMinutes,
} from "date-fns";
import ReactTooltip from "react-tooltip";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusIndicator from "components/StatusIndicator";
import Icon from "components/Icon";
import { IPolicyStats } from "interfaces/policy";
import PATHS from "router/paths";
import sortUtils from "utilities/sort";
import { PolicyResponse } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";
import { COLORS } from "styles/var/colors";
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

const createHostsByPolicyPath = (
  policyId: number,
  policyResponse: PolicyResponse,
  teamId?: number | null
) => {
  return `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams({
    policy_id: policyId,
    policy_response: policyResponse,
    team_id: teamId,
  })}`;
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
    <span className={`tooltip__tooltip-text`}>
      Fleet is collecting policy results. Try again
      <br />
      in about {getPolicyRefreshTime(osqueryPolicyMs)} as the system catches up.
    </span>
  );
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  options: {
    selectedTeamId?: number | null;
    canAddOrDeletePolicy?: boolean;
    tableType?: string;
  },
  policiesList: IPolicyStats[] = [],
  isPremiumTier?: boolean,
  isSandboxMode?: boolean
): IDataColumn[] => {
  const { selectedTeamId, tableType, canAddOrDeletePolicy } = options;

  // Figure the time since the host counts were updated.
  // First, find first policy item with host_count_updated_at.
  const updatedAt =
    policiesList.find((p) => !!p.host_count_updated_at)
      ?.host_count_updated_at || "";
  let timeSinceHostCountUpdate = "";
  if (updatedAt) {
    try {
      timeSinceHostCountUpdate = formatDistanceToNowStrict(
        new Date(updatedAt),
        { addSuffix: true }
      );
    } catch (e) {
      // Do nothing.
    }
  }

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
      Cell: (cellProps: ICellProps): JSX.Element => (
        <LinkCell
          className="w250 policy-name-cell"
          value={
            <>
              <div className="policy-name-text">{cellProps.cell.value}</div>
              {isPremiumTier && cellProps.row.original.critical && (
                <>
                  <span
                    className="tooltip-base"
                    data-tip
                    data-for={`critical-tooltip-${cellProps.row.original.id}`}
                  >
                    <Icon
                      className="critical-policy-icon"
                      name="policy"
                      size="small"
                      color="core-fleet-blue"
                    />
                  </span>
                  <ReactTooltip
                    className="critical-tooltip"
                    place="top"
                    type="dark"
                    effect="solid"
                    id={`critical-tooltip-${cellProps.row.original.id}`}
                    backgroundColor={COLORS["tooltip-bg"]}
                  >
                    This policy has been marked as critical.
                    {isSandboxMode && (
                      <>
                        <br />
                        This is a premium feature.
                      </>
                    )}
                  </ReactTooltip>
                </>
              )}
            </>
          }
          path={PATHS.EDIT_POLICY(cellProps.row.original)}
        />
      ),
      sortType: "caseInsensitive",
    },
    {
      title: "Yes",
      Header: () => (
        <PassingColumnHeader
          isPassing
          timeSinceHostCountUpdate={timeSinceHostCountUpdate}
        />
      ),
      disableSortBy: true,
      accessor: "passing_host_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        if (cellProps.row.original.has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} host${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={createHostsByPolicyPath(
                cellProps.row.original.id,
                PolicyResponse.PASSING,
                selectedTeamId
              )}
            />
          );
        }
        return (
          <>
            <span
              className="text-cell text-muted has-not-run tooltip"
              data-tip
              data-for={`passing_${cellProps.row.original.id.toString()}`}
            >
              ---
            </span>
            <ReactTooltip
              place="bottom"
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={`passing_${cellProps.row.original.id.toString()}`}
              data-html
            >
              {getTooltip(cellProps.row.original.next_update_ms)}
            </ReactTooltip>
          </>
        );
      },
    },
    {
      title: "No",
      Header: (cellProps) => (
        <HeaderCell
          value={
            <PassingColumnHeader
              isPassing={false}
              timeSinceHostCountUpdate={timeSinceHostCountUpdate}
            />
          }
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "failing_host_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        if (cellProps.row.original.has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} host${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={createHostsByPolicyPath(
                cellProps.row.original.id,
                PolicyResponse.FAILING,
                selectedTeamId
              )}
            />
          );
        }
        return (
          <>
            <span
              className="text-cell text-muted has-not-run tooltip"
              data-tip
              data-for={`failing_${cellProps.row.original.id.toString()}`}
            >
              ---
            </span>
            <ReactTooltip
              place="bottom"
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={`failing_${cellProps.row.original.id.toString()}`}
              data-html
            >
              {getTooltip(cellProps.row.original.next_update_ms)}
            </ReactTooltip>
          </>
        );
      },
      sortType: "caseInsensitive",
    },
  ];

  if (tableType !== "inheritedPolicies") {
    if (!canAddOrDeletePolicy) {
      return tableHeaders;
    }

    tableHeaders.unshift({
      id: "selection",
      Header: (cellProps: IHeaderProps) => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} />;
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
