import React from "react";
import classnames from "classnames";

import TableContainer from "components/TableContainer";
import {
  ISoftwarePackageStatus,
  ISoftwareAppStoreAppStatus,
} from "interfaces/software";
import generateSoftwareTitleDetailsTableConfig from "./InstallerStatusTableConfig";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "installer-status-table";

interface IInstallerStatusTableProps {
  className?: string;
  softwareId: number;
  teamId?: number;
  status: ISoftwarePackageStatus | ISoftwareAppStoreAppStatus;
  isLoading?: boolean;
  isScriptPackage?: boolean;
  isAndroidPlayStoreApp?: boolean;
}

const InstallerStatusTable = ({
  className,
  softwareId,
  teamId,
  status,
  isLoading = false,
  isScriptPackage = false,
  isAndroidPlayStoreApp = false,
}: IInstallerStatusTableProps) => {
  const classNames = classnames(baseClass, className);

  const softwareStatusHeaders = generateSoftwareTitleDetailsTableConfig({
    baseClass: classNames,
    softwareId,
    teamId,
    isScriptPackage,
    isAndroidPlayStoreApp,
  });

  const renderTableHelpText = isScriptPackage
    ? undefined
    : () => (
        <div>
          Installs for the current version, triggered by policy automations or{" "}
          <TooltipWrapper
            tipContent={
              <>
                On the <b>Host details</b> or{" "}
                <b>Fleet Desktop &gt; My device page.</b>
              </>
            }
          >
            manually
          </TooltipWrapper>
          .
        </div>
      );

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
      disableTableHeader
      renderTableHelpText={renderTableHelpText}
    />
  );
};

export default InstallerStatusTable;
