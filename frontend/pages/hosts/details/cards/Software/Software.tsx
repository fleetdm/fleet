import React, { useEffect, useState } from "react";
import { useDebouncedCallback } from "use-debounce/lib";

import { ISoftware } from "interfaces/software";
import { VULNERABLE_DROPDOWN_OPTIONS } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";

import EmptySoftware from "./EmptySoftware";
import SoftwareVulnCount from "./SoftwareVulnCount";

import generateSoftwareTableHeaders from "./SoftwareTableConfig";

const baseClass = "host-details";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface ISoftwareTableProps {
  isLoading: boolean;
  software: ISoftware[];
  deviceUser?: boolean;
}

const SoftwareTable = ({
  isLoading,
  software,
  deviceUser,
}: ISoftwareTableProps): JSX.Element => {
  const tableSoftware: ITableSoftware[] = software.map((s) => {
    return {
      ...s,
      vulnerabilities:
        s.vulnerabilities?.map((v) => {
          return v.cve;
        }) || [],
    };
  });

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

  const tableHeaders = generateSoftwareTableHeaders(deviceUser);

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
            <TableContainer
              columns={tableHeaders}
              data={tableSoftware}
              filters={filters}
              isLoading={isLoading}
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              inputPlaceHolder={
                "Search software by name or vulnerabilities (CVEs)"
              }
              onQueryChange={onQueryChange}
              resultsTitle={"software items"}
              emptyComponent={EmptySoftware}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              customControl={renderVulnFilterDropdown}
              isClientSidePagination
              isClientSideFilter
              highlightOnHover
            />
          )}
        </>
      ) : (
        <div className="results">
          <p className="results__header">
            No installed software detected on this host.
          </p>
          <p className="results__data">
            Expecting to see software? Try again in a few seconds as the system
            catches up.
          </p>
        </div>
      )}
    </div>
  );
};
export default SoftwareTable;
