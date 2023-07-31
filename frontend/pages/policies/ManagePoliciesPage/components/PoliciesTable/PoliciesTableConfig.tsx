/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { millisecondsToHours, millisecondsToMinutes, isAfter } from "date-fns";
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

  isPremiumTier?: boolean,
  isSandboxMode?: boolean
): IDataColumn[] => {
  const { selectedTeamId, tableType, canAddOrDeletePolicy } = options;

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
          classes="w250 policy-name-cell"
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
                      name="critical-policy"
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
      Header: () => <PassingColumnHeader isPassing />,
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
              backgroundColor="#3e4771"
              id={`passing_${cellProps.row.original.id.toString()}`}
              data-html
            >
              {getTooltip(cellProps.row.original.osquery_policy_ms)}
            </ReactTooltip>
          </>
        );
      },
    },
    {
      title: "No",
      Header: (cellProps) => (
        <HeaderCell
          value={<PassingColumnHeader isPassing={false} />}
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
              backgroundColor="#3e4771"
              id={`failing_${cellProps.row.original.id.toString()}`}
              data-html
            >
              {getTooltip(cellProps.row.original.osquery_policy_ms)}
            </ReactTooltip>
          </>
        );
      },
      sortType: "caseInsensitive",
    },
  ];

  if (tableType !== "inheritedPolicies") {
    tableHeaders.push({
      title: "Automations",
      Header: "Automations",
      disableSortBy: true,
      accessor: "webhook",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <StatusIndicator value={cellProps.cell.value} />
      ),
    });

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

const generateDataSet = (
  policiesList: IPolicyStats[] = [],
  currentAutomatedPolicies?: number[],
  osquery_policy?: number
): IPolicyStats[] => {
  policiesList = policiesList.sort((a, b) =>
    sortUtils.caseInsensitiveAsc(a.name, b.name)
  );
  let policiesLastRun: Date;
  let osqueryPolicyMs: number;

  if (osquery_policy) {
    osqueryPolicyMs = osquery_policy / 1000000;
    // Convert from nanosecond to milliseconds
    policiesLastRun = new Date(Date.now() - osqueryPolicyMs);
  }

  policiesList.forEach((policyItem) => {
    policyItem.webhook =
      currentAutomatedPolicies &&
      currentAutomatedPolicies.includes(policyItem.id)
        ? "On"
        : "Off";

    // Define policy has_run based on updated_at compared againist last time policies ran as
    // defined by osquery_policy.
    policyItem.has_run = isAfter(
      policiesLastRun,
      new Date(policyItem.updated_at)
    );
    // Include osquery policy in item for reference in tooltip
    policyItem.osquery_policy_ms = osqueryPolicyMs;
  });

  return policiesList;
};

export { generateTableHeaders, generateDataSet };
