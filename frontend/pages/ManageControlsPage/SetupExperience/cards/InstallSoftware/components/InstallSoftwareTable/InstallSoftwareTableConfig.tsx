import React from "react";
import { CellProps, Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle, SoftwareSource } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import { SetupExperiencePlatform } from "interfaces/platform";
import AndroidLatestVersionWithTooltip from "components/MDM/AndroidLatestVersionWithTooltip";

type IInstallSoftwareTableConfig = Column<ISoftwareTitle>;
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
  onSelectSoftware: (select: boolean, id: number) => void,
  manualAgentInstallBlockingSoftware = false
): IInstallSoftwareTableConfig[] => {
  const headerConfigs: IInstallSoftwareTableConfig[] = [
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
              <Checkbox
                disabled={disableChildren || manualAgentInstallBlockingSoftware}
                {...checkboxProps}
              />
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
            name={name}
            display_name={display_name}
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
        if (platform === "android") {
          const androidPlayStoreId =
            cellProps.row.original.app_store_app?.app_store_id;

          return (
            <TextCell
              value={
                <AndroidLatestVersionWithTooltip
                  androidPlayStoreId={androidPlayStoreId || ""}
                />
              }
            />
          );
        }

        const title = cellProps.row.original;
        let displayedVersion =
          title.software_package?.version || title.app_store_app?.version;

        if (platform === "linux") {
          const packageTypeCopy = getSetupExperienceLinuxPackageCopy(
            title.source
          );
          if (packageTypeCopy) {
            displayedVersion = (
              displayedVersion ?? DEFAULT_EMPTY_CELL_VALUE
            ).concat(` (.${packageTypeCopy})`);
          }
        }

        // Setup experience only installs the first-added package on a
        // multi-package custom title; surface a tooltip so the admin knows
        // the other packages aren't in scope here. Single-package titles
        // (VPP, FMA, and any custom title with only one package) don't need
        // the disclaimer — there's no "other" to explain away.
        const isMultiPackageCustom = (title.packages?.length ?? 0) > 1;
        if (isMultiPackageCustom) {
          return (
            <TextCell
              value={
                <TooltipWrapper
                  tipContent="For custom packages, the first added version will be installed."
                  showArrow
                >
                  {displayedVersion}
                </TooltipWrapper>
              }
            />
          );
        }
        return <TextCell value={displayedVersion} />;
      },
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export default generateTableConfig;
