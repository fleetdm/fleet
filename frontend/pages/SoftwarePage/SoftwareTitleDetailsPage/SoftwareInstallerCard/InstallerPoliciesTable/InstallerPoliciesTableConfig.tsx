import React from "react";

import { ISoftwareInstallPolicy } from "interfaces/software";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

interface IInstallerPoliciesTableConfig {
  teamId?: number;
}
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ISoftwareInstallPolicy;
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
          path={getPathWithQueryParams(
            PATHS.EDIT_POLICY(cellProps.row.original.id),
            {
              team_id: teamId,
            }
          )}
        />
      ),
    },
  ];

  return tableHeaders;
};

export default generateInstallerPoliciesTableConfig;
