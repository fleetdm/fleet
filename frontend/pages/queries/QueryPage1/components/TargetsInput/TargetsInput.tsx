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
import { filter, pullAllBy } from "lodash";

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

const TargetsInput = ({
  tabIndex,
  searchText,
  relatedHosts,
  selectedTargets,
  handleRowSelect,
  setSearchText,
}: ITargetsInputProps) => {
  const tableHeaders = generateTableHeaders();
  const finalRelatedHosts = relatedHosts && pullAllBy(relatedHosts, selectedTargets, "hostname");
  const finalSelectedHostTargets = selectedTargets && filter(selectedTargets, "hostname");
  
  return (
    <div className={baseClass}>
      <Input 
        autofocus={true}
        type="search"
        iconName="search"
        value={searchText}
        tabIndex={tabIndex}
        iconPosition="start"
        label="Target specific hosts"
        placeholder="Search hosts by hostname, UUID, MAC address"
        onChange={setSearchText}
      />
      {finalRelatedHosts.length > 0 && (
        <div className={`${baseClass}__hosts-search-dropdown`}>
          <TableContainer
            columns={tableHeaders}
            data={finalRelatedHosts}
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
            data={finalSelectedHostTargets}
            isLoading={false}
            resultsTitle=""
            emptyComponent={() => <></>}
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