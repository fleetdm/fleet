import React from "react";

import { IMdmStatusCardData } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";

interface IMdmStatusData extends IMdmStatusCardData {
  selectedPlatformLabelId?: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMdmStatusData;
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

const statusTableHeaders = [
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
MDM was turned on automatically using Apple Automated Device Enrollment (DEP) or Windows Autopilot. Administrators can block end users from turning MDM off.
                </span>
              `;
        }
        if (status === "On (manual)") {
          return `
                <span>
                  MDM was turned on manually. End users can turn MDM off.
                </span>
              `;
        }
        return `
                <span>
                  Hosts ordered via Apple Business Manager <br />
                  (ABM). These will automatically enroll to Fleet <br />
                  and turn on MDM when they’re unboxed.
                </span>
              `;
      };

      if (cellProps.cell.value === "Off") {
        return <TextCell value={cellProps.cell.value} />;
      }
      return (
        <span className="name-container">
          <TooltipWrapper
            position="top"
            tipContent={tooltipText(cellProps.cell.value)}
          >
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
          case "Pending":
            return "pending";
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

export const generateStatusTableHeaders = (): IDataColumn[] => {
  return statusTableHeaders;
};

const enhanceStatusData = (
  statusData: IMdmStatusCardData[],
  selectedPlatformLabelId?: number
): IMdmStatusData[] => {
  return Object.values(statusData).map((data) => {
    return {
      ...data,
      selectedPlatformLabelId,
    };
  });
};

export const generateStatusDataSet = (
  statusData: IMdmStatusCardData[] | null,
  selectedPlatformLabelId?: number
): IMdmStatusData[] => {
  if (!statusData) {
    return [];
  }
  return [...enhanceStatusData(statusData, selectedPlatformLabelId)];
};
