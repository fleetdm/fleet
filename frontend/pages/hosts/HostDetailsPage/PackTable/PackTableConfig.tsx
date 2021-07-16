import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IScheduledQuery } from "interfaces/scheduled_query";

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
    original: IScheduledQuery;
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

interface ISoftwareTableData extends IScheduledQuery {
  frequency: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generatePackTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Query name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Description",
      Header: "Description",
      accessor: "description",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: false,
      accessor: "frequency",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Last Run",
      Header: "Last Run",
      disableSortBy: true,
      accessor: "lastRun",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];
};

// const TYPE_CONVERSION: Record<string, string> = {
//   apt_sources: "Package (APT)",
//   deb_packages: "Package (deb)",
//   portage_packages: "Package (Portage)",
//   rpm_packages: "Package (RPM)",
//   yum_sources: "Package (YUM)",
//   npm_packages: "Package (NPM)",
//   atom_packages: "Package (Atom)",
//   python_packages: "Package (Python)",
//   apps: "Application (macOS)",
//   chrome_extensions: "Browser plugin (Chrome)",
//   firefox_addons: "Browser plugin (Firefox)",
//   safari_extensions: "Browser plugin (Safari)",
//   homebrew_packages: "Package (Homebrew)",
//   programs: "Program (Windows)",
//   ie_extensions: "Browser plugin (IE)",
//   chocolatey_packages: "Package (Chocolatey)",
//   pkg_packages: "Package (pkg)",
// };

// const generateTooltip = (vulnerabilities: IVulnerability[]): string | null => {
//   if (!vulnerabilities) {
//     // Uncomment to test tooltip rendering:
//     // return "0 vulnerabilities detected";
//     return null;
//   }

//   const vulText =
//     vulnerabilities.length === 1 ? "vulnerability" : "vulnerabilities";

//   return `${vulnerabilities.length} ${vulText} detected`;
// };

const enhancePackData = (query_stats: IScheduledQuery[]): IPackTable[] => {
  // return Object.values(software).map((softwareItem) => {
  //   return {
  //     id: softwareItem.id,
  //     name: softwareItem.name,
  //     source: softwareItem.source,
  //     type: TYPE_CONVERSION[softwareItem.source] || "Unknown",
  //     version: softwareItem.version,
  //     vulnerabilities: softwareItem.vulnerabilities,
  //     vulnerabilitiesTooltip: generateTooltip(softwareItem.vulnerabilities),
  //   };
  });
};

const generateDataSet = (query_stats: IScheduledQuery[]): IPackTable[] => {
  // Cannot pass undefined to enhanceSoftwareData
  if (!query_stats) {
    return query_stats;
  }

  return [...enhancePackData(query_stats)];
};

export { generatePackTableHeaders, generateDataSet };
