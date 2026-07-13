import React from "react";
import { CellProps, Column } from "react-table";

import PATHS from "router/paths";
import { IPolicyAutomationActivity } from "interfaces/policy";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  getAutomationRunDisplayName,
  getAutomationStatusIconName,
  getDetailOutputText,
} from "./helpers";

type ITableConfig = Column<IPolicyAutomationActivity>;
type ITableHeaderProps = IHeaderProps<IPolicyAutomationActivity>;
type ITableStringCellProps = IStringCellProps<IPolicyAutomationActivity>;
type ICellProps = CellProps<IPolicyAutomationActivity>;

const generateColumnConfigs = (
  baseClass: string,
  onShowDetails: (activity: IPolicyAutomationActivity) => void
): ITableConfig[] => [
  {
    Header: (cellProps: ITableHeaderProps) => (
      <HeaderCell
        value="Automation"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    id: "activity_type",
    accessor: (row) => row.type,
    Cell: (cellProps: ICellProps) => {
      const activity = cellProps.row.original;
      return (
        <div className={`${baseClass}__automation-cell`}>
          <Icon name={getAutomationStatusIconName(activity.status)} />
          <TooltipTruncatedText value={getAutomationRunDisplayName(activity)} />
        </div>
      );
    },
  },
  {
    Header: "Host",
    disableSortBy: true,
    id: "host_display_name",
    accessor: "host_display_name",
    Cell: (cellProps: ITableStringCellProps) => {
      const { host_id, host_display_name } = cellProps.row.original;
      if (!host_display_name) {
        // Host was deleted — no link target.
        return <TextCell value="Host deleted" grey italic />;
      }
      return (
        <LinkCell
          value={host_display_name}
          path={PATHS.HOST_DETAILS(host_id)}
          customOnClick={(e) => e.stopPropagation()}
        />
      );
    },
  },
  {
    Header: (cellProps: ITableHeaderProps) => (
      <HeaderCell value="Time" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    id: "created_at",
    accessor: "created_at",
    Cell: (cellProps: ICellProps) => (
      <TextCell
        value={
          <HumanTimeDiffWithDateTip
            timeString={cellProps.row.original.created_at}
          />
        }
      />
    ),
  },
  {
    Header: "Details",
    disableSortBy: true,
    id: "details",
    accessor: (row) => row.id,
    Cell: (cellProps: ICellProps) => {
      const activity = cellProps.row.original;
      const primaryText = getDetailOutputText(activity);
      return (
        <Button
          className={`${baseClass}__details-cell`}
          variant="inverse"
          onClick={() => onShowDetails(activity)}
        >
          <span className={`${baseClass}__details-text`}>
            {primaryText || "---"}
          </span>
          <Icon
            name="info-outline"
            className="row-hover-button"
            color="ui-fleet-black-50"
          />
        </Button>
      );
    },
  },
];

export default generateColumnConfigs;
