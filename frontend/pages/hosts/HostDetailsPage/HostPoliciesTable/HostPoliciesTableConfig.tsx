import React from "react";
import { Link } from "react-router";
import PATHS from "router/paths";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Button from "components/buttons/Button";
import { IHostPolicy } from "interfaces/host_policy";
import { PolicyResponse } from "utilities/constants";

import Chevron from "../../../../../assets/images/icon-chevron-right-9x6@2x.png";
import ArrowIcon from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

const TAGGED_TEMPLATES = {
  hostsByPolicyRoute: (policyId: number, policyResponse: PolicyResponse) => {
    return `?policy_id=${policyId}&policy_response=${policyResponse}`;
  },
};

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}
interface ICellProps {
  cell: {
    value: any;
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
    return "Passing";
  } else if (policy.response === "fail") {
    return "Failing";
  }
  return "---";
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePolicyTableHeaders = (
  togglePolicyDetails: () => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      accessor: "name",
      disableSortBy: true,
      Cell: (cellProps) => {
        const { query_name } = cellProps.row.original;
        return (
          <>
            <Button onClick={togglePolicyDetails} variant={"text-icon"}>
              <>
                {query_name}
                <img src={ArrowIcon} alt="View policy details" />
              </>
            </Button>
          </>
        );
      },
    },
    {
      title: "Status",
      Header: "Status",
      accessor: "response",
      disableSortBy: true,
      Cell: (cellProps) => {
        return <TextCell value={getPolicyStatus(cellProps.row.original)} />;
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps) => {
        return (
          <Link
            to={
              PATHS.MANAGE_HOSTS +
              TAGGED_TEMPLATES.hostsByPolicyRoute(
                cellProps.row.original.id,
                cellProps.row.original.response === "pass"
                  ? PolicyResponse.PASSING
                  : PolicyResponse.FAILING
              )
            }
            className={`policy-link`}
          >
            <img alt="link to hosts filtered by policy ID" src={Chevron} />
          </Link>
        );
      },
    },
  ];
};

const generatePolicyDataSet = (policies: IHostPolicy[]): IHostPolicy[] => {
  return policies;
};

export { generatePolicyTableHeaders, generatePolicyDataSet };
