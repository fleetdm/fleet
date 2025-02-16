import React from "react";
import { CellProps, Column } from "react-table";

import { IHostCertificate } from "interfaces/certificates";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import StatusIndicator from "components/StatusIndicator";
import { monthDayYearFormat } from "utilities/date_format";

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
        return <StatusIndicator value={monthDayYearFormat(cellProps.value)} />;
      },
    },
    {
      Header: "",
      id: "view-all-hosts",
      disableSortBy: true,
      Cell: (cellProps) => {
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
