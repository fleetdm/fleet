import React from "react";
import { Link } from "react-router";
import ReactTooltip from "react-tooltip";

import PATHS from "router/paths";
import { formatSoftwareType, ISoftware } from "interfaces/software";
import { IVulnerability } from "interfaces/vulnerability";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../assets/images/icon-chevron-right-blue-16x16@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: number | string | IVulnerability[];
  };
  row: {
    original: ISoftware;
  };
}
interface IStringCellProps extends ICellProps {
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
    value: IVulnerability[];
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

const condense = (vulnerabilities: IVulnerability[]): string[] => {
  const condensed =
    (vulnerabilities?.length &&
      vulnerabilities
        .slice(-3)
        .map((v) => v.cve)
        .reverse()) ||
    [];
  return vulnerabilities.length > 3
    ? condensed.concat(`+${vulnerabilities.length - 3} more`)
    : condensed;
};

const softwareTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: IStringCellProps): JSX.Element => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: IStringCellProps): JSX.Element => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Type",
    Header: "Type",
    disableSortBy: true,
    accessor: "source",
    Cell: (cellProps: IStringCellProps): JSX.Element => (
      <TextCell formatter={formatSoftwareType} value={cellProps.cell.value} />
    ),
  },
  {
    title: "Vulnerabilities",
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps): JSX.Element => {
      const vulnerabilities = cellProps.cell.value || [];
      const tooltipText = condense(vulnerabilities)?.map((value) => {
        return (
          <span key={`vuln_${value}`}>
            {value}
            <br />
          </span>
        );
      });

      if (!vulnerabilities?.length) {
        return <span className="vulnerabilities text-muted">---</span>;
      }
      return (
        <>
          <span
            className={`vulnerabilities ${
              vulnerabilities.length > 1 ? "text-muted" : ""
            }`}
            data-tip
            data-for={`vulnerabilities__${cellProps.row.original.id.toString()}`}
            data-tip-disable={vulnerabilities.length <= 1}
          >
            {vulnerabilities.length === 1
              ? vulnerabilities[0].cve
              : `${vulnerabilities.length} vulnerabilities`}
          </span>
          <ReactTooltip
            place="top"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id={`vulnerabilities__${cellProps.row.original.id.toString()}`}
            data-html
          >
            <span className={`vulnerabilities tooltip__tooltip-text`}>
              {tooltipText}
            </span>
          </ReactTooltip>
        </>
      );
    },
  },
  {
    title: "Hosts",
    Header: (cellProps: IHeaderProps): JSX.Element => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: INumberCellProps): JSX.Element => (
      <span className="hosts-cell__wrapper">
        <span className="hosts-cell__count">
          <TextCell value={cellProps.cell.value} />
        </span>
        <span className="hosts-cell__link">
          <Link
            to={`${PATHS.MANAGE_HOSTS}?software_id=${cellProps.row.original.id}`}
            className="software-link"
          >
            <span className="link-text">View all hosts</span>
            <img alt="link to hosts filtered by software ID" src={Chevron} />
          </Link>
        </span>
      </span>
    ),
  },
];

export default softwareTableHeaders;
