import React from "react";

import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { formatSeverity } from "utilities/helpers";
import { buildQueryStringFromParams } from "utilities/url";
import { formatOperatingSystemDisplayName } from "interfaces/operating_system";
import { IVulnerability } from "interfaces/vulnerability";

import ProbabilityOfExploit from "components/ProbabilityOfExploit/ProbabilityOfExploit";
import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

interface ICellProps {
  cell: {
    value: number | string | IVulnerability[];
  };
  row: {
    original: IVulnerability;
  };
}

interface ITextCellProps extends ICellProps {
  cell: {
    value: string | number;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
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

interface IVulnerabilitiesTableConfigOptions {
  includeName?: boolean;
  includeVulnerabilities?: boolean;
  includeIcon?: boolean;
}

const generateTableHeaders = (
  isPremiumTier?: boolean,
  router?: InjectedRouter,
  configOptions?: IVulnerabilitiesTableConfigOptions,
  teamId?: number
): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Vulnerability",
      Header: "Vulnerability",
      disableSortBy: true,
      accessor: "cve",
      Cell: (cellProps: ITextCellProps) => {
        if (!configOptions?.includeIcon) {
          return (
            <TextCell
              value={cellProps.cell.value}
              formatter={(name) => formatOperatingSystemDisplayName(name)}
            />
          );
        }

        const { cve } = cellProps.row.original;

        const teamQueryParam = buildQueryStringFromParams({ team_id: teamId });
        const softwareVulnerabilitiesDetailsPath = `${PATHS.SOFTWARE_VULNERABILITY_DETAILS(
          cve
        )}?${teamQueryParam}`;

        const onClickVulnerability = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();

          router?.push(softwareVulnerabilitiesDetailsPath);
        };

        return (
          <LinkCell
            path={softwareVulnerabilitiesDetailsPath}
            customOnClick={onClickVulnerability}
            value={cve}
          />
        );
      },
    },
    {
      title: "Severity",
      accessor: "cvss_score",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithTooltip = (
          <TooltipWrapper
            tipContent={
              <>
                The worst case impact across different environments (CVSS
                version 3.x base score).
              </>
            }
          >
            Severity
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithTooltip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
          </>
        );
      },
      Cell: ({ cell: { value } }: ITextCellProps): JSX.Element => (
        <TextCell formatter={formatSeverity} value={value} />
      ),
    },
    {
      title: "Probability of exploit",
      accessor: "epss_probability",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithTooltip = (
          <TooltipWrapper
            className="epss_probability"
            tipContent={
              <>
                The probability that this vulnerability will be exploited in the
                next 30 days (EPSS probability). <br />
                This data is reported by FIRST.org.
              </>
            }
            fixedPositionStrategy
          >
            Probability of exploit
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithTooltip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
          </>
        );
      },
      Cell: (cellProps: ICellProps): JSX.Element => (
        <ProbabilityOfExploit
          probabilityOfExploit={cellProps.row.original.epss_probability}
          cisaKnownExploit={cellProps.row.original.cisa_known_exploit}
        />
      ),
    },
    {
      title: "Published",
      accessor: "cve_published",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithTooltip = (
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
              value={titleWithTooltip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
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
    {
      title: "Detected",
      accessor: "created_at",
      disableSortBy: false,
      Header: (headerProps: IHeaderProps): JSX.Element => {
        const titleWithTooltip = (
          <TooltipWrapper
            tipContent={
              <>The date this vulnerability first appeared on a host.</>
            }
          >
            Detected
          </TooltipWrapper>
        );
        return (
          <>
            <HeaderCell
              value={titleWithTooltip}
              isSortedDesc={headerProps.column.isSortedDesc}
            />
          </>
        );
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const createdAt = cellProps.row.original.created_at || "";

        return (
          <TextCell
            value={{ timeString: createdAt }}
            formatter={HumanTimeDiffWithDateTip}
          />
        );
      },
    },
    {
      title: "Hosts",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value="Hosts"
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hosts_count",
      Cell: (cellProps: ITextCellProps): JSX.Element => {
        const { hosts_count } = cellProps.row.original;
        return <TextCell value={hosts_count} />;
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps: ICellProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllHostsLink
                queryParams={{
                  vulnerability: cellProps.row.original.cve,
                  team_id: teamId,
                }}
                className="vulnerabilities-link"
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];

  if (!isPremiumTier) {
    return tableHeaders.filter(
      (header) =>
        header.accessor !== "epss_probability" &&
        header.accessor !== "cve_published" &&
        header.accessor !== "cvss_score"
    );
  }

  return tableHeaders;
};

export default generateTableHeaders;
