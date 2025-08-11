import React from "react";

import { getPathWithQueryParams } from "utilities/url";
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
}

interface ICellProps {
  cell: {
    value: number | string | string[];
  };
  row: {
    original: IOperatingSystemKernels;
    index: number;
  };
}

interface IVersionCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface INumberCellProps extends ICellProps {
  cell: {
    value: number;
  };
}

interface IVulnCellProps extends ICellProps {
  cell: {
    value: string[];
  };
}

const generateTableConfig = ({ teamId }: IOsKernelsTableConfigProps) => {
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
      Cell: (cellProps: INumberCellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllHostsLink
                queryParams={{
                  software_version_id: cellProps.row.original.id,
                  team_id: teamId,
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
