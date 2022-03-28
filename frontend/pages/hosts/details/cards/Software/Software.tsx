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
  const [filterName, setFilterName] = useState("");
  const [filterVuln, setFilterVuln] = useState(false);
  const [filters, setFilters] = useState({
    name: filterName,
    vulnerabilities: filterVuln,
  });

  useEffect(() => {
    setFilters({ name: filterName, vulnerabilities: filterVuln });
  }, [filterName, filterVuln]);

  const onQueryChange = useDebouncedCallback(
    ({ searchQuery }: { searchQuery: string }) => {
      setFilterName(searchQuery);
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
              data={software}
              filters={filters}
              isLoading={isLoading}
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              inputPlaceHolder={"Filter software"}
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
