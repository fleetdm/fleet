import React from "react";
import { CellProps, Column } from "react-table";

import {
  IHostSoftware,
  SoftwareSource,
  SOURCE_TYPE_CONVERSION,
} from "interfaces/software";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";

import VulnerabilitiesCell from "pages/SoftwarePage/components/VulnerabilitiesCell";
import VersionCell from "pages/SoftwarePage/components/VersionCell";
import { getVulnerabilities } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/SoftwareTitlesTableConfig";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";

type ISoftwareTableConfig = Column<IHostSoftware>;
type ITableHeaderProps = IHeaderProps<IHostSoftware>;
type ITableStringCellProps = IStringCellProps<IHostSoftware>;
type IInstalledVersionsCellProps = CellProps<
  IHostSoftware,
  IHostSoftware["installed_versions"]
>;
type IVulnerabilitiesCellProps = IInstalledVersionsCellProps;

const formatSoftwareType = (source: SoftwareSource) => {
  const DICT = SOURCE_TYPE_CONVERSION;
  return DICT[source] || "Unknown";
};

export const generateSoftwareTableData = (
  software: IHostSoftware[]
): IHostSoftware[] => {
  return software;
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = (): ISoftwareTableConfig[] => {
  const tableHeaders: ISoftwareTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, source } = cellProps.row.original;
        return <SoftwareNameCell name={name} source={source} />;
      },
      sortType: "caseInsensitive",
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
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Type" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      disableSortBy: false,
      disableGlobalFilter: true,
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
    {
      Header: "",
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: "id",
      disableSortBy: true,
      Cell: (cellProps) => {
        return (
          <span className="link">
            <span className="link-text">Show details</span>
          </span>
        );
      },
    },
  ];

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
