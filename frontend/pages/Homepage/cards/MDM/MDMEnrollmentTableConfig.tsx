import React from "react";
import { Link } from "react-router";

import { IDataTableMdmFormat } from "interfaces/macadmins";

import PATHS from "router/paths";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import Chevron from "../../../../../assets/images/icon-chevron-right-9x6@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IDataTableMdmFormat;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
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

const enrollmentTableHeaders = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps: IStringCellProps) => {
      const tooltipText = (status: string): string => {
        if (status === "Enrolled (automatic)") {
          return `
                <span>
                  Hosts automatically enrolled to an MDM solution <br/>
                  the first time the host is used. Administrators <br />
                  might have a higher level of control over these <br />
                  hosts.
                </span>
              `;
        }
        return `
                <span>
                  Hosts manually enrolled to an MDM solution by a<br />
                  user or administrator.
                </span>
              `;
      };

      if (cellProps.cell.value === "Unenrolled") {
        return <TextCell value={cellProps.cell.value} />;
      }
      return (
        <span className="name-container">
          <TooltipWrapper tipContent={tooltipText(cellProps.cell.value)}>
            {cellProps.cell.value}
          </TooltipWrapper>
        </span>
      );
    },
    sortType: "caseInsensitive",
  },
  {
    title: "Hosts",
    Header: "Hosts",
    disableSortBy: true,
    accessor: "hosts",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "",
    Header: "",
    disableSortBy: true,
    disableGlobalFilter: true,
    accessor: "linkToFilteredHosts",
    Cell: (cellProps: IStringCellProps) => {
      const statusParam = () => {
        switch (cellProps.row.original.status) {
          case "Enrolled (automatic)":
            return "automatic";
          case "Enrolled (manual)":
            return "manual";
          default:
            return "unenrolled";
        }
      };
      return (
        <Link
          to={`${PATHS.MANAGE_HOSTS}?mdm_enrollment_status=${statusParam()}`}
          className={`mdm-solution-link`}
        >
          View all hosts{" "}
          <img alt="link to hosts filtered by MDM solution" src={Chevron} />
        </Link>
      );
    },
    disableHidden: true,
  },
];

const generateEnrollmentTableHeaders = (): IDataColumn[] => {
  return enrollmentTableHeaders;
};

export default generateEnrollmentTableHeaders;
