import React from "react";
import { CellProps, Column } from "react-table";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle } from "interfaces/software";
import { APPLE_PLATFORM_DISPLAY_NAMES } from "interfaces/platform";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";

export interface EnhancedSoftwareTitle extends ISoftwareTitle {
  isSelected: boolean;
}

type ISelectSoftwareTableConfig = Column<ISoftwareTitle>;
type ITableHeaderProps = IHeaderProps<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type ISelectionCellProps = CellProps<ISoftwareTitle>;

const generateTableConfig = (
  initialSelectedSoftware: number[],
  onSelectAll: (selectAll: boolean) => void,
  onSelectSoftware: (select: boolean, id: number) => void
): ISelectSoftwareTableConfig[] => {
  let initialRender = true;

  const headerConfigs: ISelectSoftwareTableConfig[] = [
    {
      id: "selection",
      disableSortBy: true,
      Header: (cellProps: ITableHeaderProps) => {
        const {
          checked,
          indeterminate,
        } = cellProps.getToggleAllRowsSelectedProps();

        const checkboxProps = {
          value: checked,
          indeterminate,
          onChange: () => {
            onSelectAll(!checked);
            cellProps.toggleAllRowsSelected();
          },
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ISelectionCellProps) => {
        if (initialRender) {
          const isSelected = initialSelectedSoftware.includes(
            cellProps.row.original.id
          );
          console.log("row:", cellProps.row.original.id, isSelected);
          cellProps.row.toggleRowSelected();
        }
        const { checked } = cellProps.row.getToggleRowSelectedProps();
        console.log(
          "row:",
          cellProps.row.original.id,
          cellProps.row.getToggleRowSelectedProps()
        );
        const checkboxProps = {
          value: checked,
          onChange: () => {
            onSelectSoftware(!checked, cellProps.row.original.id);
            cellProps.row.toggleRowSelected();
          },
        };
        initialRender = false;
        return <Checkbox {...checkboxProps} />;
      },
    },
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, source } = cellProps.row.original;
        return <SoftwareNameCell name={name} source={source} />;
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Platform",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        // TODO: this will need to be updated when we add support for other platforms
        <TextCell value={APPLE_PLATFORM_DISPLAY_NAMES.darwin} />
      ),
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export default generateTableConfig;
