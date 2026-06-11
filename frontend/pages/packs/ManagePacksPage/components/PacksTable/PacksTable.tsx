import React, { useCallback, useEffect, useState } from "react";

import { IPack } from "interfaces/pack";
import { IEmptyStateProps } from "interfaces/empty_state";
import Button from "components/buttons/Button";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyState from "components/EmptyState";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton/ActionButton";
import { generateTableHeaders, generateDataSet } from "./PacksTableConfig";

const baseClass = "packs-table";

interface IPacksTableProps {
  onDeletePackClick: (selectedTablePackIds: number[]) => void;
  onEnablePackClick: (selectedTablePackIds: number[]) => void;
  onDisablePackClick: (selectedTablePackIds: number[]) => void;
  onCreatePackClick: (
    event: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => void;
  packs?: IPack[];
  isLoading: boolean;
}

const PacksTable = ({
  onDeletePackClick,
  onEnablePackClick,
  onDisablePackClick,
  onCreatePackClick,
  packs,
  isLoading,
}: IPacksTableProps): JSX.Element => {
  const [filteredPacks, setFilteredPacks] = useState<IPack[] | undefined>(
    packs
  );
  const [searchString, setSearchString] = useState("");

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
    (queryData: ITableQueryData) => {
      const { searchQuery } = queryData;
      setSearchString(searchQuery);
    },
    [setSearchString]
  );

  // TODO: useCallback search string
  const emptyState = () => {
    const emptyPacks: IEmptyStateProps = {
      header: "You don't have any packs",
      info:
        "Query packs allow you to schedule recurring queries for your hosts.",
      primaryButton: (
        <Button
          className={`${baseClass}__create-button`}
          onClick={onCreatePackClick}
        >
          Create new pack
        </Button>
      ),
    };
    if (searchString) {
      emptyPacks.header = "No packs match the current search criteria";
      emptyPacks.info =
        "Expecting to see packs? Try again in a few seconds as the system catches up.";
      delete emptyPacks.primaryButton;
    }
    return emptyPacks;
  };

  const tableHeaders = generateTableHeaders();

  const secondarySelectActions: IActionButtonProps[] = [
    {
      name: "enable",
      onClick: onEnablePackClick,
      buttonText: "Enable",
      variant: "inverse",
      iconSvg: "check",
    },
    {
      name: "disable",
      onClick: onDisablePackClick,
      buttonText: "Disable",
      variant: "inverse",
      iconSvg: "disable",
    },
  ];

  const renderPackCount = useCallback(() => {
    return <TableCount name="packs" count={filteredPacks?.length || 0} />;
  }, [filteredPacks]);

  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle="packs"
        columnConfigs={tableHeaders}
        data={generateDataSet(filteredPacks)}
        isLoading={isLoading}
        defaultSortHeader="pack"
        defaultSortDirection="desc"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable={packs && packs.length > 0}
        disablePagination
        primarySelectAction={{
          name: "delete pack",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "inverse",
          onClick: onDeletePackClick,
        }}
        renderCount={renderPackCount}
        secondarySelectActions={secondarySelectActions}
        emptyComponent={() => (
          <EmptyState
            header={emptyState().header}
            info={emptyState().info}
            primaryButton={emptyState().primaryButton}
          />
        )}
      />
    </div>
  );
};

export default PacksTable;
