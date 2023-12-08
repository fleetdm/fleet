import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";

import {
  formatSoftwareType,
  ISoftwareVersion,
  ISoftwareVulnerability,
} from "interfaces/software";
import PATHS from "router/paths";
import { formatFloatAsPercentage } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import VulnerabilitiesCell from "../components/VulnerabilitiesCell";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: number | string | ISoftwareVulnerability[];
  };
  row: {
    original: ISoftwareVersion;
  };
}
interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface IVersionCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface INumberCellProps extends ICellProps {
  cell: {
    value: number;
  };
}

interface IVulnCellProps extends ICellProps {
  cell: {
    value: ISoftwareVulnerability[];
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

const condenseVulnerabilities = (
  vulnerabilities: ISoftwareVulnerability[]
): string[] => {
  const condensed =
    (vulnerabilities?.length &&
      vulnerabilities
        .slice(-3)
        .map((v) => v.cve)
        .reverse()) ||
    [];
  return vulnerabilities.length > 3
    ? condensed.concat(`+${vulnerabilities.length - 3} more`)
    : condensed;
};

const getMaxProbability = (vulns: ISoftwareVulnerability[]) =>
  vulns.reduce(
    (max, { epss_probability }) => Math.max(max, epss_probability || 0),
    0
  );

const generateEPSSColumnHeader = (isSandboxMode = false) => {
  return {
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              The probability that this software will be exploited
              <br />
              in the next 30 days (EPSS probability). This data is
              <br />
              reported by FIRST.org.
            </>
          }
        >
          Probability of exploit
        </TooltipWrapper>
      );
      return (
        <>
          {isSandboxMode && <PremiumFeatureIconWithTooltip />}
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={headerProps.column.isSortedDesc}
          />
        </>
      );
    },
    disableSortBy: false,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps): JSX.Element => {
      const vulns = cellProps.cell.value || [];
      const maxProbability = (!!vulns.length && getMaxProbability(vulns)) || 0;
      const displayValue =
        (maxProbability && formatFloatAsPercentage(maxProbability)) ||
        DEFAULT_EMPTY_CELL_VALUE;

      return (
        <span
          className={`vulnerabilities ${!vulns.length ? "text-muted" : ""}`}
        >
          {displayValue}
        </span>
      );
    },
  };
};

const generateVulnColumnHeader = () => {
  return {
    title: "Vulnerabilities",
    Header: "Vulnerabilities",
    disableSortBy: true,
    accessor: "vulnerabilities",
    Cell: (cellProps: IVulnCellProps): JSX.Element => {
      const vulnerabilities = cellProps.cell.value || [];
      const tooltipText = condenseVulnerabilities(vulnerabilities)?.map(
        (value) => {
          return (
            <span key={`vuln_${value}`}>
              {value}
              <br />
            </span>
          );
        }
      );

      if (!vulnerabilities?.length) {
        return <span className="vulnerabilities text-muted">---</span>;
      }
      return (
        <>
          <span
            className={`text-cell vulnerabilities ${
              vulnerabilities.length > 1 ? "text-muted tooltip" : ""
            }`}
            data-tip
            data-for={`vulnerabilities__${cellProps.row.original.id}`}
            data-tip-disable={vulnerabilities.length <= 1}
          >
            {vulnerabilities.length === 1
              ? vulnerabilities[0].cve
              : `${vulnerabilities.length} vulnerabilities`}
          </span>
          <ReactTooltip
            effect="solid"
            backgroundColor="#3e4771"
            id={`vulnerabilities__${cellProps.row.original.id}`}
            data-html
          >
            <span className={`vulnerabilities tooltip__tooltip-text`}>
              {tooltipText}
            </span>
          </ReactTooltip>
        </>
      );
    },
  };
};

const generateTableHeaders = (
  router: InjectedRouter,
  isPremiumTier?: boolean,
  isSandboxMode?: boolean,
  teamId?: number
): Column[] => {
  const softwareTableHeaders = [
    {
      title: "Name",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "name",
      Cell: (cellProps: IStringCellProps): JSX.Element => {
        const { id, name } = cellProps.row.original;

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();
          router?.push(PATHS.SOFTWARE_VERSION_DETAILS(id.toString()));
        };

        return (
          <LinkCell
            path={PATHS.SOFTWARE_VERSION_DETAILS(id.toString())}
            customOnClick={onClickSoftware}
            value={name}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: IVersionCellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Type",
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: IStringCellProps): JSX.Element => (
        <TextCell formatter={formatSoftwareType} value={cellProps.cell.value} />
      ),
    },
    {
      title: "Vulnerabilities",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      // the "vulnerabilities" accessor is used but the data is actually coming
      // from the version attribute. We do this as we already have a "versions"
      // attribute used for the "Version" column and we cannot reuse. This is a
      // limitation of react-table.
      // With the versions data, we can sum up the vulnerabilities to get the
      // total number of vulnerabilities for the software title
      accessor: "vulnerabilities",
      Cell: (cellProps: IVulnCellProps): JSX.Element => (
        <VulnerabilitiesCell vulnerabilities={cellProps.cell.value} />
      ),
    },
    {
      title: "Hosts",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hosts_count",
      Cell: (cellProps: INumberCellProps): JSX.Element => (
        <span className="hosts-cell__wrapper">
          <span className="hosts-cell__count">
            <TextCell value={cellProps.cell.value} />
          </span>
          <span className="hosts-cell__link">
            <ViewAllHostsLink
              queryParams={{
                software_id: cellProps.row.original.id,
                team_id: teamId,
              }}
              className="software-link"
            />
          </span>
        </span>
      ),
    },
  ];

  return softwareTableHeaders;
};

export default generateTableHeaders;
