import React from "react";
import StatusIndicator from "components/StatusIndicator";
import Button from "components/buttons/Button";
import { IHostPolicy } from "interfaces/policy";
import { PolicyResponse, DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import ViewAllHostsLink from "components/ViewAllHostsLink";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHostPolicy;
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

const getPolicyStatus = (policy: IHostPolicy): string => {
  if (policy.response === "pass") {
    return "Yes";
  } else if (policy.response === "fail") {
    return "No";
  }
  return DEFAULT_EMPTY_CELL_VALUE;
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePolicyTableHeaders = (
  togglePolicyDetails: (policy: IHostPolicy) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      accessor: "name",
      disableSortBy: true,
      Cell: (cellProps) => {
        const { name } = cellProps.row.original;
        return (
          <Button
            className={`policy-info`}
            onClick={() => {
              togglePolicyDetails(cellProps.row.original);
            }}
            variant={"text-icon"}
          >
            <span className={`policy-info-text`}>{name}</span>
          </Button>
        );
      },
    },
    {
      title: "Status",
      Header: "Status",
      accessor: "response",
      disableSortBy: true,
      Cell: (cellProps) => {
        return (
          <StatusIndicator value={getPolicyStatus(cellProps.row.original)} />
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
                }}
                className="policy-link"
              />
            )}
          </>
        );
      },
    },
  ];
};

const generatePolicyDataSet = (policies: IHostPolicy[]): IHostPolicy[] => {
  return policies;
};

export { generatePolicyTableHeaders, generatePolicyDataSet };
