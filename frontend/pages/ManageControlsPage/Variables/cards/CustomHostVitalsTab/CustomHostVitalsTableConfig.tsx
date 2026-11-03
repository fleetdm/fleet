import React from "react";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import { ICustomHostVital } from "interfaces/custom_host_vitals";

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
          value={getTokenFromVitalId(cellProps.row.original.id)}
          className="w400"
        />
      ),
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
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => {
        const vital = cellProps.row.original;
        return (
          <div className="custom-host-vitals-tab__actions">
            <CopyButton
              copyText={getTokenFromVitalId(vital.id)}
              variant="secondary"
              size="small"
              ariaLabel={`Copy ${vital.name}`}
              tooltip="Copy the variable"
            />
            {/* In GitOps mode edit/delete are shown but disabled with the
                standard GitOps tooltip (matching the "Add vital" button), since
                custom host vitals are then managed via the config file. */}
            {canEdit && (
              <GitOpsModeTooltipWrapper
                position="top"
                fixedPositionStrategy
                renderChildren={(disableChildren) => (
                  <div className="custom-host-vitals-tab__actions">
                    <Button
                      variant="secondary"
                      size="small"
                      icon="pencil"
                      disabled={disableChildren}
                      onClick={() => onEdit(vital)}
                      ariaLabel={`Edit ${vital.name}`}
                    />
                    <Button
                      variant="secondary"
                      size="small"
                      icon="trash"
                      disabled={disableChildren}
                      onClick={() => onDelete(vital)}
                      ariaLabel={`Delete ${vital.name}`}
                    />
                  </div>
                )}
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
