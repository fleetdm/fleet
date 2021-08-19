import React, { useState } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";

import hostsAPI from "services/entities/hosts";
import { IHost } from "interfaces/host";
import { ITarget } from "interfaces/target";

// @ts-ignore
import Input from "components/forms/fields/InputFieldWithIcon";
import TableContainer from "components/TableContainer";
import { generateTableHeaders } from "./TargetsInputHostsTableConfig";
import { xorBy } from "lodash";

interface IHostsQueryResponse {
  hosts: IHost[];
}

interface ITargetsInputProps {
  tabIndex: number;
  searchText: string;
  relatedHosts: IHost[];
  selectedTargets: ITarget[];
  setSearchText: (value: string) => void;
  handleRowSelect: (value: Row) => void;
}

const baseClass = "targets-input";

const EmptyHosts = () => (
  <p>No hosts match the current search criteria.</p>
);

const EmptyChosenHosts = () => (
  <p>No hosts are chosen. Type something above.</p>
);

const TargetsInput = ({
  tabIndex,
  searchText,
  relatedHosts,
  selectedTargets,
  handleRowSelect,
  setSearchText,
}: ITargetsInputProps) => {
  // const { 
  //   status: hostsLoadedStatus, 
  //   data: { hosts: loadedHosts } = {}, 
  //   error: hostsLoadedError 
  // } = useQuery<IHostsQueryResponse, Error>(
  //   ["hostsFromInput", searchText], 
  //   () => hostsAPI.search(searchText), {
  //     enabled: !!searchText,
  //     refetchOnWindowFocus: false,
  //   }
  // );

  // get the difference of all hosts returned vs hosts selected (inside selectedTargets)
  // so we can remove selected hosts from the dropdown table
  const finalHosts = relatedHosts && xorBy(relatedHosts, selectedTargets, "uuid");
  const tableHeaders = generateTableHeaders();
  return (
    <div className={baseClass}>
      <Input 
        autofocus={true}
        type="text"
        iconName="search"
        value={searchText}
        tabIndex={tabIndex}
        iconPosition="start"
        label="Target specific hosts"
        placeholder="Search hosts by hostname, UUID, MAC address"
        onChange={setSearchText}
      />
      {finalHosts && (
        <div className={`${baseClass}__hosts-search-dropdown`}>
          <TableContainer
            columns={tableHeaders}
            data={finalHosts}
            isLoading={false}
            resultsTitle=""
            emptyComponent={EmptyHosts}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount={true}
            disablePagination={true}
            disableMultiRowSelect={true}
            onSelectSingleRow={handleRowSelect}
          />
        </div>
      )}
      <div className={`${baseClass}__hosts-selected-table`}>
        <TableContainer
            columns={tableHeaders}
            data={selectedTargets}
            isLoading={false}
            resultsTitle=""
            emptyComponent={EmptyChosenHosts}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount={true}
            disablePagination={true}
            disableMultiRowSelect={true}
          />
      </div>
    </div>
  )
};

export default TargetsInput;