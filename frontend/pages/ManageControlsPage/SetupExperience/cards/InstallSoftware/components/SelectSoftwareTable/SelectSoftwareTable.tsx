import React, { useCallback, useMemo } from "react";

import { ISoftwareTitle } from "interfaces/software";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import TableCount from "components/TableContainer/TableCount";

import generateTableConfig from "./SelectSoftwareTableConfig";

const baseClass = "select-software-table";

const generateSelectedRows = (softwareTitles: ISoftwareTitle[]) => {
  return softwareTitles.reduce<Record<string, boolean>>((acc, software, i) => {
    if (
      software.software_package?.install_during_setup ||
      software.app_store_app?.install_during_setup
    ) {
      acc[i] = true;
    }
    return acc;
  }, {});
};

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
  const tabelConfig = useMemo(() => {
    return generateTableConfig(onChangeSelectAll, onChangeSoftwareSelect);
  }, [onChangeSelectAll, onChangeSoftwareSelect]);

  const initialSelectedSoftwareRows = useMemo(() => {
    return generateSelectedRows(softwareTitles);
  }, [softwareTitles]);

  const renderCount = useCallback(() => {
    if (softwareTitles.length === 0) {
      return <></>;
    }

    return <TableCount name="items" count={softwareTitles?.length} />;
  }, [softwareTitles]);

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
      renderCount={renderCount}
      defaultSelectedRows={initialSelectedSoftwareRows}
      showMarkAllPages
      isAllPagesSelected={false}
      persistSelectedRows
      disablePagination
      searchable
      searchQueryColumn="name"
      isClientSideFilter
      onClearSelection={() => onChangeSelectAll(false)}
    />
  );
};

export default SelectSoftwareTable;
