import React from "react";

import { ISoftwareTitle } from "interfaces/software";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./SelectSoftwareTableConfig";

const baseClass = "select-software-table";

interface ISelectSoftwareTableProps {
  software: ISoftwareTitle[];
}

const SelectSoftwareTable = ({ software }: ISelectSoftwareTableProps) => {
  const tabelConfig = generateTableConfig();

  return (
    <TableContainer
      className={baseClass}
      data={software}
      columnConfigs={tabelConfig}
      isLoading={false}
      emptyComponent={() => null}
      showMarkAllPages
      isAllPagesSelected={false}
    />
  );
};

export default SelectSoftwareTable;
