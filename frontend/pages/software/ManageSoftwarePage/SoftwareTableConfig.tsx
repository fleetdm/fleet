import React from "react";
import { Link } from "react-router";
import ReactTooltip from "react-tooltip";

import PATHS from "router/paths";
import { ISoftware } from "interfaces/software";
import { IVulnerability } from "interfaces/vulnerability";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../assets/images/icon-chevron-right-blue-16x16@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: ISoftware;
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
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

const softwareTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Vulnerabilities",
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: ICellProps) => {
      const vulnerabilities: IVulnerability[] = cellProps.cell.value;
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
              {vulnerabilities.map((v) => (
                <span key={v.cve}>
                  {v.cve}
                  <br />
                </span>
              ))}
            </span>
          </ReactTooltip>
        </>
      );
    },
  },
  {
    title: "Hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Actions",
    Header: "",
    disableSortBy: true,
    accessor: "id",
    Cell: (cellProps: ICellProps) => {
      return (
        <Link
          to={`${PATHS.MANAGE_HOSTS}?software_id=${cellProps.cell.value}`}
          className="software-link"
        >
          <span className="link-text">View all hosts</span>
          <img alt="link to hosts filtered by software ID" src={Chevron} />
        </Link>
      );
    },
  },
];

const generateTableHeaders = (): IDataColumn[] => {
  return softwareTableHeaders;
};

export default generateTableHeaders;
