import React from "react";
import { Link } from "react-router"; // TODO: Enable after manage hosts page has been updated to filter hosts by software id
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";
// import distanceInWordsToNow from "date-fns/distance_in_words_to_now"; // TODO: Enable after backend has been updated to provide last_opened_at

import PATHS from "router/paths"; // TODO: Enable after manage hosts page has been updated to filter hosts by software id
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { ISoftware } from "interfaces/software";
import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";
import Chevron from "../../../../../assets/images/icon-chevron-blue-16x16@2x.png"; // TODO: Enable after manage hosts page has been updated to filter hosts by software id

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

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateSoftwareTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Vulnerabilities",
      Header: "",
      disableSortBy: true,
      accessor: "vulnerabilities",
      Cell: (cellProps) => {
        const vulnerabilities = cellProps.cell.value;
        if (isEmpty(vulnerabilities)) {
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
              <img alt="software vulnerabilities" src={IssueIcon} />
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
                <img alt="bundle identifier" src={QuestionIcon} />
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
    // TODO: Enable after backend has been updated to provide last_opened_at
    // {
    //   title: "Last used",
    //   Header: (cellProps) => (
    //     <HeaderCell
    //       value={cellProps.column.title}
    //       isSortedDesc={cellProps.column.isSortedDesc}
    //     />
    //   ),
    //   accessor: "last_opened_at",
    //   Cell: (cellProps) => {
    //     const lastUsed = isNaN(Date.parse(cellProps.cell.value))
    //       ? "Unavailable"
    //       : `${distanceInWordsToNow(Date.parse(cellProps.cell.value))} ago`;
    //     return (
    //       <span
    //         className={
    //           lastUsed === "Unavailable"
    //             ? "software-last-used-muted"
    //             : "software-last-used"
    //         }
    //       >
    //         {lastUsed}
    //       </span>
    //     );
    //   },
    //   sortType: "dateStrings",
    // },
    // TODO: Enable after manage hosts page has been updated to filter hosts by software id
    {
      title: "",
      Header: "",
      disableSortBy: true,
      accessor: "linkToFilteredHosts",
      Cell: (cellProps) => {
        return (
          <Link
            to={`${
              PATHS.MANAGE_HOSTS
            }?software_id=${cellProps.row.original.id.toString()}`}
            className={`software-link`}
          >
            <img alt="link to hosts filtered by software ID" src={Chevron} />
          </Link>
        );
      },
      disableHidden: true,
    },
  ];
};

const enhanceSoftwareData = (software: ISoftware[]): ISoftwareTableData[] => {
  return Object.values(software).map((softwareItem) => {
    return {
      ...softwareItem,
      // linkToFilteredHosts: `${PATHS.MANAGE_HOSTS}?software_id=${softwareItem.id}`,
      type: TYPE_CONVERSION[softwareItem.source] || "Unknown",
    };
  });
};

const generateSoftwareDataSet = (
  software: ISoftware[]
): ISoftwareTableData[] => {
  // Cannot pass undefined to enhanceSoftwareData
  if (!software) {
    return software;
  }

  return [...enhanceSoftwareData(software)];
};

export { generateSoftwareTableHeaders, generateSoftwareDataSet };
