import React from "react";

import { formatFloatAsPercentage } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { ISoftwareVulnerability } from "interfaces/software";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";

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
    original: ISoftwareVulnerability;
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

const formatSeverity = (float?: number | null) => {
  if (float === null || float === undefined) {
    return DEFAULT_EMPTY_CELL_VALUE;
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

const generateTableConfig = (
  isPremiumTier: boolean,
  isSandboxMode: boolean
): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Vunerability",
      accessor: "cve",
      disableSortBy: true,
      Header: "Vulnerability",
      Cell: ({ cell: { value }, row }: ITextCellProps) => {
        return (
          <CustomLink
            url={row.original.details_link}
            text={value.toString()}
            newTab
          />
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
            tipContent={
              <>
                The probability that this vulnerability will be exploited in the
                next 30 days (EPSS probability).
                <br />
                This data is reported by FIRST.org.
              </>
            }
          >
            Probability of exploit
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithToolTip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
            {isSandboxMode && <PremiumFeatureIconWithTooltip />}
          </>
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
            tipContent={
              <>
                The worst case impact across different environments (CVSS base
                score).
                <br />
                This data is reported by the National Vulnerability Database
                (NVD).
              </>
            }
          >
            Severity
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithToolTip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
            {isSandboxMode && <PremiumFeatureIconWithTooltip />}
          </>
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
            tipContent={
              <>
                The vulnerability has been actively exploited in the wild. This
                data is reported by the Cybersecurity and Infrustructure
                Security Agency (CISA).
              </>
            }
          >
            Known exploit
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithToolTip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
            {isSandboxMode && <PremiumFeatureIconWithTooltip />}
          </>
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => (
        <TextCell value={value ? "Yes" : "No"} />
      ),
    },
    {
      title: "Published",
      accessor: "cve_published",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                The date this vulnerability was published in the National
                Vulnerability Database (NVD).
              </>
            }
          >
            Published
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithToolTip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
            {isSandboxMode && <PremiumFeatureIconWithTooltip />}
          </>
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => {
        const valString = typeof value === "number" ? value.toString() : value;
        return (
          <TextCell
            value={valString ? { timeString: valString } : undefined}
            formatter={valString ? HumanTimeDiffWithDateTip : undefined}
          />
        );
      },
    },
  ];

  return isPremiumTier ? tableHeaders.concat(premiumHeaders) : tableHeaders;
};

export default generateTableConfig;
