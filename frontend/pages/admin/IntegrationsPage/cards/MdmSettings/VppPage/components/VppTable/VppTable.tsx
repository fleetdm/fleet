import React from "react";

import { IMdmVppToken } from "interfaces/mdm";

import TableContainer from "components/TableContainer";

import { generateTableConfig } from "./VppTableConfig";

const baseClass = "vpp-table";

interface IVppTableProps {
  vppTokens: IMdmVppToken[];
  onEditTokenTeam: (token: IMdmVppToken) => void;
  onRenewToken: (token: IMdmVppToken) => void;
  onDeleteToken: (token: IMdmVppToken) => void;
}

const VppTable = ({
  vppTokens,
  onEditTokenTeam,
  onRenewToken,
  onDeleteToken,
}: IVppTableProps) => {
  const onSelectAction = (action: string, abmToken: IMdmVppToken) => {
    switch (action) {
      case "editTeams":
        onEditTokenTeam(abmToken);
        break;
      case "renew":
        onRenewToken(abmToken);
        break;
      case "delete":
        onDeleteToken(abmToken);
        break;
      default:
        break;
    }
  };

  const tableConfig = generateTableConfig(onSelectAction);

  return (
    <TableContainer<IMdmVppToken>
      columnConfigs={tableConfig}
      defaultSortHeader="org_name"
      disableTableHeader
      disablePagination
      showMarkAllPages={false}
      isAllPagesSelected={false}
      emptyComponent={() => <></>}
      isLoading={false}
      data={vppTokens}
      className={baseClass}
    />
  );
};

export default VppTable;
