import React from "react";

import { IVulnerability } from "interfaces/vulnerability";
import { formatFloatAsPercentage } from "utilities/helpers";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}
interface ICellProps {
  cell: {
    value: number | string | string[];
  };
  row: {
    original: IVulnerability;
    index: number;
  };
}

interface ITextCellProps extends ICellProps {
  cell: {
    value: string | number;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ITextCellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const formatSeverity = (float: number | null) => {
  if (float === null) {
    return "---";
  }

  let severity = "";
  if (float < 4.0) {
    severity = "Low";
  } else if (float < 7.0) {
    severity = "Medium";
  } else if (float < 9.0) {
    severity = "High";
  } else if (float <= 10.0) {
    severity = "Critical";
  }

  return `${severity} (${float.toFixed(1)})`;
};

const generateVulnTableHeaders = (isPremiumTier: boolean): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Vunerability",
      accessor: "cve",
      disableSortBy: true,
      Header: "Vulnerability",
      Cell: ({ cell: { value }, row }: ITextCellProps) => {
        return (
          <a
            className="text-link"
            href={row.original.details_link}
            target="_blank"
            rel="noopener noreferrer"
          >
            <span>
              {value}
              <img src={ExternalLinkIcon} alt={`link to ${value}`} />
            </span>
          </a>
        );
      },
    },
  ];

  const premiumHeaders: IDataColumn[] = [
    {
      title: "Probability of exploit",
      accessor: "epss_probability",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={`
            The probability that this vulnerability will be exploited in the next 30 days (EPSS probability).<br />
            This data is reported by FIRST.org.
          `}
          >
            Probability of exploit
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={headerProps.column.isSortedDesc}
          />
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => (
        <TextCell formatter={formatFloatAsPercentage} value={value} />
      ),
    },
    {
      title: "Severity",
      accessor: "cvss_score",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={`
            The worst case impact across different environments (CVSS base score).<br />
            This data is reported by the National Vulnerability Database (NVD).
          `}
          >
            Severity
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={headerProps.column.isSortedDesc}
          />
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => (
        <TextCell formatter={formatSeverity} value={value} />
      ),
    },
    {
      title: "Known exploit",
      accessor: "cisa_known_exploit",
      disableSortBy: false,
      sortType: "boolean",
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={`
            The vulnerability has been actively exploited in the wild. This data is reported by 
            the Cybersecurity and Infrustructure Security Agency (CISA).
          `}
          >
            Known exploit
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={headerProps.column.isSortedDesc}
          />
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => (
        <TextCell value={value ? "Yes" : "No"} />
      ),
    },
  ];

  return isPremiumTier ? tableHeaders.concat(premiumHeaders) : tableHeaders;
};

export default generateVulnTableHeaders;
