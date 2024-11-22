import React from "react";
import { CellProps, Column } from "react-table";

import { IMdmAbmToken } from "interfaces/mdm";
import { IPkiConfig } from "interfaces/pki";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ActionsDropdown from "components/ActionsDropdown";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

import RenewDateCell from "../../../components/RenewDateCell";
import OrgNameCell from "./OrgNameCell";
import { IRenewDateCellStatusConfig } from "../../../components/RenewDateCell/RenewDateCell";

type IPkiTableConfig = Column<IPkiConfig>;
type ITableStringCellProps = IStringCellProps<IPkiConfig>;
type IPkiTemplatesCellProps = CellProps<IPkiConfig, IPkiConfig["templates"]>;

type ITableHeaderProps = IHeaderProps<IPkiConfig>;

const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "view_template", label: "View template", disabled: false },
  // { value: "renew", label: "Renew", disabled: false },
  { value: "delete", label: "Delete", disabled: false },
];

const generateActions = () => {
  return DEFAULT_ACTION_OPTIONS;
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, pkiConfig: IPkiConfig) => void
): IPkiTableConfig[] => {
  return [
    {
      accessor: "name",
      sortType: "caseInsensitive",
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      Cell: (cellProps: ITableStringCellProps) => {
        const { name } = cellProps.cell.row.original;
        return <TextCell value={name} />;
      },
    },
    {
      accessor: "templates",
      Header: "Certificate template",
      disableSortBy: true,
      Cell: ({ value: templates }: IPkiTemplatesCellProps) => {
        return <TextCell value={templates.length ? "✅ Added" : "---"} />; // TODO: use our own icon
      },
    },
    {
      Header: "",
      id: "actions",
      disableSortBy: true,
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: () => "name",
      Cell: (cellProps: CellProps<IPkiConfig, IPkiConfig["name"]>) => (
        <ActionsDropdown
          options={generateActions()}
          onChange={(action: string) =>
            actionSelectHandler(action, cellProps.row.original)
          }
          placeholder="Actions"
        />
      ),
    },
  ];
};

export const generateTableData = (data: IMdmAbmToken[]) => {
  return data;
};
