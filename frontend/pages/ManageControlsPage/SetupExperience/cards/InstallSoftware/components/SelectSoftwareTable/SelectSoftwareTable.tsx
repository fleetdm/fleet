import React, { useMemo } from "react";

import { ISoftwareTitle } from "interfaces/software";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./SelectSoftwareTableConfig";
import EmptyTable from "components/EmptyTable";

const baseClass = "select-software-table";

interface ISelectSoftwareTableProps {
  initialSelectedSoftware: number[];
  softwareTitles: ISoftwareTitle[];
  onChangeSoftwareSelect: (select: boolean, id: number) => void;
  onChangeSelectAll: (selectAll: boolean) => void;
}

const SelectSoftwareTable = ({
  softwareTitles,
  initialSelectedSoftware,
  onChangeSoftwareSelect,
  onChangeSelectAll,
}: ISelectSoftwareTableProps) => {
  const tabelConfig = useMemo(() => {
    console.log("initial selected software", initialSelectedSoftware);
    return generateTableConfig(
      initialSelectedSoftware,
      onChangeSelectAll,
      onChangeSoftwareSelect
    );
  }, [initialSelectedSoftware, onChangeSelectAll, onChangeSoftwareSelect]);

  return (
    <TableContainer
      className={baseClass}
      data={softwareTitles}
      columnConfigs={tabelConfig}
      isLoading={false}
      emptyComponent={() => (
        <EmptyTable
          header="No software available"
          info=" There are no results to your query."
          className={baseClass}
        />
      )}
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
