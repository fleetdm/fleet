import React, { useMemo } from "react";

import { ISoftwareTitle } from "interfaces/software";
import { SetupExperiencePlatform } from "interfaces/platform";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import generateTableConfig from "./InstallSoftwareTableConfig";

const DEFAULT_PAGE_SIZE = 10;

const baseClass = "select-software-table";

const generateSelectedRows = (softwareTitles: ISoftwareTitle[]) => {
  return softwareTitles.reduce<Record<string, boolean>>((acc, software) => {
    if (
      software.software_package?.install_during_setup ||
      software.app_store_app?.install_during_setup
    ) {
      if (software.id != null) {
        acc[String(software.id)] = true; // key must match DataTable getRowId(row) for selection to persist
      }
    }
    return acc;
  }, {});
};

interface IInstallSoftwareTableProps {
  softwareTitles: ISoftwareTitle[];
  onChangeSoftwareSelect: (select: boolean, id: number) => void;
  platform: SetupExperiencePlatform;
  renderCustomCount?: () => JSX.Element;
  manualAgentInstallBlockingSoftware?: boolean;
}

const InstallSoftwareTable = ({
  softwareTitles,
  onChangeSoftwareSelect,
  platform,
  renderCustomCount,
  manualAgentInstallBlockingSoftware = false,
}: IInstallSoftwareTableProps) => {
  const tableConfig = useMemo(() => {
    return generateTableConfig(
      platform,
      onChangeSoftwareSelect,
      manualAgentInstallBlockingSoftware
    );
  }, [onChangeSoftwareSelect, platform, manualAgentInstallBlockingSoftware]);

  const initialSelectedSoftwareRows = useMemo(() => {
    return generateSelectedRows(softwareTitles);
  }, [softwareTitles]);

  return (
    <TableContainer
      className={baseClass}
      data={softwareTitles}
      columnConfigs={tableConfig}
      isLoading={false}
      emptyComponent={() => (
        <EmptyTable
          header="No software available"
          info=" There are no results to your query."
          className={baseClass}
        />
      )}
      renderCount={renderCustomCount}
      defaultSelectedRows={initialSelectedSoftwareRows}
      showMarkAllPages
      isAllPagesSelected={false}
      persistSelectedRows // Keeps selected rows across pagination (client-side)
      isClientSidePagination
      pageSize={DEFAULT_PAGE_SIZE}
      searchable
      searchQueryColumn="name"
      isClientSideFilter
      renderTableHelpText={() => (
        <p className={`${baseClass}__help-text`}>
          Software will be installed on{" "}
          {platform === "linux" ? "compatible platforms" : "all hosts"}.
          Currently, custom targets (labels) don&apos;t apply during setup
          experience.
        </p>
      )}
      suppressHeaderActions
    />
  );
};

export default InstallSoftwareTable;
