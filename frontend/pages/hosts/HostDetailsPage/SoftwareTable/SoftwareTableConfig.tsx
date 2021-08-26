import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import IconTooltipCell from "components/TableContainer/DataTable/IconTooltipCell";
import { ISoftware } from "interfaces/software";
import { IVulnerability } from "interfaces/vulnerability";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: ISoftware;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface ISoftwareTableData extends ISoftware {
  type: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Vulnerabilities",
      Header: "",
      disableSortBy: true,
      accessor: "vulnerabilitiesTooltip",
      Cell: (cellProps) => <IconTooltipCell value={cellProps.cell.value} />,
    },
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Type",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "type",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Installed version",
      Header: "Installed version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];
};

const TYPE_CONVERSION: Record<string, string> = {
  apt_sources: "Package (APT)",
  deb_packages: "Package (deb)",
  portage_packages: "Package (Portage)",
  rpm_packages: "Package (RPM)",
  yum_sources: "Package (YUM)",
  npm_packages: "Package (NPM)",
  atom_packages: "Package (Atom)",
  python_packages: "Package (Python)",
  apps: "Application (macOS)",
  chrome_extensions: "Browser plugin (Chrome)",
  firefox_addons: "Browser plugin (Firefox)",
  safari_extensions: "Browser plugin (Safari)",
  homebrew_packages: "Package (Homebrew)",
  programs: "Program (Windows)",
  ie_extensions: "Browser plugin (IE)",
  chocolatey_packages: "Package (Chocolatey)",
  pkg_packages: "Package (pkg)",
};

const generateTooltip = (vulnerabilities: IVulnerability[]): string | null => {
  if (!vulnerabilities) {
    // Uncomment to test tooltip rendering:
    // return "0 vulnerabilities detected";
    return null;
  }

  const vulText =
    vulnerabilities.length === 1 ? "vulnerability" : "vulnerabilities";

  return `${vulnerabilities.length} ${vulText} detected`;
};

const enhanceSoftwareData = (software: ISoftware[]): ISoftwareTableData[] => {
  return Object.values(software).map((softwareItem) => {
    return {
      id: softwareItem.id,
      name: softwareItem.name,
      source: softwareItem.source,
      type: TYPE_CONVERSION[softwareItem.source] || "Unknown",
      version: softwareItem.version,
      vulnerabilities: softwareItem.vulnerabilities,
      vulnerabilitiesTooltip: generateTooltip(softwareItem.vulnerabilities),
    };
  });
};

const generateDataSet = (software: ISoftware[]): ISoftwareTableData[] => {
  // Cannot pass undefined to enhanceSoftwareData
  if (!software) {
    return software;
  }

  return [...enhanceSoftwareData(software)];
};

export { generateTableHeaders, generateDataSet };
