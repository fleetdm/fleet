import React from "react";

import { ISoftwareTitleVersion } from "interfaces/software";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { generateResultsCountText } from "components/TableContainer/utilities/TableContainerUtils";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

interface ISoftwareTitleDetailsTableConfigProps {
  softwareId?: number;
  teamId?: number;
  baseClass?: string;
}
interface ICellProps {
  cell: {
    value: number;
  };
  row: {
    original: ISoftwareTitleVersion;
  };
}

interface IStatusDisplayOption {
  displayName: string;
  iconName: "success" | "pending-outline" | "error";
  tooltip: React.ReactNode;
}

// "pending" and "failed" each encompass both "_install" and "_uninstall" sub-statuses
type SoftwareInstallDisplayStatus = "installed" | "pending" | "failed";

const STATUS_DISPLAY_OPTIONS: Record<
  SoftwareInstallDisplayStatus,
  IStatusDisplayOption
> = {
  installed: {
    displayName: "Installed",
    iconName: "success",
    tooltip: (
      <>
        Software is installed on these hosts (install script finished
        <br />
        with exit code 0). Currently, if the software is uninstalled, the
        <br />
        &quot;Installed&quot; status won&apos;t be updated.
      </>
    ),
  },
  pending: {
    displayName: "Pending",
    iconName: "pending-outline",
    tooltip: (
      <>
        Fleet is installing/uninstalling or will
        <br />
        do so when the host comes online.
      </>
    ),
  },
  failed: {
    displayName: "Failed",
    iconName: "error",
    tooltip: (
      <>
        These hosts failed to install/uninstall software.
        <br />
        Click on a host to view error(s).
      </>
    ),
  },
};

const generateSoftwareTitleDetailsTableConfig = ({
  softwareId,
  teamId,
  baseClass,
}: ISoftwareTitleDetailsTableConfigProps) => {
  const tableHeaders = [
    {
      accessor: "installed",
      disableSortBy: true,
      title: "Installed",
      Header: () => {
        const displayData = STATUS_DISPLAY_OPTIONS.installed;
        const titleWithTooltip = (
          <TooltipWrapper
            position="top"
            tipContent={displayData.tooltip}
            underline={false}
            showArrow
            tipOffset={10}
          >
            <div className={`${baseClass}__status-title`}>
              <Icon name={displayData.iconName} />
              <div>{displayData.displayName}</div>
            </div>
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithTooltip} disableSortBy />;
      },
      Cell: (cellProps: ICellProps) => {
        return (
          <LinkCell
            value={generateResultsCountText("hosts", cellProps.cell.value)}
            path={getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
              software_title_id: softwareId,
              software_status: "installed",
              team_id: teamId,
            })}
          />
        );
      },
    },
    {
      accessor: "pending",
      disableSortBy: true,
      title: "Pending",
      Header: () => {
        const displayData = STATUS_DISPLAY_OPTIONS.pending;
        return (
          <TooltipWrapper
            position="top"
            tipContent={displayData.tooltip}
            underline={false}
            showArrow
            tipOffset={10}
          >
            <div className={`${baseClass}__status-title`}>
              <Icon name={displayData.iconName} />
              <div>{displayData.displayName}</div>
            </div>
          </TooltipWrapper>
        );
      },
      Cell: (cellProps: ICellProps) => {
        return (
          <LinkCell
            value={generateResultsCountText("hosts", cellProps.cell.value)}
            path={getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
              software_title_id: softwareId,
              software_status: "pending",
              team_id: teamId,
            })}
          />
        );
      },
    },
    {
      accessor: "failed",
      disableSortBy: true,
      title: "Failed",
      Header: () => {
        const displayData = STATUS_DISPLAY_OPTIONS.failed;
        return (
          <TooltipWrapper
            position="top"
            tipContent={displayData.tooltip}
            underline={false}
            showArrow
            tipOffset={10}
          >
            <div className={`${baseClass}__status-title`}>
              <Icon name={displayData.iconName} />
              <div>{displayData.displayName}</div>
            </div>
          </TooltipWrapper>
        );
      },
      Cell: (cellProps: ICellProps) => {
        return (
          <LinkCell
            value={generateResultsCountText("hosts", cellProps.cell.value)}
            path={getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
              software_title_id: softwareId,
              software_status: "failed",
              team_id: teamId,
            })}
          />
        );
      },
    },
  ];

  return tableHeaders;
};

export default generateSoftwareTitleDetailsTableConfig;
