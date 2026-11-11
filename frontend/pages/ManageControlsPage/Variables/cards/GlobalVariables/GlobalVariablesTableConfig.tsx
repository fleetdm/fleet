import React from "react";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import { IVariable } from "interfaces/variables";

export const getTokenFromVariableName = (variableName: string): string =>
  `$FLEET_SECRET_${variableName.toUpperCase()}`;

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IStringCellProps {
  cell: { value: string };
  row: { original: IVariable };
}

interface IDataColumn {
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: IStringCellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

interface IGenerateTableHeadersParams {
  canEdit: boolean;
  onDelete: (variable: IVariable) => void;
}

const generateTableHeaders = ({
  canEdit,
  onDelete,
}: IGenerateTableHeadersParams): IDataColumn[] => {
  const columns: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps) => (
        <TooltipTruncatedTextCell
          value={cellProps.cell.value}
          className="w250"
        />
      ),
    },
    {
      title: "Variable",
      Header: "Variable",
      disableSortBy: true,
      accessor: "id",
      Cell: (cellProps) => (
        <TooltipTruncatedTextCell
          value={getTokenFromVariableName(cellProps.row.original.name)}
          className="w400"
        />
      ),
    },
    {
      title: "Created",
      Header: "Created",
      disableSortBy: true,
      accessor: "created_at",
      Cell: (cellProps) => (
        <HumanTimeDiffWithDateTip timeString={cellProps.cell.value} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => {
        const variable = cellProps.row.original;
        return (
          <div className="global-variables__actions">
            <CopyButton
              copyText={getTokenFromVariableName(variable.name)}
              variant="secondary"
              size="small"
              ariaLabel={`Copy ${variable.name}`}
              tooltip="Copy the variable"
            />
            {/* Global variables support delete only (no edit); delete stays
                enabled in GitOps mode. */}
            {canEdit && (
              <Button
                variant="secondary"
                size="small"
                icon="trash"
                onClick={() => onDelete(variable)}
                ariaLabel={`Delete ${variable.name}`}
              />
            )}
          </div>
        );
      },
    },
  ];

  return columns;
};

export default generateTableHeaders;
