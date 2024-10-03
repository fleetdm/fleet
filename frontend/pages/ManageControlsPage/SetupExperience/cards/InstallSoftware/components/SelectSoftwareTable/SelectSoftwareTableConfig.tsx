import React from "react";

import { IStringCellProps } from "interfaces/datatable_config";
import { ISoftwareTitle } from "interfaces/software";
import {
  APPLE_PLATFORM_DISPLAY_NAMES,
  ApplePlatform,
} from "interfaces/platform";

import TextCell from "components/TableContainer/DataTable/TextCell";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import { Column } from "react-table";

type ISelectSoftwareTableConfig = Column<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;

const generateTableHeaders = (): ISelectSoftwareTableConfig[] => {
  const headerConfigs: ISelectSoftwareTableConfig[] = [
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => (
        <SoftwareNameCell name={cellProps.name} source={cellProps.source} />
      ),
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

export default generateTableHeaders;
