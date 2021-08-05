import React, { useCallback, useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { IPack } from "interfaces/pack";
import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./PacksTableConfig";

const baseClass = "packs-list-wrapper";
const noPacksClass = "no-packs";

interface IPacksListWrapperProps {
  onRemovePackClick: any;
  packsList: IPack[];
}

interface IRootState {
  auth: {
    user: IUser;
  };
  entities: {
    packs: {
      isLoading: boolean;
      data: IPack[];
    };
  };
}

const PacksListWrapper = (props: IPacksListWrapperProps): JSX.Element => {
  const { onRemovePackClick, packsList } = props;

  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.packs.isLoading
  );

  const currentUser = useSelector((state: IRootState) => state.auth.user);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);

  const [filteredPacks, setFilteredPacks] = useState<IPack[]>(packsList);
  const [searchString, setSearchString] = useState<string>("");

  useEffect(() => {
    setFilteredPacks(packsList);
  }, [packsList]);

  useEffect(() => {
    setFilteredPacks(() => {
      return packsList.filter((pack) => {
        // return pack.name.toLowerCase().includes(searchString.toLowerCase());
        return pack.name.toLowerCase();
      });
    });
  }, [packsList, searchString, setFilteredPacks]);

  const onQueryChange = useCallback(
    (packData) => {
      const { searchPack } = packData;
      setSearchString(searchPack);
    },
    [setSearchString]
  );

  console.log("DONT FORGET TO CHANGE NO PACKS COMPONENT");

  const NoPacksComponent = useCallback(() => {
    return (
      <div className={`${noPacksClass}`}>
        <div className={`${noPacksClass}__inner`}>
          <div className={`${noPacksClass}__inner-text`}>
            {!searchString ? (
              <h2>You don&apos;t have any packs.</h2>
            ) : (
              <h2>No packs match your search.</h2>
            )}
            <p>
              CHANGE ME CHANGE ME CHANGE ME Create a new pack, or{" "}
              <a href="https://github.com/fleetdm/fleet/tree/main/docs/1-Using-Fleet/standard-query-library">
                go to GitHub
              </a>{" "}
              to import Fleet’s standard query library.
            </p>
          </div>
        </div>
      </div>
    );
  }, [searchString]);

  const tableHeaders = generateTableHeaders(isOnlyObserver);

  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"packs"}
        columns={tableHeaders}
        data={generateDataSet(filteredPacks)}
        isLoading={loadingTableData}
        defaultSortHeader={"pack"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable
        disablePagination
        onPrimarySelectActionClick={onRemovePackClick}
        primarySelectActionButtonVariant="text-link"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={NoPacksComponent}
      />
    </div>
  );
};

export default PacksListWrapper;
