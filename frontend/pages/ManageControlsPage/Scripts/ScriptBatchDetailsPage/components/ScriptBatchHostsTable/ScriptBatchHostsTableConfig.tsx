import React from "react";

import PATHS from "router/paths";

import {
  SCRIPT_BATCH_HOST_EXECUTED_STATUSES,
  SCRIPT_BATCH_HOST_NOT_EXECUTED_STATUSES,
  ScriptBatchHostStatus,
} from "interfaces/script";
import { IScriptBatchHostResult } from "services/entities/scripts";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { CellProps, Column } from "react-table";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import LinkCell from "components/TableContainer/DataTable/LinkCell";

type IScriptBatchHostsTableConfig = Column<IScriptBatchHostResult>;
type ITableHeaderProps = IHeaderProps<IScriptBatchHostResult>;
type ITableStringCellProps = IStringCellProps<IScriptBatchHostResult>;
type ITimeCellProps = CellProps<IScriptBatchHostResult>;

const ScriptOutputCell = (cellProps: CellProps<IScriptBatchHostResult>) => {
  return (
    <span className="script-output-cell">
      <TooltipTruncatedText
        value={cellProps.row.original.script_output_preview}
      />
      <ViewAllHostsLink
        customContent="View script details"
        rowHover
        noLink
        responsive
      />
    </span>
  );
};

const generateColumnConfigs = (
  hostStatus: ScriptBatchHostStatus
): IScriptBatchHostsTableConfig[] => {
  let columns: IScriptBatchHostsTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Host name"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "display_name",
      Cell: (cellProps: ITableStringCellProps) => (
        <span className="host-name-cell">
          <LinkCell
            value={cellProps.row.original.display_name}
            path={PATHS.HOST_DETAILS(cellProps.row.original.id)}
            customOnClick={(e) => {
              e.stopPropagation();
            }}
          />
          {SCRIPT_BATCH_HOST_NOT_EXECUTED_STATUSES.includes(hostStatus) && (
            <ViewAllHostsLink
              customContent="View host details"
              rowHover
              noLink
              responsive
            />
          )}
        </span>
      ),
    },
  ];

  if (SCRIPT_BATCH_HOST_EXECUTED_STATUSES.includes(hostStatus)) {
    columns = columns.concat([
      {
        Header: (cellProps: ITableHeaderProps) => (
          <HeaderCell
            value="Time"
            disableSortBy={false}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        ),
        accessor: "script_executed_at",
        Cell: (cellProps: ITimeCellProps) => (
          <TextCell
            value={
              <HumanTimeDiffWithDateTip
                timeString={cellProps.row.original.script_executed_at ?? ""}
              />
            }
          />
        ),
      },
      {
        Header: "Script output",
        disableSortBy: true,
        accessor: "script_output_preview",
        Cell: (cellProps: any) => <ScriptOutputCell {...cellProps} />,
      },
    ]);
  }

  return columns;
};

export default generateColumnConfigs;
