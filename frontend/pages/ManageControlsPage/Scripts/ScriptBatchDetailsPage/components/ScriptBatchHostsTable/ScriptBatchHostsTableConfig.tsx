import React from "react";

import { ScriptBatchHostStatus } from "interfaces/script";
import { IScriptBatchHostResult } from "services/entities/scripts";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { CellProps, Column } from "react-table";
import TooltipTruncatedText from "components/TooltipTruncatedText";

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
      // TODO - make link to host details on click
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.row.original.display_name} />
      ),
    },
  ];

  if (["ran", "errored"].includes(hostStatus)) {
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
        // Header: (cellProps: ITableHeaderProps) => (
        //   <HeaderCell
        //     value="Host name"
        //     isSortedDesc={cellProps.column.isSortedDesc}
        //   />
        // ),
        disableSortBy: true,
        accessor: "script_output_preview",
        Cell: (cellProps: any) => <ScriptOutputCell {...cellProps} />,
      },
    ]);
  }
  // columns.push({
  //   Header: "",
  //   id: "view-script-details",
  //   disableSortBy: true,
  //   Cell: <ViewAllHostsLink customText="View script details" rowHover noLink />,
  // });

  return columns;
};

export default generateColumnConfigs;
