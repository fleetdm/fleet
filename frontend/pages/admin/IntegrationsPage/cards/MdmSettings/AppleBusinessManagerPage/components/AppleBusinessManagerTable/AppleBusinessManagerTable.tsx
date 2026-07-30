import React, { useState } from "react";

import { IMdmAbToken } from "interfaces/mdm";
import useGitOpsMode from "hooks/useGitOpsMode";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";

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
  const [searchQuery, setSearchQuery] = useState("");
  const normalizedQuery = searchQuery.toLowerCase();
  const filteredAbTokens = normalizedQuery
    ? abTokens.filter((token) =>
        token.org_name.toLowerCase().includes(normalizedQuery)
      )
    : abTokens;

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

  const onQueryChange = (queryData: ITableQueryData) => {
    setSearchQuery(queryData.searchQuery);
  };

  return (
    <TableContainer<IMdmAbToken>
      columnConfigs={tableConfig}
      defaultSortHeader="org_name"
      disablePagination
      showMarkAllPages={false}
      isAllPagesSelected={false}
      emptyComponent={() => <></>}
      isLoading={false}
      data={filteredAbTokens}
      className={baseClass}
      searchable
      inputPlaceHolder="Search by organization name"
      searchQuery={searchQuery}
      onQueryChange={onQueryChange}
    />
  );
};

export default AppleBusinessManagerTable;
