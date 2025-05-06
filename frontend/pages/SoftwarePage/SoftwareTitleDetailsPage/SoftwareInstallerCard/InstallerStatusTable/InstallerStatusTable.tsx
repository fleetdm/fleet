import React from "react";
import classnames from "classnames";

import TableContainer from "components/TableContainer";
import {
  ISoftwarePackageStatus,
  ISoftwareAppStoreAppStatus,
} from "interfaces/software";
import generateSoftwareTitleDetailsTableConfig from "./InstallerStatusTableConfig";

const baseClass = "installer-status-table";

interface ISoftwareDetailsWidget {
  className?: string;
  softwareId: number;
  teamId?: number;
  status: ISoftwarePackageStatus | ISoftwareAppStoreAppStatus;
  isLoading?: boolean;
}
const InstallerStatusTable = ({
  className,
  softwareId,
  teamId,
  status,
  isLoading = false,
}: ISoftwareDetailsWidget) => {
  const classNames = classnames(baseClass, className);

  const softwareStatusHeaders = generateSoftwareTitleDetailsTableConfig({
    baseClass: classNames,
    softwareId,
    teamId,
  });

  console.log("softwareStatusHeaders", softwareStatusHeaders);
  console.log("status", status);
  return (
    <TableContainer
      className={baseClass}
      isLoading={isLoading}
      columnConfigs={softwareStatusHeaders}
      data={[status]}
      disablePagination
      disableMultiRowSelect
      emptyComponent={() => <></>}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      disableHighlightOnHover
      hideFooter
    />
  );
};

export default InstallerStatusTable;
