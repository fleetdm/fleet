import React from "react";

import {
  ISoftwareInstallerPolicyIncludeType,
  ISoftwareInstallPolicy,
} from "interfaces/software";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import SoftwareInstallPolicyBadges from "components/SoftwareInstallPolicyBadges";

interface IInstallerPoliciesTableConfig {
  teamId?: number;
}
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ISoftwareInstallerPolicyIncludeType;
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
            PATHS.EDIT_POLICY(cellProps.row.original.id),
            {
              fleet_id: teamId,
            }
          )}
          suffix={
            // TODO: We're not using type
            <SoftwareInstallPolicyBadges
              policyType={cellProps.row.original.type} // TODO: Update according to Marko
            />
          }
        />
      ),
    },
  ];

  return tableHeaders;
};

export default generateInstallerPoliciesTableConfig;
