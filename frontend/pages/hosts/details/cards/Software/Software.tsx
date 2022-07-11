import React, { useEffect, useMemo, useState } from "react";
import { useDebouncedCallback } from "use-debounce";

import { ISoftware } from "interfaces/software";
import { VULNERABLE_DROPDOWN_OPTIONS } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";

import EmptyState from "../EmptyState";
import SoftwareVulnCount from "./SoftwareVulnCount";

import {
  generateSoftwareTableHeaders,
  generateSoftwareTableData,
} from "./SoftwareTableConfig";

const baseClass = "host-details";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface ISoftwareTableProps {
  isLoading: boolean;
  software: ISoftware[];
  deviceUser?: boolean;
  deviceType?: string;
  softwareInventoryEnabled?: boolean;
}

const SoftwareTable = ({
  isLoading,
  software,
  deviceUser,
  deviceType,
  softwareInventoryEnabled,
}: ISoftwareTableProps): JSX.Element => {
  const [searchString, setSearchString] = useState("");
  const [filterVuln, setFilterVuln] = useState(false);
  const [filters, setFilters] = useState({
    global: searchString,
    vulnerabilities: filterVuln,
  });

  useEffect(() => {
    setFilters({ global: searchString, vulnerabilities: filterVuln });
  }, [searchString, filterVuln]);

  const onQueryChange = useDebouncedCallback(
    ({ searchQuery }: { searchQuery: string }) => {
      setSearchString(searchQuery);
    },
    300
  );

  const tableSoftware = useMemo(() => generateSoftwareTableData(software), [
    software,
  ]);
  const tableHeaders = useMemo(() => generateSoftwareTableHeaders(deviceUser), [
    deviceUser,
  ]);

  const onVulnFilterChange = (value: boolean) => {
    setFilterVuln(value);
  };

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filters.vulnerabilities}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={onVulnFilterChange}
      />
    );
  };

  const EmptySoftwareSearch = () => (
    <EmptyState title="software" reason="empty-search" />
  );

  if (softwareInventoryEnabled === false) {
    return (
      <div className="section section--software">
        <p className="section__header">Software</p>
        <EmptyState title="software" reason="disabled" />
      </div>
    );
  }

  return (
    <div className="section section--software">
      <p className="section__header">Software</p>

      {software?.length ? (
        <>
          {software && (
            <SoftwareVulnCount
              softwareList={software}
              deviceUser={deviceUser}
            />
          )}
          {software && (
            <div className={deviceType || ""}>
              <TableContainer
                columns={tableHeaders}
                data={tableSoftware || []}
                filters={filters}
                isLoading={isLoading}
                defaultSortHeader={"name"}
                defaultSortDirection={"asc"}
                inputPlaceHolder={
                  "Search software by name or vulnerabilities ( CVEs)"
                }
                onQueryChange={onQueryChange}
                resultsTitle={"software items"}
                emptyComponent={EmptySoftwareSearch}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                searchable
                customControl={renderVulnFilterDropdown}
                isClientSidePagination
                pageSize={20}
                isClientSideFilter
                highlightOnHover
              />
            </div>
          )}
        </>
      ) : (
        <EmptyState title="software" />
      )}
    </div>
  );
};
export default SoftwareTable;
