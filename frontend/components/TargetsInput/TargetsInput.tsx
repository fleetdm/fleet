import { isEmpty, pullAllBy } from "lodash";
import React, { useRef, useEffect, useState } from "react";
import { Row } from "react-table";

import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon/InputFieldWithIcon";
import TableContainer from "components/TableContainer";
import { IHost } from "interfaces/host";
import { HOSTS_SEARCH_BOX_PLACEHOLDER } from "utilities/constants";

import { ITargestInputHostTableConfig } from "./TargetsInputHostsTableConfig";

interface ITargetsInputProps {
  searchText: string;
  searchResults: IHost[];
  isTargetsLoading: boolean;
  hasFetchError: boolean;
  targetedHosts: IHost[];
  searchResultsTableConfig: ITargestInputHostTableConfig[];
  selectedHostsTableConifg: ITargestInputHostTableConfig[];
  /** disabled pagination for the results table. The pagination is currently
   * client side pagination. Defaults to `false` */
  disablePagination?: boolean;
  label?: string;
  placeholder?: string;
  autofocus?: boolean;
  setSearchText: (value: string) => void;
  handleRowSelect: (value: Row<IHost>) => void;
}

const baseClass = "targets-input";

const DEFAULT_LABEL = "Target specific hosts";

const TargetsInput = ({
  searchText,
  searchResults,
  isTargetsLoading,
  hasFetchError,
  targetedHosts,
  searchResultsTableConfig,
  selectedHostsTableConifg,
  disablePagination = false,
  label = DEFAULT_LABEL,
  placeholder = HOSTS_SEARCH_BOX_PLACEHOLDER,
  autofocus = false,
  handleRowSelect,
  setSearchText,
}: ITargetsInputProps): JSX.Element => {
  const searchRef = useRef<HTMLDivElement | null>(null);
  const dropdownHosts =
    searchResults && pullAllBy(searchResults, targetedHosts, "id");

  const [isDismissed, setIsDismissed] = useState(false);
  const showDropdown = !isDismissed && !isEmpty(searchText);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        searchRef.current &&
        !searchRef.current.contains(event.target as Node)
      ) {
        setIsDismissed(true);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div>
      <div className={baseClass}>
        <div ref={searchRef}>
          <InputFieldWithIcon
            autofocus={autofocus}
            type="search"
            iconSvg="search"
            value={searchText}
            label={label}
            placeholder={placeholder}
            onFocus={() => setIsDismissed(false)}
            onChange={(value: string) => {
              setIsDismissed(false);
              setSearchText(value);
            }}
          />
          {showDropdown && !hasFetchError && (
            <div className={`${baseClass}__hosts-search-dropdown`}>
              <TableContainer<Row<IHost>>
                columnConfigs={searchResultsTableConfig}
                data={dropdownHosts}
                isLoading={isTargetsLoading}
                emptyComponent={() => (
                  <EmptyState
                    header="No matching hosts"
                    info="Expecting to see hosts? Try again in a few seconds as the system catches up."
                  />
                )}
                showMarkAllPages={false}
                isAllPagesSelected={false}
                disableCount
                disablePagination
                disableMultiRowSelect
                onClickRow={handleRowSelect}
                keyboardSelectableRows
              />
            </div>
          )}
          {showDropdown && hasFetchError && (
            <div className={`${baseClass}__hosts-search-dropdown`}>
              <DataError />
            </div>
          )}
        </div>
        <div className={`${baseClass}__hosts-selected-table`}>
          <TableContainer
            columnConfigs={selectedHostsTableConifg}
            data={targetedHosts}
            isLoading={false}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disablePagination={disablePagination}
            isClientSidePagination={!disablePagination}
            emptyComponent={() => <></>}
          />
        </div>
      </div>
    </div>
  );
};

export default TargetsInput;
