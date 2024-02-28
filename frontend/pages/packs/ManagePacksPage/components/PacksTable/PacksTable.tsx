import React, { useCallback, useEffect, useState } from "react";

import { IPack } from "interfaces/pack";
import { IEmptyTableProps } from "interfaces/empty_table";
import Button from "components/buttons/Button";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
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
    (queryData) => {
      const { searchQuery } = queryData;
      setSearchString(searchQuery);
    },
    [setSearchString]
  );

  // TODO: useCallback search string
  const emptyState = () => {
    const emptyPacks: IEmptyTableProps = {
      graphicName: "empty-packs",
      header: "You don't have any packs",
      info:
        "Query packs allow you to schedule recurring queries for your hosts.",
      primaryButton: (
        <Button
          variant="brand"
          className={`${baseClass}__create-button`}
          onClick={onCreatePackClick}
        >
          Create new pack
        </Button>
      ),
    };
    if (searchString) {
      delete emptyPacks.graphicName;
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
      onActionButtonClick: onEnablePackClick,
      buttonText: "Enable",
      variant: "text-icon",
      iconSvg: "check",
    },
    {
      name: "disable",
      onActionButtonClick: onDisablePackClick,
      buttonText: "Disable",
      variant: "text-icon",
      iconSvg: "disable",
    },
  ];
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
          variant: "text-icon",
          onActionButtonClick: onDeletePackClick,
        }}
        secondarySelectActions={secondarySelectActions}
        emptyComponent={() =>
          EmptyTable({
            graphicName: emptyState().graphicName,
            header: emptyState().header,
            info: emptyState().info,
            primaryButton: emptyState().primaryButton,
          })
        }
      />
    </div>
  );
};

export default PacksTable;
