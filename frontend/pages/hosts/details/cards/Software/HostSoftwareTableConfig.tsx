import React from "react";
import { InjectedRouter } from "react-router";
import { CellProps, Column } from "react-table";

import { IHostSoftware, SOURCE_TYPE_CONVERSION } from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";

import IconCell from "pages/SoftwarePage/components/IconCell";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import VulnerabilitiesCell from "pages/SoftwarePage/components/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/SoftwareTitlesTableConfig";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import InstallStatusCell from "./InstallStatusCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledStatusCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["status"]
>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;

const formatSoftwareType = (source: string) => {
  const DICT = SOURCE_TYPE_CONVERSION;
  return DICT[source] || "Unknown";
};

interface ISoftwareTableHeadersProps {
  deviceUser?: boolean;
  router?: InjectedRouter;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  router,
}: ISoftwareTableHeadersProps): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { id, name, source } = cellProps.row.original;

        const softwareTitleDetailsPath = PATHS.SOFTWARE_TITLE_DETAILS(
          id.toString()
        );

        return (
          <SoftwareNameCell
            name={name}
            source={source}
            path={softwareTitleDetailsPath}
            router={router}
          />
        );
      },
    },
    {
      Header: "Install status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps: IInstalledStatusCellProps) => {
        const { original } = cellProps.row;
        const { value } = cellProps.cell;
        return value ? (
          <InstallStatusCell
            status={value}
            packageToInstall={original.package_available_for_install}
            installedAt={original.last_install?.installed_at}
          />
        ) : null;
      },
    },
    {
      Header: "Version",
      disableSortBy: true,
      // we use function as accessor because we have two columns that
      // need to access the same data. This is not supported with a string
      // accessor.
      accessor: (originalRow) => originalRow.installed_versions,
      Cell: (cellProps: IInstalledVersionsCellProps) => {
        return <VersionCell versions={cellProps.cell.value} />;
      },
    },
    {
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} formatter={formatSoftwareType} />
      ),
    },
    {
      Header: "Vulnerabilities",
      accessor: (originalRow) => originalRow.installed_versions,
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        const vulnerabilities = getVulnerabilities(cellProps.cell.value ?? []);
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders };
