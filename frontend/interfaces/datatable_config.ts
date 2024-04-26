/** This file contains reusable types that can be used when creating new tables
 *  using react-table.
 */

import { CellProps, Column, HeaderProps } from "react-table";

export type IDataColumn = Column & {
  title?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  filterValue?: any;
  preFilteredRows?: any;
  setFilter?: any;
};

/**
 * Interface for a column configuration in a table. This is a wrapper around the `Column` type from
 * `react-table`.
 * @param T - The type of the data in the column
 * @example
 * ```ts
 * type IHostTableColumnConfig = IColumnConfig<IHost>;
 */
export type IColumnConfig<T extends object> = Column<T>;

/**
 * Interface for a cell with a string value. This is a wrapper around the
 * `CellProps` type from `react-table`.
 */
export type IStringCellProps<T extends object> = CellProps<T, string>;

/**
 * Interface for a cell with a number value. This is a wrapper around the
 * `CellProps` type from `react-table`.
 */
export type INumberCellProps<T extends object> = CellProps<T, number>;

/**
 *
 */
export type IBoolCellProps<T extends object> = CellProps<T, boolean>;

/**
 * Interface for a cell with a value that is an array of objects. This is a
 * wrapper around the `HeaderProps` type from `react-table`.
 */
export type IHeaderProps<T extends object> = HeaderProps<T>;

/**
 * The typing for web socket data is loose as we are getting the data is
 * not typed and is not guaranteed to have the same shape every time.
 * */
export type IWebSocketData = Record<string, unknown>;
