import React from "react";

import { ICustomHostVital } from "interfaces/custom_host_vitals";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import Icon from "components/Icon";

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
  gitOpsModeEnabled: boolean;
  onEdit: (vital: ICustomHostVital) => void;
  onDelete: (vital: ICustomHostVital) => void;
}

const generateTableHeaders = ({
  canEdit,
  gitOpsModeEnabled,
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
            <CopyButton copyText={token} variant="compact" />
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

  // Non-write roles and GitOps mode don't get row actions.
  if (canEdit && !gitOpsModeEnabled) {
    columns.push({
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => {
        const vital = cellProps.row.original;
        return (
          <div className="custom-host-vitals-tab__actions">
            <Button
              variant="icon"
              onClick={() => onEdit(vital)}
              ariaLabel={`Edit ${vital.name}`}
            >
              <Icon name="pencil" color="ui-fleet-black-75" />
            </Button>
            <Button
              variant="icon"
              onClick={() => onDelete(vital)}
              ariaLabel={`Delete ${vital.name}`}
            >
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          </div>
        );
      },
    });
  }

  return columns;
};

export default generateTableHeaders;
