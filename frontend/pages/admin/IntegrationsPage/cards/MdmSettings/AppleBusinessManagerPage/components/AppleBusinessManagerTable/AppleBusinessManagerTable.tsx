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
  const [filteredAbTokens, setFilteredAbTokens] = useState(abTokens);

  const handleSearchQueryChange = (query: string) => {
    setSearchQuery(query);
    const lowerCaseQuery = query.toLowerCase();
    const filteredTokens = abTokens.filter((token) =>
      token.org_name.toLowerCase().includes(lowerCaseQuery)
    );
    setFilteredAbTokens(filteredTokens);
  };

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
    handleSearchQueryChange(queryData.searchQuery);
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
