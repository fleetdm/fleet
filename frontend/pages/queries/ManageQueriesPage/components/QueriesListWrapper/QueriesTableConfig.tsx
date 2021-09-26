/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import ReactTooltip from "react-tooltip";

import moment from "moment";
import { capitalize } from "lodash";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";

import PATHS from "router/paths";

import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";

import permissionsUtils from "utilities/permissions";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
  toggleRowSelected: (id: string, value?: boolean) => void;
  rows: any;
  selectedFlatRows: any;
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

const countUserAuthoredQueries = (
  currentUser: IUser,
  queries: IQuery[]
): number => {
  const userAuthoredQueries = queries.filter(
    (q: IQuery) => q.author_id === currentUser.id
  );
  return userAuthoredQueries.length || 0;
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (currentUser: IUser): IDataColumn[] => {
  const isOnlyObserver = permissionsUtils.isOnlyObserver(currentUser);
  const isAnyTeamMaintainer = permissionsUtils.isAnyTeamMaintainer(currentUser);

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
      title: "Author",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "author_name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
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
        <TextCell value={moment(cellProps.cell.value).format("MM/DD/YY")} />
      ),
    },
  ];
  if (!isOnlyObserver) {
    tableHeaders.splice(0, 0, {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        console.log("headercell: ", cellProps);
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
          // onChange: () => cellProps.toggleAllRowsSelected(),
          onChange: () => {
            console.log("clicked header select checkbox");
            if (!isAnyTeamMaintainer) {
              console.log("not team maintainer so toggled all rows seleted");
              toggleAllRowsSelected();
            } else {
              console.log("team maintainer do some more logic");
              const userAuthoredQueries = rows.filter(
                (r: any) => r.original.author_id === currentUser.id
              );
              if (
                selectedFlatRows.length &&
                selectedFlatRows.length !== userAuthoredQueries.length
              ) {
                console.log("some but not all selected");
                userAuthoredQueries.forEach((r: any) =>
                  toggleRowSelected(r.id, true)
                );
              } else {
                console.log("all or none selected");
                userAuthoredQueries.forEach((r: any) =>
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
            isAnyTeamMaintainer && row.original.author_id !== currentUser.id,
        };
        return (
          <>
            <div
              data-tip
              data-for={`${"select-checkbox"}__${row.original.id}`}
              data-tip-disable={
                !isAnyTeamMaintainer ||
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
    tableHeaders.splice(3, 0, {
      title: "Observer can run",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "observer_can_run",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={capitalize(cellProps.cell.value.toString())} />
      ),
      sortType: "basic",
    });
  }
  return tableHeaders;
};

export { countUserAuthoredQueries, generateTableHeaders };
