import React from "react";

import { ICustomHostVital } from "interfaces/custom_host_vitals";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import Icon from "components/Icon";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

export const getTokenFromVitalId = (id: number): string =>
  `$FLEET_HOST_VITAL_${id}`;

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IStringCellProps {
  cell: { value: string };
  row: { original: ICustomHostVital };
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
  onEdit: (vital: ICustomHostVital) => void;
  onDelete: (vital: ICustomHostVital) => void;
}

const generateTableHeaders = ({
  canEdit,
  onEdit,
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
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Variable",
      Header: "Variable",
      disableSortBy: true,
      accessor: "id",
      Cell: (cellProps) => {
        const token = getTokenFromVitalId(cellProps.row.original.id);
        return (
          <div className="custom-host-vitals-tab__token">
            <TextCell value={token} />
            <CopyButton copyText={token} variant="subdued" size="small" />
          </div>
        );
      },
    },
    {
      title: "Updated",
      Header: "Updated",
      disableSortBy: true,
      accessor: "updated_at",
      Cell: (cellProps) => (
        <HumanTimeDiffWithDateTip timeString={cellProps.cell.value} />
      ),
    },
  ];

  // Non-write roles don't get row actions. In GitOps mode the actions are shown
  // but disabled with the standard GitOps tooltip (matching the "Add vital"
  // button), since custom host vitals are then managed via the config file.
  if (canEdit) {
    columns.push({
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => {
        const vital = cellProps.row.original;
        return (
          <GitOpsModeTooltipWrapper
            position="top"
            fixedPositionStrategy
            renderChildren={(disableChildren) => (
              <div className="custom-host-vitals-tab__actions">
                <Button
                  variant="secondary"
                  size="small"
                  disabled={disableChildren}
                  onClick={() => onEdit(vital)}
                  ariaLabel={`Edit ${vital.name}`}
                >
                  <Icon name="pencil" size="small" />
                </Button>
                <Button
                  variant="secondary"
                  size="small"
                  disabled={disableChildren}
                  onClick={() => onDelete(vital)}
                  ariaLabel={`Delete ${vital.name}`}
                >
                  <Icon name="trash" size="small" />
                </Button>
              </div>
            )}
          />
        );
      },
    });
  }

  return columns;
};

export default generateTableHeaders;
