import React from "react";
import { Row } from "react-table";

import { IHost } from "interfaces/host";
import { ITarget } from "interfaces/target";
import { filter, isEmpty, pullAllBy } from "lodash";

// @ts-ignore
import Input from "components/forms/fields/InputFieldWithIcon";
import TableContainer from "components/TableContainer";
import { generateTableHeaders } from "./TargetsInputHostsTableConfig";

interface ITargetsInputProps {
  tabIndex: number;
  searchText: string;
  relatedHosts: IHost[];
  selectedTargets: ITarget[];
  setSearchText: (value: string) => void;
  handleRowSelect: (value: Row) => void;
  onPrimarySelectActionClick: (value: number[]) => void;
}

const baseClass = "targets-input";

const TargetsInput = ({
  tabIndex,
  searchText,
  relatedHosts,
  selectedTargets,
  handleRowSelect,
  setSearchText,
  onPrimarySelectActionClick,
}: ITargetsInputProps) => {
  const resultsDropdownTableHeaders = generateTableHeaders(false);
  const selectedTableHeaders = generateTableHeaders(true);

  const finalRelatedHosts =
    relatedHosts && pullAllBy(relatedHosts, selectedTargets, "hostname");
  const finalSelectedHostTargets =
    selectedTargets && filter(selectedTargets, "hostname");

  return (
    <div className={baseClass}>
      <Input
        autofocus
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
            columns={resultsDropdownTableHeaders}
            data={finalRelatedHosts}
            isLoading={false}
            resultsTitle=""
            emptyComponent={() => <></>}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disablePagination
            disableMultiRowSelect
            onSelectSingleRow={handleRowSelect}
          />
        </div>
      )}
      {finalRelatedHosts.length === 0 && !isEmpty(searchText) && (
        <div className={`${baseClass}__hosts-search-dropdown`}>
          <div className="empty-search">
            <div className="empty-search__inner">
              <h4>No hosts match the current search criteria.</h4>
              <p>
                Expecting to see hosts? Try again in a few seconds as the system
                catches up.
              </p>
            </div>
          </div>
        </div>
      )}
      <div className={`${baseClass}__hosts-selected-table`}>
        <TableContainer
          resultsTitle=""
          columns={selectedTableHeaders}
          data={finalSelectedHostTargets}
          isLoading={false}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disablePagination
          onPrimarySelectActionClick={onPrimarySelectActionClick}
          primarySelectActionButtonVariant="text-link"
          primarySelectActionButtonIcon="close"
          primarySelectActionButtonText={"Remove"}
          emptyComponent={() => <></>}
        />
      </div>
    </div>
  );
};

export default TargetsInput;
