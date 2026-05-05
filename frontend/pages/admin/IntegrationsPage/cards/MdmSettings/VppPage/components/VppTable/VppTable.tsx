import React, { useContext } from "react";

import { IMdmVppToken } from "interfaces/mdm";

import TableContainer from "components/TableContainer";
import { AppContext } from "context/app";

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
  const { config } = useContext(AppContext); // We load gitops context here, since we can't use the default gitops wrapper since it's on a nested dropdown option
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;
  const repoURL = config?.gitops.repository_url;

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

  const tableConfig = generateTableConfig(
    onSelectAction,
    gitOpsModeEnabled ?? false,
    repoURL ?? ""
  );

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
