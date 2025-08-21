import React from "react";

import {
  IMdmStatusCardData,
  MDM_ENROLLMENT_STATUS_UI_MAP,
} from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { MDM_STATUS_TOOLTIP } from "utilities/constants";
import { CellProps, Column } from "react-table";
import { INumberCellProps } from "interfaces/datatable_config";

type IMdmStatusTableConfig = Column<IMdmStatusCardData>;
type IMdmStatusCellProps = CellProps<
  IMdmStatusCardData,
  IMdmStatusCardData["status"]
>;
type IHostCountCellProps = INumberCellProps<IMdmStatusCardData>;
type IViewAllHostsLinkProps = CellProps<IMdmStatusCardData>;

export const generateStatusTableHeaders = (
  teamId?: number
): IMdmStatusTableConfig[] => [
  {
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: ({ cell: { value: status } }: IMdmStatusCellProps) =>
      !MDM_STATUS_TOOLTIP[status] ? (
        <TextCell value={status} />
      ) : (
        <TooltipWrapper tipContent={MDM_STATUS_TOOLTIP[status]}>
          {MDM_ENROLLMENT_STATUS_UI_MAP[status].displayName}
        </TooltipWrapper>
      ),
    sortType: "hasLength",
  },
  {
    Header: "Hosts",
    disableSortBy: true,
    accessor: "hosts",
    Cell: (cellProps: IHostCountCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    Header: "",
    id: "view-all-hosts",
    disableSortBy: true,
    disableGlobalFilter: true,
    Cell: (cellProps: IViewAllHostsLinkProps) => {
      return (
        <ViewAllHostsLink
          queryParams={{
            mdm_enrollment_status:
              MDM_ENROLLMENT_STATUS_UI_MAP[cellProps.row.original.status]
                .filterValue,
            team_id: teamId,
          }}
          className="mdm-solution-link"
          platformLabelId={cellProps.row.original.selectedPlatformLabelId}
          rowHover
        />
      );
    },
  },
];

const enhanceStatusData = (
  statusData: IMdmStatusCardData[],
  selectedPlatformLabelId?: number
): IMdmStatusCardData[] => {
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
): IMdmStatusCardData[] => {
  if (!statusData) {
    return [];
  }
  return [...enhanceStatusData(statusData, selectedPlatformLabelId)];
};
