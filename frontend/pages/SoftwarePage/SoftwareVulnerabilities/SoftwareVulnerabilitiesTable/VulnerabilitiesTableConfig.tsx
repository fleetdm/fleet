import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import formatSeverity from "utilities/helpers";
import { formatOperatingSystemDisplayName } from "interfaces/operating_system";
import { IVulnerability } from "interfaces/vulnerability";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
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

interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface INumberCellProps extends ICellProps {
  cell: {
    value: number;
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

const isSandboxMode = false; // Hardcoded false for now;

const generateTableHeaders = (
  teamId?: number,
  router?: InjectedRouter,
  configOptions?: IVulnerabilitiesTableConfigOptions
): Column[] => [
  {
    Header: "Vulnerability",
    disableSortBy: true,
    accessor: "cve",
    Cell: (cellProps: IStringCellProps) => {
      if (!configOptions?.includeIcon) {
        return (
          <TextCell
            value={cellProps.cell.value}
            formatter={(name) => formatOperatingSystemDisplayName(name)}
          />
        );
      }

      const { cve } = cellProps.row.original;
      const onClickVulnerability = (e: React.MouseEvent) => {
        // Allows for button to be clickable in a clickable row
        e.stopPropagation();

        router?.push(PATHS.SOFTWARE_VULNERABILITY_DETAILS(cve));
      };

      return (
        <LinkCell
          path={PATHS.SOFTWARE_VULNERABILITY_DETAILS(cve)}
          customOnClick={onClickVulnerability}
          value={cve}
        />
      );
    },
  },
  {
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: IStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    Header: (cellProps: IHeaderProps): JSX.Element => (
      <HeaderCell
        value="Hosts"
        disableSortBy={false}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: INumberCellProps): JSX.Element => {
      const { cve, hosts_count } = cellProps.row.original;
      return (
        <span className="hosts-cell__wrapper">
          <span className="hosts-cell__count">
            <TextCell value={hosts_count} />
          </span>
          <span className="hosts-cell__link">
            <ViewAllHostsLink
              queryParams={{
                vulnerability: cve,
                team_id: teamId,
              }}
              className="vulnerability-hosts-link"
            />
          </span>
        </span>
      );
    },
  },
];

const tableHeaders: IDataColumn[] = [
  {
    title: "Vunerability",
    accessor: "cve",
    disableSortBy: true,
    Header: "Vulnerability",
    Cell: (cellProps: ICellProps) => {
      if (cellProps.row.original.cve) {
        const cveId = cellProps.row.original.cve.toString();
        return (
          <LinkCell
            value={cellProps.row.original.cve}
            path={PATHS.SOFTWARE_VULNERABILITY_DETAILS(cveId)}
          />
        );
      }
      return <TextCell value={cellProps.row.original.cve} />;
    },
  },
];

const premiumHeaders: IDataColumn[] = [
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
      // <TextCell formatter={formatSeverity} value={value} />
      // TODO: Fix format severity
      <TextCell value={value} />
    ),
  },
  {
    title: "Probability of exploit",
    accessor: "epss_probability",
    disableSortBy: false,
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          className="epss_probability"
          tipContent={
            <>
              The probability that this vulnerability will be exploited in the
              next 30 days (EPSS probability). <br />
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
    Cell: (cellProps: ICellProps): JSX.Element => (
      // <ProbabilityOfExploitCell
      //   probabilityOfExploit={cellProps.row.original.epss_probability}
      //   cisaKnownExploit={cellProps.row.original.cisa_known_exploit}
      //   rowId={cellProps.row.original.cve}
      // />
      <>Uncomment probability of exploit cell when merged </>
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
  {
    title: "Detected",
    accessor: "created_at",
    disableSortBy: false,
    Header: (headerProps: IHeaderProps): JSX.Element => {
      return (
        <>
          <HeaderCell
            value="Detected"
            isSortedDesc={headerProps.column.isSortedDesc}
          />
          {isSandboxMode && <PremiumFeatureIconWithTooltip />}
        </>
      );
    },
    Cell: (cellProps: ICellProps): JSX.Element => {
      // const createdAt = cellProps.row.original.created_at || "";
      // TODO: Uncomment when fixing created_at on API
      const createdAt = "";

      return (
        <TextCell
          value={{ timeString: createdAt }}
          formatter={HumanTimeDiffWithDateTip}
        />
      );
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
              }}
              className="vulnerabilities-link"
              // rowHover TODO: Uncomment when existing page changes is implemented
            />
          )}
        </>
      );
    },
  },
];

export default generateTableHeaders;
