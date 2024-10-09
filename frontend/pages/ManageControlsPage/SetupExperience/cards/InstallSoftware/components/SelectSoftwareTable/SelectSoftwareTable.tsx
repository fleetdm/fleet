import React from "react";

import { ISoftwareTitle } from "interfaces/software";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./SelectSoftwareTableConfig";

const baseClass = "select-software-table";

interface ISelectSoftwareTableProps {
  softwareTitles: ISoftwareTitle[];
  onChangeSoftwareSelect: (select: boolean, id: number) => void;
  onChangeSelectAll: (selectAll: boolean) => void;
}

const SelectSoftwareTable = ({
  softwareTitles,
  onChangeSoftwareSelect,
  onChangeSelectAll,
}: ISelectSoftwareTableProps) => {
  const tabelConfig = generateTableConfig(
    onChangeSelectAll,
    onChangeSoftwareSelect
  );

  return (
    <TableContainer
      className={baseClass}
      data={softwareTitles}
      columnConfigs={tabelConfig}
      isLoading={false}
      emptyComponent={() => null}
      showMarkAllPages
      isAllPagesSelected={false}
      disablePagination
      searchable
      searchQueryColumn="name"
      isClientSideFilter
      onClearSelection={() => onChangeSelectAll(false)}
    />
  );
};

export default SelectSoftwareTable;
