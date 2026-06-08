import React from "react";

import { IMdmAbToken } from "interfaces/mdm";
import useGitOpsMode from "hooks/useGitOpsMode";

import TableContainer from "components/TableContainer";

import { generateTableConfig } from "./AppleBusinessManagerTableConfig";

const baseClass = "apple-business-manager-table";

interface IAppleBusinessManagerTableProps {
  abTokens: IMdmAbToken[];
  onEditTokenTeam: (token: IMdmAbToken) => void;
  onRenewToken: (token: IMdmAbToken) => void;
  onDeleteToken: (token: IMdmAbToken) => void;
}

const AppleBusinessManagerTable = ({
  abTokens,
  onEditTokenTeam,
  onRenewToken,
  onDeleteToken,
}: IAppleBusinessManagerTableProps) => {
  const { gitOpsModeEnabled, repoURL } = useGitOpsMode();

  const onSelectAction = (action: string, abmToken: IMdmAbToken) => {
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

  const tableConfig = generateTableConfig(
    onSelectAction,
    gitOpsModeEnabled,
    repoURL
  );

  return (
    <TableContainer<IMdmAbToken>
      columnConfigs={tableConfig}
      defaultSortHeader="org_name"
      disableTableHeader
      disablePagination
      showMarkAllPages={false}
      isAllPagesSelected={false}
      emptyComponent={() => <></>}
      isLoading={false}
      data={abTokens}
      className={baseClass}
    />
  );
};

export default AppleBusinessManagerTable;
