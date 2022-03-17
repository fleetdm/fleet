import React from "react";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import { ISoftware } from "interfaces/software";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import IssueIcon from "../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

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
  sortType?: string;
  // Filter can be used by react-table to render a filter input inside the column header
  Filter?: () => null | JSX.Element;
  filter?: string; // one of the enumerated `filterTypes` for react-table
  // (see https://github.com/tannerlinsley/react-table/blob/master/src/filterTypes.js)
  // or one of the custom `filterTypes` defined for the `useTable` instance (see `DataTable`)
}

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

const formatSoftwareType = (source: string) => {
  const DICT = TYPE_CONVERSION;
  return DICT[source] || "Unknown";
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateSoftwareTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Vulnerabilities",
      Header: "",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Filter: () => null, // input for this column filter outside of column header
      filter: "hasLength", // filters out rows where vulnerabilities has no length if filter value is `true`
      Cell: (cellProps) => {
        const vulnerabilities = cellProps.cell.value;
        if (isEmpty(vulnerabilities)) {
          return <></>;
        }
        return (
          <>
            <span
              className={`vulnerabilities tooltip__tooltip-icon`}
              data-tip
              data-for={`vulnerabilities__${cellProps.row.original.id.toString()}`}
              data-tip-disable={false}
            >
              <img alt="software vulnerabilities" src={IssueIcon} />
            </span>
            <ReactTooltip
              place="bottom"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id={`vulnerabilities__${cellProps.row.original.id.toString()}`}
              data-html
            >
              <span className={`vulnerabilities tooltip__tooltip-text`}>
                {vulnerabilities.length === 1
                  ? "1 vulnerability detected"
                  : `${vulnerabilities.length} vulnerabilities detected`}
              </span>
            </ReactTooltip>
          </>
        );
      },
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
      Filter: () => null, // input for this column filter is rendered outside of column header
      filter: "text", // filters name text based on the user's search query
      Cell: (cellProps) => {
        const { name, bundle_identifier } = cellProps.row.original;
        if (bundle_identifier) {
          return (
            <span className="name-container">
              <TooltipWrapper
                tipContent={`
                <span>
                  <b>Bundle identifier: </b>
                  <br />
                  ${bundle_identifier}
                </span>
              `}
              >
                {name}
              </TooltipWrapper>
            </span>
          );
        }
        return <TextCell value={name} />;
      },
      sortType: "caseInsensitive",
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
      accessor: "source",
      Cell: (cellProps) => (
        <TextCell value={cellProps.cell.value} formatter={formatSoftwareType} />
      ),
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

export default generateSoftwareTableHeaders;
