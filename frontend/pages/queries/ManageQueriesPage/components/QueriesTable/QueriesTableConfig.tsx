/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { formatDistanceToNow } from "date-fns";
import PATHS from "router/paths";

import permissionsUtils from "utilities/permissions";
import { IUser } from "interfaces/user";
import { secondsToDhms } from "utilities/helpers";
import {
  IEnhancedQuery,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import {
  isScheduledQueryablePlatform,
  ScheduledQueryablePlatform,
  SelectedPlatformString,
} from "interfaces/platform";
import { API_ALL_TEAMS_ID } from "interfaces/team";

import Icon from "components/Icon";
import Checkbox from "components/forms/fields/Checkbox";
import { getConditionalSelectHeaderCheckboxProps } from "components/TableContainer/utilities/config_utils";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PlatformCell from "components/TableContainer/DataTable/PlatformCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";
import InheritedBadge from "components/InheritedBadge";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";
import QueryAutomationsStatusIndicator from "../QueryAutomationsStatusIndicator";

interface IQueryRow {
  id: string;
  original: ISchedulableQuery;
}

interface IGetToggleAllRowsSelectedProps {
  checked: boolean;
  indeterminate: boolean;
  title: string;
  onChange: () => void;
  style: { cursor: string };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => IGetToggleAllRowsSelectedProps;
  toggleAllRowsSelected: () => void;
  toggleRowSelected: (id: string, value?: boolean) => void;
  rows: IQueryRow[];
  selectedFlatRows: IQueryRow[];
}
interface IRowProps {
  row: {
    original: IEnhancedQuery;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
  toggleRowSelected: (id: string, value: boolean) => void;
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface INumberCellProps extends IRowProps {
  cell: {
    value: number;
  };
}

interface IStringCellProps extends IRowProps {
  cell: { value: string };
}

interface IBoolCellProps extends IRowProps {
  cell: { value: boolean };
}
interface IPlatformCellProps extends IRowProps {
  cell: {
    value: SelectedPlatformString;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPlatformCellProps) => JSX.Element)
    | ((props: IStringCellProps) => JSX.Element)
    | ((props: INumberCellProps) => JSX.Element)
    | ((props: IBoolCellProps) => JSX.Element);
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

interface IGenerateTableHeaders {
  currentUser: IUser;
  currentTeamId?: number;
  omitSelectionColumn?: boolean;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = ({
  currentUser,
  currentTeamId,
  omitSelectionColumn = false,
}: IGenerateTableHeaders): IDataColumn[] => {
  const isCurrentTeamObserverOrGlobalObserver = currentTeamId
    ? permissionsUtils.isTeamObserver(currentUser, currentTeamId)
    : permissionsUtils.isOnlyObserver(currentUser);
  const viewingTeamScope = currentTeamId !== API_ALL_TEAMS_ID;

  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => {
        return (
          <LinkCell
            className="w400 query-name-cell"
            value={
              <>
                <div className="query-name-text">{cellProps.cell.value}</div>
                {!isCurrentTeamObserverOrGlobalObserver &&
                  cellProps.row.original.observer_can_run && (
                    <div className="observer-can-run-badge">
                      <span
                        className="observer-can-run-icon"
                        data-tooltip-id={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                      >
                        <Icon
                          className="observer-can-run-query-icon"
                          name="query"
                          size="small"
                          color="core-fleet-blue"
                        />
                      </span>
                      <ReactTooltip5
                        className="observer-can-run-tooltip"
                        disableStyleInjection
                        place="top"
                        opacity={1}
                        id={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                        offset={8}
                        positionStrategy="fixed"
                      >
                        Observers can run this query.
                      </ReactTooltip5>
                    </div>
                  )}
                {viewingTeamScope &&
                  // inherited
                  cellProps.row.original.team_id !== currentTeamId && (
                    <InheritedBadge tooltipContent="This query runs on all hosts." />
                  )}
              </>
            }
            path={PATHS.QUERY_DETAILS(
              cellProps.row.original.id,
              cellProps.row.original.team_id ?? currentTeamId
            )}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Platform",
      Header: "Targeted platforms",
      disableSortBy: true,
      accessor: "platform",
      Cell: (cellProps: IPlatformCellProps): JSX.Element => {
        if (!cellProps.row.original.interval) {
          // if the query isn't scheduled to run, return default empty call
          return <TextCell />;
        }
        const platforms = cellProps.cell.value
          .split(",")
          .map((s) => s.trim())
          // this casting is necessary because make generate for some reason doesn't recognize the
          // type guarding of `isQueryablePlatform` even though the language server in VSCode does
          .filter((s) =>
            isScheduledQueryablePlatform(s)
          ) as ScheduledQueryablePlatform[];
        return <PlatformCell platforms={platforms} />;
      },
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: true,
      accessor: "interval",
      Cell: (cellProps: INumberCellProps): JSX.Element => {
        const val = cellProps.cell.value
          ? `Every ${secondsToDhms(cellProps.cell.value)}`
          : undefined;
        return (
          <TextCell
            value={val}
            emptyCellTooltipText={
              <>Assign a frequency to collect data at an interval.</>
            }
          />
        );
      },
    },
    {
      title: "Performance impact",
      Header: () => {
        return (
          <div>
            <TooltipWrapper tipContent="The average performance impact across all hosts.">
              Performance impact
            </TooltipWrapper>
          </div>
        );
      },
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps: IStringCellProps) => (
        <PerformanceImpactCell
          value={{
            indicator: cellProps.cell.value,
            id: cellProps.row.original.id,
          }}
        />
      ),
    },
    {
      title: "Automations",
      Header: "Automations",
      disableSortBy: true,
      accessor: "automations_enabled",
      Cell: (cellProps: IBoolCellProps): JSX.Element => {
        return (
          <QueryAutomationsStatusIndicator
            automationsEnabled={cellProps.cell.value}
            interval={cellProps.row.original.interval}
          />
        );
      },
    },
    {
      title: "Last modified",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "updated_at",
      Cell: (cellProps: INumberCellProps): JSX.Element => (
        <TextCell
          value={formatDistanceToNow(new Date(cellProps.cell.value), {
            includeSeconds: true,
            addSuffix: true,
          })}
        />
      ),
    },
  ];
  if (!isCurrentTeamObserverOrGlobalObserver && !omitSelectionColumn) {
    tableHeaders.unshift({
      id: "selection",
      // TODO - improve typing of IHeaderProps instead of using any
      // Header: (headerProps: IHeaderProps): JSX.Element => {
      Header: (headerProps: any): JSX.Element => {
        const checkboxProps = getConditionalSelectHeaderCheckboxProps({
          headerProps,
          checkIfRowIsSelectable: (row) =>
            (row.original.team_id ?? undefined) === currentTeamId,
        });

        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const isInheritedQuery =
          (cellProps.row.original.team_id ?? undefined) !== currentTeamId;
        if (viewingTeamScope && isInheritedQuery) {
          // disallow selecting inherited queries
          return <></>;
        }
        const { row } = cellProps;
        const { checked } = row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: checked,
          onChange: () => row.toggleRowSelected(),
        };
        // v4.35.0 Any team admin or maintainer now can add, edit, delete their team's queries
        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      disableHidden: true,
    });
  }
  return tableHeaders;
};

export default generateTableHeaders;
