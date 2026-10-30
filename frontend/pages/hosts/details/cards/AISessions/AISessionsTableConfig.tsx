import React from "react";
import { Column } from "react-table";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IHostAISession } from "interfaces/ai_sessions";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { monthDayTimeFormat } from "utilities/date_format";

import {
  AI_SESSION_BACKFILL_MESSAGE,
  getAIAgentDisplay,
  getAISessionDuration,
  getAISessionUserAccount,
} from "./helpers";

const baseClass = "ai-sessions-table";

type IAISessionsTableConfig = Column<IHostAISession>;

// Duration and Events have no API sort key; User account mixes two columns,
// neither of which sorts the way it reads.
const generateTableConfig = (): IAISessionsTableConfig[] => [
  {
    accessor: "agent",
    Header: (cellProps) => (
      <HeaderCell value="Agent" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    Cell: (cellProps) => {
      const { name, iconName } = getAIAgentDisplay(cellProps.cell.value);
      return (
        <span className={`${baseClass}__agent`}>
          <SoftwareIcon name={iconName} />
          {name}
        </span>
      );
    },
  },
  {
    accessor: "started_at",
    Header: (cellProps) => (
      <HeaderCell
        value="Started"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    Cell: (cellProps) => (
      <TextCell
        value={
          cellProps.cell.value ? monthDayTimeFormat(cellProps.cell.value) : null
        }
        className="text-nowrap"
      />
    ),
  },
  {
    id: "duration",
    Header: "Duration",
    disableSortBy: true,
    accessor: (session) => getAISessionDuration(session),
    Cell: (cellProps: { cell: { value: string | null } }) => (
      <TextCell value={cellProps.cell.value} className="text-nowrap" />
    ),
  },
  {
    id: "user_account",
    Header: "User account",
    disableSortBy: true,
    accessor: (session) => getAISessionUserAccount(session),
    Cell: (cellProps: { cell: { value: string } }) =>
      cellProps.cell.value ? (
        <TooltipTruncatedTextCell value={cellProps.cell.value} />
      ) : (
        <TextCell value={null} />
      ),
  },
  {
    accessor: "tool_call_count",
    Header: "Events",
    disableSortBy: true,
    Cell: (cellProps) =>
      cellProps.row.original.backfill ? (
        <TextCell
          value={
            <TooltipWrapper tipContent={AI_SESSION_BACKFILL_MESSAGE}>
              {cellProps.cell.value}
            </TooltipWrapper>
          }
        />
      ) : (
        <TextCell value={cellProps.cell.value} />
      ),
  },
];

export default generateTableConfig;
