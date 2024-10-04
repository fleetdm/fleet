import React from "react";
import { CellProps, Column } from "react-table";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle } from "interfaces/software";
import {
  APPLE_PLATFORM_DISPLAY_NAMES,
  ApplePlatform,
} from "interfaces/platform";

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
  onSelectAll: (selectAll: boolean) => void,
  onSelectSoftware: (select: boolean, id: number) => void
): ISelectSoftwareTableConfig[] => {
  const headerConfigs: ISelectSoftwareTableConfig[] = [
    {
      id: "selection",
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
        const { checked } = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: checked,
          onChange: () => {
            onSelectSoftware(!checked, cellProps.row.original.id);
            cellProps.row.toggleRowSelected();
          },
        };
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
        <TextCell
          value={APPLE_PLATFORM_DISPLAY_NAMES[cellProps.value as ApplePlatform]}
        />
      ),
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export default generateTableConfig;
