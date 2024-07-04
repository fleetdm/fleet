import React from "react";

import { IMdmStatusCardData, MDM_ENROLLMENT_STATUS } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { MDM_STATUS_TOOLTIP } from "utilities/constants";

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
  disableGlobalFilter?: boolean;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

export const generateStatusTableHeaders = (teamId?: number): IDataColumn[] => [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: ({ cell: { value: status } }: IStringCellProps) =>
      !MDM_STATUS_TOOLTIP[status] ? (
        <TextCell value={status} />
      ) : (
        <TooltipWrapper tipContent={MDM_STATUS_TOOLTIP[status]}>
          {status}
        </TooltipWrapper>
      ),
    sortType: "hasLength",
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
            mdm_enrollment_status:
              MDM_ENROLLMENT_STATUS[cellProps.row.original.status],
            team_id: teamId,
          }}
          className="mdm-solution-link"
          platformLabelId={cellProps.row.original.selectedPlatformLabelId}
        />
      );
    },
    disableHidden: true,
  },
];

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
