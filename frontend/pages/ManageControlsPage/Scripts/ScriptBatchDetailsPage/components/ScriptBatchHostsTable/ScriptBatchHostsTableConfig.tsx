import React from "react";
import { CellProps, Column } from "react-table";

import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import {
  SCRIPT_BATCH_HOST_EXECUTED_STATUSES,
  SCRIPT_BATCH_HOST_NOT_EXECUTED_STATUSES,
  ScriptBatchHostStatus,
} from "interfaces/script";
import PATHS from "router/paths";
import { IScriptBatchHostResult } from "services/entities/scripts";

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
        customText="View script details"
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
              customText="View host details"
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
        Cell: (cellProps: CellProps<IScriptBatchHostResult>) => (
          <ScriptOutputCell {...cellProps} />
        ),
      },
    ]);
  }

  return columns;
};

export default generateColumnConfigs;
