import React from "react";

import { IMdmAbmToken } from "interfaces/mdm";

import TableContainer from "components/TableContainer";

import { generateTableConfig } from "./AppleBusinessManagerTableConfig";

const baseClass = "apple-business-manager-table";

interface IAppleBusinessManagerTableProps {
  abmTokens: IMdmAbmToken[];
  onEditTokenTeam: (token: IMdmAbmToken) => void;
  onRenewToken: (token: IMdmAbmToken) => void;
  onDeleteToken: (token: IMdmAbmToken) => void;
}

const AppleBusinessManagerTable = ({
  abmTokens,
  onEditTokenTeam,
  onRenewToken,
  onDeleteToken,
}: IAppleBusinessManagerTableProps) => {
  const onSelectAction = (action: string, abmToken: IMdmAbmToken) => {
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
    <TableContainer<IMdmAbmToken>
      columnConfigs={tableConfig}
      defaultSortHeader="org_name"
      disableTableHeader
      disablePagination
      showMarkAllPages={false}
      isAllPagesSelected={false}
      emptyComponent={() => <></>}
      isLoading={false}
      data={abmTokens}
      className={baseClass}
    />
  );
};

export default AppleBusinessManagerTable;
