import React from "react";
import { CellProps, Column } from "react-table";

import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import AndroidLatestVersionWithTooltip from "components/MDM/AndroidLatestVersionWithTooltip";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IStringCellProps } from "interfaces/datatable_config";
import { SetupExperiencePlatform } from "interfaces/platform";
import { ISoftwareTitle, SoftwareSource } from "interfaces/software";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

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
      id: "name",
      // `disableSortBy` doesn't stop TableContainer's default sort header from
      // ordering this column, so the sort key has to be the rendered string.
      accessor: (originalRow) =>
        getDisplayedSoftwareName(originalRow.name, originalRow.display_name),
      // The search box filters this column, which now holds the display name;
      // keep the raw name matchable so packages stay findable by filename.
      filter: (rows, _columnIds, query) => {
        const q = String(query).toLowerCase();
        return rows.filter(({ original }) =>
          [
            getDisplayedSoftwareName(original.name, original.display_name),
            original.name,
          ].some((field) => field.toLowerCase().includes(q))
        );
      },
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
      id: "version",
      Header: () => (
        <TooltipWrapper
          tipContent={
            <>
              For custom packages, the first
              <br />
              added version will be installed.
            </>
          }
        >
          Version
        </TooltipWrapper>
      ),
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
