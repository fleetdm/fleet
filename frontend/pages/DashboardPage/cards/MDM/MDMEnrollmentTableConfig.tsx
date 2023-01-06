import React from "react";

import { IMdmEnrollmentCardData } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";

interface IMdmEnrollmentData extends IMdmEnrollmentCardData {
  selectedPlatformLabelId?: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMdmEnrollmentData;
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
        if (status === "On (automatic)") {
          return `
                <span>
                  Hosts automatically enrolled to an MDM solution <br />
                  using Apple Automated Device Enrollment (DEP) <br />
                  or Windows Autopilot. Administrators can block <br />
                  users from unenrolling these hosts from MDM.
                </span>
              `;
        }
        return `
                <span>
                  Hosts manually enrolled to an MDM solution. Users <br />
                  can unenroll these hosts from MDM.
                </span>
              `;
      };

      if (cellProps.cell.value === "Off") {
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
          case "On (automatic)":
            return "automatic";
          case "On (manual)":
            return "manual";
          default:
            return "unenrolled";
        }
      };
      return (
        <ViewAllHostsLink
          queryParams={{ mdm_enrollment_status: statusParam() }}
          className="mdm-solution-link"
          platformLabelId={cellProps.row.original.selectedPlatformLabelId}
        />
      );
    },
    disableHidden: true,
  },
];

export const generateEnrollmentTableHeaders = (): IDataColumn[] => {
  return enrollmentTableHeaders;
};

const enhanceEnrollmentData = (
  enrollmentData: IMdmEnrollmentCardData[],
  selectedPlatformLabelId?: number
): IMdmEnrollmentData[] => {
  return Object.values(enrollmentData).map((data) => {
    return {
      ...data,
      selectedPlatformLabelId,
    };
  });
};

export const generateEnrollmentDataSet = (
  enrollmentData: IMdmEnrollmentCardData[] | null,
  selectedPlatformLabelId?: number
): IMdmEnrollmentData[] => {
  if (!enrollmentData) {
    return [];
  }
  return [...enhanceEnrollmentData(enrollmentData, selectedPlatformLabelId)];
};
