import React from "react";

import { IPkiConfig } from "interfaces/pki";

import TableContainer from "components/TableContainer";

import { generateTableConfig } from "./PkiTableConfig";

const baseClass = "pki-table";

interface IPkiTableProps {
  data: IPkiConfig[];
  onEdit: (pkiConfig: IPkiConfig) => void;
  onDelete: (pkiConfig: IPkiConfig) => void;
}

const PkiTable = ({ data, onEdit, onDelete }: IPkiTableProps) => {
  const onSelectAction = (action: string, pkiConfig: IPkiConfig) => {
    switch (action) {
      case "view_template":
        onEdit(pkiConfig);
        break;
      case "add_template":
        onEdit(pkiConfig);
        break;
      case "delete":
        onDelete(pkiConfig);
        break;
      default:
        break;
    }
  };

  const tableConfig = generateTableConfig(onSelectAction);

  return (
    <TableContainer<IPkiConfig>
      columnConfigs={tableConfig}
      defaultSortHeader="pki_name"
      disableTableHeader
      disablePagination
      showMarkAllPages={false}
      isAllPagesSelected={false}
      emptyComponent={() => <></>}
      isLoading={false}
      data={data}
      className={baseClass}
    />
  );
};

export default PkiTable;
