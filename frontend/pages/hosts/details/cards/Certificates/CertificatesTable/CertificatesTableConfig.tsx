import React from "react";
import { Column } from "react-table";

import { IHostCertificate } from "interfaces/certificates";
import { monthDayYearFormat } from "utilities/date_format";
import { hasExpired, willExpireWithinXDays } from "utilities/helpers";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import StatusIndicator from "components/StatusIndicator";
import { IIndicatorValue } from "components/StatusIndicator/StatusIndicator";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IStringCellProps } from "interfaces/datatable_config";

type IHostCertificatesTableConfig = Column<IHostCertificate>;
type IIssuerCellProps = IStringCellProps<IHostCertificate>;

const generateTableConfig = (): IHostCertificatesTableConfig[] => {
  return [
    {
      accessor: "common_name",
      Header: (cellProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      accessor: (data) => data.issuer.common_name,
      id: "issuer",
      disableSortBy: true,
      Header: "Issuer",
      Cell: (cellProps: IIssuerCellProps) => (
        <TooltipTruncatedTextCell
          value={cellProps.cell.value}
          tooltip={cellProps.cell.value}
        />
      ),
    },
    {
      accessor: "source",
      disableSortBy: true,
      Header: "Keychain",
      Cell: (cellProps) => {
        if (cellProps.cell.value === "system") {
          return <TextCell value="System" />;
        }
        return (
          <TooltipWrapper
            tipContent={cellProps.cell.row.original.username || "Unknown user"}
          >
            User
          </TooltipWrapper>
        );
      },
    },
    {
      accessor: "not_valid_before",
      disableSortBy: true,
      Header: "Issued",
      Cell: (cellProps) => {
        return (
          <TextCell
            value={monthDayYearFormat(cellProps.value)}
            className="text-nowrap"
          />
        );
      },
    },
    {
      accessor: "not_valid_after",
      Header: (cellProps) => (
        <HeaderCell
          value="Expires"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: (cellProps) => {
        let status: IIndicatorValue = "success";
        if (hasExpired(cellProps.value)) {
          status = "error";
        } else if (willExpireWithinXDays(cellProps.value, 30)) {
          status = "warning";
        }
        return (
          <StatusIndicator
            className="cert-table__status-indicator"
            value={monthDayYearFormat(cellProps.value)}
            indicator={status}
          />
        );
      },
    },
    {
      Header: "",
      id: "view-all-hosts",
      disableSortBy: true,
      Cell: () => {
        return (
          <ViewAllHostsLink
            className="view-cert-details"
            noLink
            rowHover
            excludeChevron
            customText="View details"
          />
        );
      },
    },
  ];
};

export default generateTableConfig;
