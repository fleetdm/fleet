import React, { useCallback, useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { IPack } from "interfaces/pack";
import { IUser } from "interfaces/user";
import Button from "components/buttons/Button";
import permissionUtils from "utilities/permissions";

import TableContainer from "components/TableContainer";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton";
import { generateTableHeaders, generateDataSet } from "./PacksTableConfig";

const baseClass = "packs-list-wrapper";
const noPacksClass = "no-packs";

interface IPacksListWrapperProps {
  onRemovePackClick: any;
  onEnablePackClick: any;
  onDisablePackClick: any;
  onCreatePackClick: any;
  packs?: IPack[];
  isLoading: boolean;
}

const PacksListWrapper = ({
  onRemovePackClick,
  onEnablePackClick,
  onDisablePackClick,
  onCreatePackClick,
  packs,
  isLoading,
}: IPacksListWrapperProps): JSX.Element => {
  const [filteredPacks, setFilteredPacks] = useState<IPack[] | undefined>(
    packs
  );
  const [searchString, setSearchString] = useState<string>("");

  useEffect(() => {
    setFilteredPacks(packs);
  }, [packs]);

  useEffect(() => {
    setFilteredPacks(() => {
      return packs?.filter((pack) => {
        return pack.name.toLowerCase().includes(searchString.toLowerCase());
      });
    });
  }, [packs, searchString, setFilteredPacks]);

  const onQueryChange = useCallback(
    (queryData) => {
      const { searchQuery } = queryData;
      setSearchString(searchQuery);
    },
    [setSearchString]
  );

  const NoPacksComponent = useCallback(() => {
    return (
      <div className={`${noPacksClass}`}>
        <div className={`${noPacksClass}__inner`}>
          <div className={`${noPacksClass}__inner-text`}>
            {searchString ? (
              <>
                <h2>No packs match the current search criteria.</h2>
                <p>
                  Expecting to see packs? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            ) : (
              <>
                <h2>You don&apos;t have any packs</h2>
                <p>
                  Query packs allow you to schedule recurring queries for your
                  hosts.
                </p>
                <Button
                  variant="brand"
                  className={`${baseClass}__create-button`}
                  onClick={onCreatePackClick}
                >
                  Create new pack
                </Button>
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString]);

  const tableHeaders = generateTableHeaders();

  const secondarySelectActions: IActionButtonProps[] = [
    {
      name: "enable",
      onActionButtonClick: onEnablePackClick,
      buttonText: "Enable",
      variant: "text-icon",
      icon: "check",
    },
    {
      name: "disable",
      onActionButtonClick: onDisablePackClick,
      buttonText: "Disable",
      variant: "text-icon",
      icon: "disable",
    },
  ];
  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"packs"}
        columns={tableHeaders}
        data={generateDataSet(filteredPacks)}
        isLoading={isLoading}
        defaultSortHeader={"pack"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable={packs && packs.length > 0}
        disablePagination
        onPrimarySelectActionClick={onRemovePackClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        secondarySelectActions={secondarySelectActions}
        emptyComponent={NoPacksComponent}
      />
    </div>
  );
};

export default PacksListWrapper;
