/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import ReactTooltip from "react-tooltip";
import formatDistanceToNow from "date-fns/formatDistanceToNow";

import permissionsUtils from "utilities/permissions";

// @ts-ignore
import Avatar from "components/Avatar";
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PlatformCell from "components/TableContainer/DataTable/PlatformCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import PillCell from "components/TableContainer/DataTable/PillCell";

import PATHS from "router/paths";

import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
import { addGravatarUrlToResource } from "fleet/helpers";

interface IQueryRow {
  id: string;
  original: IQuery;
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
  toggleRowSelected: (id: string, value?: boolean) => void;
  rows: IQueryRow[];
  selectedFlatRows: IQueryRow[];
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IQuery;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
  };
  toggleRowSelected: (id: string, value: boolean) => void;
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
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
      Cell: (cellProps: ICellProps): JSX.Element => {
        return <PlatformCell value={cellProps.cell.value} />;
      },
    },
    {
      title: "Performance impact",
      Header: "Performance impact",
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps) => (
        <PillCell value={[cellProps.cell.value, cellProps.row.original.id]} />
      ),
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
            {author}
          </span>
        );
      },
      sortType: "caseInsensitive",
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
            >
              <Checkbox {...checkboxProps} />
            </div>{" "}
            <ReactTooltip
              className="select-checkbox-tooltip"
              place="bottom"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id={`${"select-checkbox"}__${row.original.id}`}
              data-html
            >
              <div style={{ width: "196px", textAlign: "center" }}>
                You can only delete a<br /> query if you are the author.
              </div>
            </ReactTooltip>
          </>
        );
      },
      disableHidden: true,
    });
    tableHeaders.splice(2, 0, {
      title: "Observer can run",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "observer_can_run",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell
          value={cellProps.row.original.observer_can_run ? "Yes" : "No"}
        />
      ),
      sortType: "basic",
    });
  }
  return tableHeaders;
};

export default generateTableHeaders;
