/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import ReactTooltip from "react-tooltip";
import formatDistanceToNow from "date-fns/formatDistanceToNow";
import PATHS from "router/paths";

import permissionsUtils from "utilities/permissions";
import { IUser } from "interfaces/user";
import { secondsToDhms } from "utilities/helpers";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import { SupportedPlatform } from "interfaces/platform";

import Icon from "components/Icon";
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PlatformCell from "components/TableContainer/DataTable/PlatformCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import TooltipWrapper from "components/TooltipWrapper";
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
    original: ISchedulableQuery;
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
    value: SupportedPlatform[];
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
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = ({
  currentUser,
}: IGenerateTableHeaders): IDataColumn[] => {
  const isOnlyObserver = permissionsUtils.isOnlyObserver(currentUser);
  const isAnyTeamMaintainerOrTeamAdmin = permissionsUtils.isAnyTeamMaintainerOrTeamAdmin(
    currentUser
  );

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
            classes="w400"
            value={
              <>
                <div className="query-name-text">{cellProps.cell.value}</div>
                {!isOnlyObserver && cellProps.row.original.observer_can_run && (
                  <>
                    <span
                      className="tooltip-base"
                      data-tip
                      data-for={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                    >
                      <Icon className="query-icon" name="query" size="small" />
                    </span>
                    <ReactTooltip
                      className="observer-can-run-tooltip"
                      place="top"
                      type="dark"
                      effect="solid"
                      id={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                      backgroundColor="#3e4771"
                    >
                      Observers can run this query.
                    </ReactTooltip>
                  </>
                )}
              </>
            }
            path={PATHS.EDIT_QUERY(
              cellProps.row.original.id,
              cellProps.row.original.team_id ?? undefined
            )}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Platform",
      Header: "Platform",
      disableSortBy: true,
      accessor: "platforms",
      Cell: (cellProps: IPlatformCellProps): JSX.Element => {
        // translate the SelectedPlatformString into an array of `SupportedPlatform`s
        const selectedPlatforms =
          (cellProps.row.original.platform
            ?.split(",")
            .filter((platform) => platform !== "") as SupportedPlatform[]) ??
          [];

        const platformIconsToRender: SupportedPlatform[] =
          selectedPlatforms.length === 0
            ? // User didn't select any platforms, so we render all compatible
              cellProps.cell.value
            : // Render the platforms the user has selected for this query
              selectedPlatforms;

        return <PlatformCell platforms={platformIconsToRender} />;
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
              <>
                Assign a frequency and turn <strong>automations</strong> on to
                collect data at an interval.
              </>
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
            <TooltipWrapper
              tipContent={`
                This is the average <br />
                performance impact <br />
                across all hosts where this <br />
                query was scheduled.`}
            >
              Performance impact
            </TooltipWrapper>
          </div>
        );
      },
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps: IStringCellProps) => (
        <PillCell
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
  if (!isOnlyObserver) {
    tableHeaders.splice(0, 0, {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const {
          getToggleAllRowsSelectedProps,
          rows,
          selectedFlatRows,
          toggleAllRowsSelected,
          toggleRowSelected,
        } = cellProps;
        const { checked, indeterminate } = getToggleAllRowsSelectedProps();

        const disableToggleAllRowsSelected = () => {
          /* Team admin or team maintainer can only delete queries they authored
          If team admin or team maintainer authored 0 queries, disable select all queries for deletion */
          if (isAnyTeamMaintainerOrTeamAdmin) {
            return (
              rows.filter(
                (r: IQueryRow) => r.original.author_id === currentUser.id
              ).length === 0
            );
          }
          return false;
        };

        const checkboxProps = {
          value: checked,
          indeterminate,
          disabled: disableToggleAllRowsSelected(), // Disable select all if all rows are disabled
          onChange: () => {
            if (!isAnyTeamMaintainerOrTeamAdmin) {
              toggleAllRowsSelected();
            } else {
              // Team maintainers may only delete the queries that they have authored
              // so we need to do some filtering and then modify the toggle select all
              // behavior for the header checkbox
              const userAuthoredQueries = rows.filter(
                (r: IQueryRow) => r.original.author_id === currentUser.id
              );
              if (
                selectedFlatRows.length &&
                selectedFlatRows.length !== userAuthoredQueries.length
              ) {
                // If some but not all of the user authored queries are already selected,
                // we toggle all of the user's unselected queries to true
                userAuthoredQueries.forEach((r: IQueryRow) =>
                  toggleRowSelected(r.id, true)
                );
              } else {
                // Otherwise, we toggle all of the user's queries to the opposite of their current state
                userAuthoredQueries.forEach((r: IQueryRow) =>
                  toggleRowSelected(r.id)
                );
              }
            }
          },
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { row } = cellProps;
        const { checked } = row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: checked,
          onChange: () => row.toggleRowSelected(),
          disabled:
            isAnyTeamMaintainerOrTeamAdmin &&
            row.original.author_id !== currentUser.id,
        };
        // If the user is a team maintainer, we only enable checkboxes for queries
        // that they authored and we include a tooltip to explain disabled checkboxes
        return (
          <>
            <div
              data-tip
              data-for={`${"select-checkbox"}__${row.original.id}`}
              data-tip-disable={
                !isAnyTeamMaintainerOrTeamAdmin ||
                row.original.author_id === currentUser.id
              }
              className={`${
                !(
                  !isAnyTeamMaintainerOrTeamAdmin ||
                  row.original.author_id === currentUser.id
                ) && "tooltip"
              }`}
            >
              <Checkbox {...checkboxProps} />
            </div>{" "}
            <ReactTooltip
              className="select-checkbox-tooltip"
              place="bottom"
              effect="solid"
              backgroundColor="#3e4771"
              id={`${"select-checkbox"}__${row.original.id}`}
              data-html
            >
              <>
                You can only delete a<br /> query if you are the author.
              </>
            </ReactTooltip>
          </>
        );
      },
      disableHidden: true,
    });
  }
  return tableHeaders;
};

export default generateTableHeaders;
