import React from "react";

import moment from "moment";

import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import { IQuery } from "interfaces/query";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
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
}

interface IDataColumn {
  id?: string;
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor?: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IQueryTableData {
  id: number;
  name: string;
  description: string;
  observer_can_run: string | boolean;
  author_name: string;
  last_modified: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (isOnlyObserver = false): IDataColumn[] => {
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
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Description",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "description",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
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
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Last modified",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "last_modified",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
  ];

  // Add select checkbox and observers can run columns for non-observer (i.e. maintainer and admin) users only
  if (!isOnlyObserver) {
    tableHeaders.splice(0, 0, {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      disableHidden: true,
    });

    tableHeaders.splice(3, 0, {
      title: "Observers can run",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "observer_can_run",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    });
  }

  return tableHeaders;
};

const generateTableData = (queries: IQuery[]): IQueryTableData[] => {
  return queries.map((query) => {
    return {
      id: query.id,
      name: query.name || "---",
      description: query.description || "--",
      observer_can_run: query.observer_can_run,
      author_name: query.author_name,
      last_modified: moment(query.updated_at).format("MM/DD/YY"),
    };
  });
};

export { generateTableHeaders, generateTableData };
