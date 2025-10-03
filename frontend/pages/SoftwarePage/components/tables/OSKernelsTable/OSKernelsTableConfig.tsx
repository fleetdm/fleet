import React from "react";

import { getPathWithQueryParams } from "utilities/url";

import { CellProps } from "react-table";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IOperatingSystemKernels } from "interfaces/operating_system";

import PATHS from "router/paths";

import ViewAllHostsLink from "components/ViewAllHostsLink";
import TooltipWrapper from "components/TooltipWrapper";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import VulnerabilitiesCell from "../VulnerabilitiesCell";

interface IOsKernelsTableConfigProps {
  teamId?: number;
  osName: string;
  osVersion: string;
}

type IHostCountCellProps = INumberCellProps<IOperatingSystemKernels>;
type IVersionCellProps = IStringCellProps<IOperatingSystemKernels>;
type IViewAllHostsLinkProps = CellProps<IOperatingSystemKernels>;
type IVulnCellProps = CellProps<IOperatingSystemKernels, string[] | null>;

const generateTableConfig = ({
  teamId,
  osName,
  osVersion,
}: IOsKernelsTableConfigProps) => {
  const tableHeaders = [
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: IVersionCellProps): JSX.Element => {
        if (!cellProps.cell.value) {
          // renders desired empty state
          return <TextCell />;
        }
        const { id } = cellProps.row.original;
        const softwareVersionDetailsPath = getPathWithQueryParams(
          PATHS.SOFTWARE_VERSION_DETAILS(id.toString()),
          { team_id: teamId }
        );

        return (
          <LinkCell
            className="name-link"
            path={softwareVersionDetailsPath}
            value={cellProps.cell.value}
          />
        );
      },
    },
    {
      title: "Vulnerabilities",
      Header: "Vulnerabilities",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnCellProps): JSX.Element => {
        return <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />;
      },
    },
    {
      title: "Hosts",
      Header: () => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                Linux hosts may have multiple kernels
                <br /> installed. Containers do not have their
                <br /> own kernel.
              </>
            }
            className="status-header"
          >
            Hosts
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithToolTip} disableSortBy />;
      },
      disableSortBy: true,
      accessor: "hosts_count",
      Cell: (cellProps: IHostCountCellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps: IViewAllHostsLinkProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllHostsLink
                queryParams={{
                  software_version_id: cellProps.row.original.id,
                  team_id: teamId,
                  os_name: osName,
                  os_version: osVersion,
                }}
                className="software-link"
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];

  return tableHeaders;
};

export default generateTableConfig;
