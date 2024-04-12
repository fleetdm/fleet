import React, { useContext } from "react";

import { IQueryTableColumn } from "interfaces/osquery_table";
import { QueryContext } from "context/query";

import ColumnListItem from "./ColumnListItem";

const sortAlphabetically = (
  columnA: IQueryTableColumn,
  columnB: IQueryTableColumn
) => {
  return columnA.name.localeCompare(columnB.name);
};

/**
 * Orders the columns by required columns first sorted alphabetically,
 * then the rest of the columns sorted alphabetically.
 */
const orderColumns = (columns: IQueryTableColumn[]) => {
  const requiredColumns = columns.filter((column) => column.required);
  const nonRequiredColumns = columns.filter((column) => !column.required);

  const sortedRequiredColumns = requiredColumns.sort(sortAlphabetically);
  const sortedNonRequiredColumns = nonRequiredColumns.sort(sortAlphabetically);

  return [...sortedRequiredColumns, ...sortedNonRequiredColumns];
};

interface IQueryTableColumnsProps {
  columns: IQueryTableColumn[];
}

const baseClass = "query-table-columns";

const QueryTableColumns = ({ columns }: IQueryTableColumnsProps) => {
  const { selectedOsqueryTable } = useContext(QueryContext);

  const columnListItems = orderColumns(columns).map((column) => {
    return (
      <ColumnListItem
        key={column.name}
        column={column}
        selectedTableName={selectedOsqueryTable.name}
      />
    );
  });

  return (
    <div className={baseClass}>
      <h3>Columns</h3>
      <ul className={`${baseClass}__column-list`}>{columnListItems}</ul>
    </div>
  );
};

export default QueryTableColumns;
