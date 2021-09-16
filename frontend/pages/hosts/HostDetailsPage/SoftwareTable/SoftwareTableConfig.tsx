import React from "react";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { ISoftware } from "interfaces/software";
import { IVulnerability } from "interfaces/vulnerability";
import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

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
}

interface ISoftwareTableData extends ISoftware {
  type: string;
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

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Vulnerabilities",
      Header: "",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps) => {
        if (isEmpty(cellProps.cell.value)) {
          return <></>;
        }
        return (
          <>
            <span
              className={`software-vuln tooltip__tooltip-icon`}
              data-tip
              data-for={`software-vuln__${cellProps.row.original.id.toString()}`}
              data-tip-disable={false}
            >
              <img alt="Tooltip icon" src={IssueIcon} />
            </span>
            <ReactTooltip
              place="bottom"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id={`software-vuln__${cellProps.row.original.id.toString()}`}
              data-html
            >
              <span className={`tooltip__tooltip-text`}>
                {generateTooltip(cellProps.cell.value)}
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
      Cell: (cellProps) => {
        const { name, bundle_identifier } = cellProps.row.original;
        if (bundle_identifier) {
          return (
            <span>
              {name}
              <span
                className={`software-name tooltip__tooltip-icon`}
                data-tip
                data-for={`software-name__${cellProps.row.original.id.toString()}`}
                data-tip-disable={false}
              >
                <img alt="Tooltip icon" src={QuestionIcon} />
              </span>
              <ReactTooltip
                place="bottom"
                type="dark"
                effect="solid"
                backgroundColor="#3e4771"
                id={`software-name__${cellProps.row.original.id.toString()}`}
                data-html
              >
                <span className={`tooltip__tooltip-text`}>
                  <b>Bundle identifier: </b>
                  <br />
                  {bundle_identifier}
                </span>
              </ReactTooltip>
            </span>
          );
        }
        return <TextCell value={name} />;
      },
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
    {
      title: "Last used",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "last_opened_at",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
      sortType: "dateStringsAsc",
    },
  ];
};

const FAKEID = "foo.app";
const FAKEDATE = "2021-08-18T15:11:35Z";

const enhanceSoftwareData = (software: ISoftware[]): ISoftwareTableData[] => {
  return Object.values(software).map((softwareItem) => {
    let TIME = "unavailable";
    if (softwareItem.id % 3) {
      TIME = FAKEDATE;
    }
    if (softwareItem.id % 2) {
      TIME = new Date(Date.now()).toString();
    }
    return {
      ...softwareItem,
      bundle_identifier: FAKEID,
      last_opened_at: TIME,
      vulnerabilitiesTooltip: generateTooltip(softwareItem.vulnerabilities),
      type: TYPE_CONVERSION[softwareItem.source] || "Unknown",
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
