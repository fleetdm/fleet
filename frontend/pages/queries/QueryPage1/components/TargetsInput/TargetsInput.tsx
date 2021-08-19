import React, { useState } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";

import hostsAPI from "services/entities/hosts";
import { IHost } from "interfaces/host";

// @ts-ignore
import Input from "components/forms/fields/InputFieldWithIcon";
import TableContainer from "components/TableContainer";
import { generateTableHeaders } from "./TargetsInputHostsTableConfig";

interface IHostsQueryResponse {
  hosts: IHost[];
}

interface ITargetsInputProps {
  tabIndex: number;
  handleRowSelect: (value: Row) => void;
}

const baseClass = "targets-input";

const EmptyHosts = () => (
  <p>No hosts match the current search criteria.</p>
);

const TargetsInput = ({
  tabIndex,
  handleRowSelect,
}: ITargetsInputProps) => {
  const [searchText, setSearchText] = useState<string>("");
  
  const { 
    status: hostsLoadedStatus, 
    data: { hosts: loadedHosts } = {}, 
    error: hostsLoadedError 
  } = useQuery<IHostsQueryResponse, Error>(
    ["hostsFromInput", searchText], 
    () => hostsAPI.search(searchText), {
      enabled: !!searchText,
      refetchOnWindowFocus: false,
    }
  );

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
      {loadedHosts && (
        <div className={`${baseClass}__hosts-search-dropdown`}>
          <TableContainer
            columns={tableHeaders}
            data={loadedHosts}
            isLoading={hostsLoadedStatus === "loading"}
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

      </div>
    </div>
  )
};

export default TargetsInput;