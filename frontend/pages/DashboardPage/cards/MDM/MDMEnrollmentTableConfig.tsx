import React from "react";

import { formatMdmStatusForUrl, IMdmEnrollmentCardData } from "interfaces/mdm";

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
  manualSortBy?: boolean;
}

// TODO: Consider implementing a global helper that could be used for various MDM tooltips around
// the UI (e.g., managage hosts page, host details page, dashboard, etc.)
const mdmStatusTooltipText = (status: string): string => {
  switch (status) {
    case "Pending":
      return `
        <span>
          Hosts ordered via Apple Business Manager (ABM). <br />
          These will automatically enroll to Fleet and turn on <br />
          MDM when they&apos;re unboxed.
        <span />
      `;
    case "On (automatic)":
      return `
        <span>
          MDM was turned on automatically using Apple <br />
          Automated Device Enrollment (DEP) or Windows <br />
          Autopilot. Administrators can block end users from <br />
          turning MDM off.
        <span />
      `;
    case "On (manual)":
      return `
        <span>
          MDM was turned on manually. End users can turn <br />
          MDM off.
        <span />
      `;
    // TODO: Figma doesn't include a tooltip for this row on the dashboard card, but there is a
    // tooltip on the label filter pill for the manage hosts page. Confirm what is intended.
    // case "Off":
    //   return `
    //     <span>
    //       Hosts not enrolled to an MDM solution.
    //     </ span>
    //   `;
    default:
      return "";
  }
};

const enrollmentTableHeaders = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps: IStringCellProps) => {
      const tipContent = mdmStatusTooltipText(cellProps.cell.value);
      if (!tipContent) {
        return <TextCell value={cellProps.cell.value} />;
      }

      return (
        <span className="name-container">
          <TooltipWrapper tipContent={tipContent}>
            {cellProps.cell.value}
          </TooltipWrapper>
        </span>
      );
    },
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
      return (
        <ViewAllHostsLink
          queryParams={{
            mdm_enrollment_status: formatMdmStatusForUrl(
              cellProps.row.original.status
            ),
          }}
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
  return enrollmentData.map((data) => {
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
