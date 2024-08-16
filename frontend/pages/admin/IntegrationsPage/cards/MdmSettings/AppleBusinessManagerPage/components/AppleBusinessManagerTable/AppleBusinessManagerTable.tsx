import React from "react";

import { IMdmAbmToken } from "interfaces/mdm";

import TableContainer from "components/TableContainer";

import { generateTableConfig } from "./AppleBusinessManagerTableConfig";

const baseClass = "apple-business-manager-table";

interface IAppleBusinessManagerTableProps {
  abmTokens: IMdmAbmToken[];
}

const AppleBusinessManagerTable = ({
  abmTokens,
}: IAppleBusinessManagerTableProps) => {
  const onSelectAction = (action: string, original: IMdmAbmToken) => {
    console.log(action, original);
  };

  const tableConfig = generateTableConfig(onSelectAction);

  return (
    <TableContainer<IMdmAbmToken>
      columnConfigs={tableConfig}
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
