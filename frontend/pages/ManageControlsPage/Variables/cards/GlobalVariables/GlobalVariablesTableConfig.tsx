import React from "react";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import Icon from "components/Icon";
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
      title: "Variable name",
      Header: "Variable name",
      disableSortBy: true,
      accessor: "id",
      Cell: (cellProps) => {
        const token = getTokenFromVariableName(cellProps.row.original.name);
        return (
          <TooltipTruncatedTextCell
            value={token}
            suffix={
              <CopyButton
                copyText={token}
                variant="subdued"
                size="small"
                rowHover
              />
            }
            justifySuffixEnd
            className="w400"
          />
        );
      },
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
  ];

  // Non-write roles don't get row actions. Global variables support delete
  // only (no edit). Delete is allowed in GitOps mode, matching prior behavior.
  if (canEdit) {
    columns.push({
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => {
        const variable = cellProps.row.original;
        return (
          <div className="global-variables__actions">
            <Button
              variant="secondary"
              size="small"
              onClick={() => onDelete(variable)}
              ariaLabel={`Delete ${variable.name}`}
            >
              <Icon name="trash" size="small" />
            </Button>
          </div>
        );
      },
    });
  }

  return columns;
};

export default generateTableHeaders;
