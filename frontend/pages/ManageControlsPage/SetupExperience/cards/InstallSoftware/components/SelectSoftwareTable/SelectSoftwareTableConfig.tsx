import React from "react";
import { CellProps, Column } from "react-table";

import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { SetupExperiencePlatform } from "interfaces/platform";

type ISelectSoftwareTableConfig = Column<ISoftwareTitle>;
type ITableHeaderProps = IHeaderProps<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type ISelectionCellProps = CellProps<ISoftwareTitle>;

const generateTableConfig = (
  platform: SetupExperiencePlatform,
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
      Cell: (cellProps: ITableStringCellProps) => {
        let versionFoRender = cellProps.row.original.software_package?.version;
        if (versionFoRender && platform === "linux") {
          // TODO - append package type for linux
          versionFoRender = versionFoRender.concat("");
        }
        return <TextCell value={versionFoRender} />;
      },
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

// export const generateDataSet = (
//   platform: SetupExperiencePlatform,
//   swTitles: ISoftwareTitle[]
// ): IEnhancedSoftwareTitle[] => {
//   return swTitles.map((title) => {
//     let version = title?.software_package?.version;
//     if (version && platform === "linux") {
//       // TODO - append package type for linux
//       version = version.concat("");
//     }
//     return {
//       ...title,
//       versionForRender: version ?? "",
//     };
//   });
// };

export default generateTableConfig;
