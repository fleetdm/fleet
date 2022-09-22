import React from "react";

import { IOsqueryPlatform } from "interfaces/platform";

import ColumnListItem from "./ColumnListItem";

type ColumnType = "integet" | "bigint" | "double" | "text" | "unsigned_bigint";

// TODO: move to common location
export interface IQueryTableColumn {
  name: string;
  description: string;
  type: ColumnType;
  required: boolean;
  platforms?: IOsqueryPlatform[];
  requires_user_context?: boolean;
}

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
  const columnListItems = orderColumns(columns).map((column) => {
    return <ColumnListItem key={column.name} column={column} />;
  });

  return (
    <div className={baseClass}>
      <h3>Columns</h3>
      <ul className={`${baseClass}__column-list`}>{columnListItems}</ul>
    </div>
  );
};

export default QueryTableColumns;
