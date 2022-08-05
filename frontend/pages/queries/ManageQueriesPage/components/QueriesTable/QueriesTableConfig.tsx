/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import ReactTooltip from "react-tooltip";
import formatDistanceToNow from "date-fns/formatDistanceToNow";
import PATHS from "router/paths";

import permissionsUtils from "utilities/permissions";
import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
import { addGravatarUrlToResource } from "utilities/helpers";

import Avatar from "components/Avatar";
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PlatformCell from "components/TableContainer/DataTable/PlatformCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import TooltipWrapper from "components/TooltipWrapper";

interface IQueryRow {
  id: string;
  original: IQuery;
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
    original: IQuery;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
  toggleRowSelected: (id: string, value: boolean) => void;
}

interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IPlatformCellProps extends IRowProps {
  cell: {
    value: string[];
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPlatformCellProps) => JSX.Element);
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (currentUser: IUser): IDataColumn[] => {
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
      Cell: (cellProps: ICellProps): JSX.Element => (
        <LinkCell
          classes="w400"
          value={cellProps.cell.value}
          path={PATHS.EDIT_QUERY(cellProps.row.original)}
        />
      ),
      sortType: "caseInsensitive",
    },
    {
      title: "Platform",
      Header: "Platform",
      disableSortBy: true,
      accessor: "platforms",
      Cell: (cellProps: IPlatformCellProps): JSX.Element => {
        return <PlatformCell value={cellProps.cell.value} />;
      },
    },
    {
      title: "Author",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "author_name",
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { author_name, author_email } = cellProps.row.original;
        const author = author_name === currentUser.name ? "You" : author_name;
        return (
          <span>
            <Avatar
              user={addGravatarUrlToResource({ email: author_email })}
              size="xsmall"
            />
            <span className="text-cell author-name">{author}</span>
          </span>
        );
      },
      sortType: "caseInsensitive",
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
      Cell: (cellProps: ICellProps) => (
        <PillCell value={[cellProps.cell.value, cellProps.row.original.id]} />
      ),
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
      Cell: (cellProps: ICellProps): JSX.Element => (
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
        const checkboxProps = {
          value: checked,
          indeterminate,
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
