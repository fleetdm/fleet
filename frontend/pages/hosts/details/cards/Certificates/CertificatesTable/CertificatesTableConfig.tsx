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

type IHostCertificatesTableConfig = Column<IHostCertificate>;

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
