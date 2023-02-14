import React from "react";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import { IMacSettings } from "interfaces/mdm";

import { tableHeaders, generateDataSet } from "./MacSettingsTableConfig";

const baseClass = "macsettings-table";

interface IMacSettingsTableProps {
  isLoading: boolean;
  hostMacSettings: IMacSettings;
}

const MacSettingsTable = ({
  isLoading,
  hostMacSettings,
}: IMacSettingsTableProps) => {
  return (
    <div className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
        <TableContainer
          resultsTitle="settings"
          defaultSortHeader="name"
          columns={tableHeaders}
          data={generateDataSet(hostMacSettings)} // TODO
          isLoading={isLoading}
          emptyComponent={"symbol"}
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
        />
      )}
    </div>
  );
};

export default MacSettingsTable;
