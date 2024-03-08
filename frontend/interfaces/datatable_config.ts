import { Column } from "react-table";

export type IDataColumn = Column & {
  title?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  filterValue?: any;
  preFilteredRows?: any;
  setFilter?: any;
};
