import React from "react";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import { IMacSettings } from "interfaces/mdm";

import tableHeaders from "./MacSettingsTableConfig";

const baseClass = "macsettings-table";

interface IMacSettingsTableProps {
  hostMacSettings?: IMacSettings;
}

const MacSettingsTable = ({ hostMacSettings }: IMacSettingsTableProps) => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="settings"
        defaultSortHeader="name"
        columns={tableHeaders}
        data={hostMacSettings}
        emptyComponent={"symbol"}
        isLoading={false}
        // TODO:
        // emptyComponent={() =>
        //   EmptyTable({
        //     iconName: emptyState().iconName,
        //     header: emptyState().header,
        //     info: emptyState().info,
        //     additionalInfo: emptyState().additionalInfo,
        //     primaryButton: emptyState().primaryButton,
        //   })
        // }
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
      />
    </div>
  );
};

export default MacSettingsTable;
