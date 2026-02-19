import React from "react";
import { CellProps, Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle, SoftwareSource } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { SetupExperiencePlatform } from "interfaces/platform";
import AndroidLatestVersionWithTooltip from "components/MDM/AndroidLatestVersionWithTooltip";
import TooltipWrapper from "components/TooltipWrapper";

type IInstallSoftwareTableConfig = Column<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type ISelectionCellProps = CellProps<ISoftwareTitle>;

export const manuallyInstallTooltipText = (
  <>
    Disabled because you manually install Fleet&apos;s agent (
    <b>Bootstrap package {">"} Advanced options</b>). Use your bootstrap package
    to install software during the setup experience.
  </>
);

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
  onInstallSoftware: (select: boolean, id: number) => void,
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
            onInstallSoftware(!checked, cellProps.row.original.id);
            cellProps.row.toggleRowSelected();
          },
        };

        return (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={6}
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <TooltipWrapper
                className={"select-software-table__manual-install-tooltip"}
                tipContent={manuallyInstallTooltipText}
                disableTooltip={
                  disableChildren || !manualAgentInstallBlockingSoftware
                }
                position="top"
                showArrow
                underline={false}
                tipOffset={12}
              >
                <Checkbox
                  disabled={
                    disableChildren || manualAgentInstallBlockingSoftware
                  }
                  {...checkboxProps}
                />
              </TooltipWrapper>
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
        return <TextCell value={displayedVersion} />;
      },
      sortType: "caseInsensitive",
    },
  ];

  return headerConfigs;
};

export default generateTableConfig;
