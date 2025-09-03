import React from "react";
import { CellProps, Column } from "react-table";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { SetupExperiencePlatform } from "interfaces/platform";

export interface IEnhancedSoftwareTitle extends ISoftwareTitle {
  versionForRender: string;
  isSelected: boolean;
}

type ISelectSoftwareTableConfig = Column<IEnhancedSoftwareTitle>;
type ITableHeaderProps = IHeaderProps<IEnhancedSoftwareTitle>;
type ITableStringCellProps = IStringCellProps<IEnhancedSoftwareTitle>;
type ISelectionCellProps = CellProps<IEnhancedSoftwareTitle>;

const generateTableConfig = (
  onSelectAll: (selectAll: boolean) => void,
  onSelectSoftware: (select: boolean, id: number) => void
): ISelectSoftwareTableConfig[] => {
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
        return (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={6}
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <Checkbox disabled={disableChildren} {...checkboxProps} />
            )}
          />
        );
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
        return (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={6}
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <Checkbox disabled={disableChildren} {...checkboxProps} />
            )}
          />
        );
      },
    },
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const { name, source, app_store_app } = cellProps.row.original;

        const url = app_store_app?.icon_url;

        return <SoftwareNameCell name={name} source={source} iconUrl={url} />;
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Version",
      disableSortBy: true,
      accessor: "versionForRender",
      Cell: (cellProps: ITableStringCellProps) => {
        return <TextCell value={cellProps.row.original.versionForRender} />;
      },
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export const generateDataSet = (
  platform: SetupExperiencePlatform,
  swTitles: ISoftwareTitle[]
): IEnhancedSoftwareTitle[] => {
  return swTitles.map((title) => {
    let version = title?.software_package?.version;
    if (version && platform === "linux") {
      // TODO - append package type for linux
      version = version.concat("");
    }
    return {
      ...title,
      versionForRender: version ?? "",
    };
  });
};

export default generateTableConfig;
