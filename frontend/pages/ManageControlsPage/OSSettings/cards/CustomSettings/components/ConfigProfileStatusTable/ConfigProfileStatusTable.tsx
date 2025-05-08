import React from "react";

import { IGetConfigProfileStatusResponse } from "services/entities/config_profiles";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import {
  generateTableConfig,
  generateTableData,
} from "./ConfigProfileStatusTableConfig";

const baseClass = "config-profile-status-table";

interface IConfigProfileStatusTableProps {
  teamId: number;
  uuid: string;
  profileStatus: IGetConfigProfileStatusResponse;
}

const ConfigProfileStatusTable = ({
  teamId,
  uuid,
  profileStatus,
}: IConfigProfileStatusTableProps) => {
  const columnConfigs = generateTableConfig(teamId, uuid, profileStatus, () => {
    console.log("Resend clicked");
  });
  const tableData = generateTableData(profileStatus);

  return (
    <TableContainer
      className={baseClass}
      columnConfigs={columnConfigs}
      data={tableData}
      isLoading={false}
      emptyComponent={() => <EmptyTable />}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      manualSortBy
      disableTableHeader
      disablePagination
      disableCount
      hideFooter
    />
  );
};

export default ConfigProfileStatusTable;
