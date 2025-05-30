import React from "react";

import { IHostPolicy } from "interfaces/policy";
import { PolicyResponse, DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { noop } from "lodash";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

interface IEnhancedHostPolicy extends IHostPolicy {
  status: PolicyStatus | null;
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

export type PolicyStatus = "pass" | "fail" | "actionRequired"; // action required indicates a failed policy with conditional access enabled
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IEnhancedHostPolicy;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const getPolicyStatus = (policy: IHostPolicy): PolicyStatus | null => {
  if (policy.response === "pass") {
    return "pass";
  }
  if (policy.response === "fail") {
    if (policy.conditional_access_enabled) {
      return "actionRequired";
    }
    return "fail";
  }
  // can occur when response === ""
  return null;
};

const POLICY_STATUS_TO_INDICATOR_PARAMS: Record<
  PolicyStatus,
  [IndicatorStatus, string]
> = {
  pass: ["success", "Pass"],
  fail: ["failure", "Fail"],
  actionRequired: ["actionRequired", "Action required"],
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePolicyTableHeaders = (
  togglePolicyDetails: (policy: IHostPolicy, teamId?: number) => void,
  currentTeamId?: number
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      accessor: "name",
      disableSortBy: true,
      Cell: (cellProps) => {
        const { name } = cellProps.row.original;

        return <LinkCell customOnClick={noop} tooltipTruncate value={name} />;
      },
    },
    {
      title: "Status",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      sortType: "hostPolicyStatus",
      accessor: "status",
      Cell: (cellProps) => {
        const {
          row: {
            original: { status },
          },
        } = cellProps;
        if (status === null) {
          return <>{DEFAULT_EMPTY_CELL_VALUE}</>;
        }
        const [
          indicatorStatus,
          displayText,
        ] = POLICY_STATUS_TO_INDICATOR_PARAMS[status];
        return (
          <StatusIndicatorWithIcon
            value={displayText}
            status={indicatorStatus}
          />
        );
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps) => {
        return (
          <>
            {cellProps.row.original.response && (
              <ViewAllHostsLink
                queryParams={{
                  policy_id: cellProps.row.original.id,
                  policy_response:
                    cellProps.row.original.response === "pass"
                      ? PolicyResponse.PASSING
                      : PolicyResponse.FAILING,
                  team_id: currentTeamId,
                }}
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];
};

const generatePolicyDataSet = (
  policies: IHostPolicy[]
): IEnhancedHostPolicy[] => {
  return policies.map((policy) => ({
    ...policy,
    status: getPolicyStatus(policy),
  }));
};

export { generatePolicyTableHeaders, generatePolicyDataSet };
