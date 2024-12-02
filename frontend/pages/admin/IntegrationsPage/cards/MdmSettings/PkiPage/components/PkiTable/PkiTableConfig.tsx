import React from "react";
import { CellProps, Column } from "react-table";

import { IMdmAbmToken } from "interfaces/mdm";
import { IPkiConfig } from "interfaces/pki";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ActionsDropdown from "components/ActionsDropdown";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Icon from "components/Icon";

type IPkiTableConfig = Column<IPkiConfig>;
type ITableStringCellProps = IStringCellProps<IPkiConfig>;
type IPkiTemplatesCellProps = CellProps<IPkiConfig, IPkiConfig["templates"]>;

type ITableHeaderProps = IHeaderProps<IPkiConfig>;

const generateActions = (pkiConfig: IPkiConfig): IDropdownOption[] => {
  return [
    {
      value: pkiConfig.templates?.length ? "view_template" : "add_template",
      label: pkiConfig.templates?.length ? "View template" : "Add template",
      disabled: false,
    },
    // {
    //   value: "delete",
    //   label: "Delete",
    //   disabled: false,
    // },
  ];
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, pkiConfig: IPkiConfig) => void
): IPkiTableConfig[] => {
  return [
    {
      accessor: "pki_name",
      sortType: "caseInsensitive",
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      Cell: (cellProps: ITableStringCellProps) => {
        const { pki_name: name } = cellProps.cell.row.original;
        return <TextCell value={name} />;
      },
    },
    {
      accessor: "templates",
      Header: "Certificate template",
      disableSortBy: true,
      Cell: ({ value: templates }: IPkiTemplatesCellProps) => {
        return templates.length ? (
          // FIXME: See related note in frontend/components/StatusIndicatorWithIcon/StatusIndicatorWithIcon.tsx
          <span className="status-indicator-with-icon__value">
            <Icon name="success" />
            <span>Added</span>
          </span>
        ) : (
          <TextCell value={"---"} />
        );
      },
    },
    {
      Header: "",
      id: "actions",
      disableSortBy: true,
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: () => "name",
      Cell: (cellProps: CellProps<IPkiConfig, IPkiConfig["pki_name"]>) => (
        <ActionsDropdown
          options={generateActions(cellProps.row.original)}
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
