import React from "react";
import { CellProps, Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle, SoftwareSource } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { SetupExperiencePlatform } from "interfaces/platform";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

type ISelectSoftwareTableConfig = Column<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type ISelectionCellProps = CellProps<ISoftwareTitle>;

const getSetupExperienceLinuxPackageCopy = (source: SoftwareSource) => {
  switch (source) {
    case "rpm_packages":
      return "rpm";
    case "deb_packages":
      return "deb";
    case "tgz_packages":
      return "tar";
    default:
      return null;
  }
};

const generateTableConfig = (
  platform: SetupExperiencePlatform,
  onSelectSoftware: (select: boolean, id: number) => void
): ISelectSoftwareTableConfig[] => {
  const headerConfigs: ISelectSoftwareTableConfig[] = [
    {
      id: "selection",
      disableSortBy: true,
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
        const { name, display_name, source, icon_url } = cellProps.row.original;

        return (
          <SoftwareNameCell
            name={display_name || name}
            source={source}
            iconUrl={icon_url}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Version",
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => {
        const title = cellProps.row.original;
        let versionFoRender = title.software_package?.version;
        if (platform === "linux") {
          const packageTypeCopy = getSetupExperienceLinuxPackageCopy(
            title.source
          );
          if (packageTypeCopy) {
            versionFoRender = (
              versionFoRender ?? DEFAULT_EMPTY_CELL_VALUE
            ).concat(` (.${packageTypeCopy})`);
          }
        }
        return <TextCell value={versionFoRender} />;
      },
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export default generateTableConfig;
