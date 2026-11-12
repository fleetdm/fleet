import React from "react";

import SoftwareInstallPolicyBadges from "components/SoftwareInstallPolicyBadges";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import { ISoftwareInstallPolicyUI } from "interfaces/software";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

interface IInstallerPoliciesTableConfig {
  teamId?: number;
}
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ISoftwareInstallPolicyUI;
  };
  column: {
    isSortedDesc: boolean;
    title: string;
  };
}

const generateInstallerPoliciesTableConfig = ({
  teamId,
}: IInstallerPoliciesTableConfig) => {
  const tableHeaders = [
    {
      accessor: "name",
      title: "Name",
      Header: (cellProps: ICellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: (cellProps: ICellProps) => (
        <LinkCell
          value={cellProps.cell.value}
          tooltipTruncate
          path={getPathWithQueryParams(
            PATHS.POLICY_DETAILS(cellProps.row.original.id),
            {
              fleet_id: teamId,
            }
          )}
          className="w400"
          suffix={
            <SoftwareInstallPolicyBadges
              policyType={cellProps.row.original.type}
            />
          }
        />
      ),
    },
  ];

  return tableHeaders;
};

export default generateInstallerPoliciesTableConfig;
