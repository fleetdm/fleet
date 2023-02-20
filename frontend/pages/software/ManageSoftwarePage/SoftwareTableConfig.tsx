import React from "react";
import { Column } from "react-table";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";

import { formatSoftwareType, ISoftware } from "interfaces/software";
import { IVulnerability } from "interfaces/vulnerability";
import PATHS from "router/paths";
import { formatFloatAsPercentage } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import Button from "components/buttons/Button";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: number | string | IVulnerability[];
  };
  row: {
    original: ISoftware;
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

interface IVulnCellProps extends ICellProps {
  cell: {
    value: IVulnerability[];
  };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

const condenseVulnerabilities = (
  vulnerabilities: IVulnerability[]
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

const renderBundleTooltip = (name: string, bundle: string) => (
  <span className="name-container">
    <TooltipWrapper
      tipContent={`
        <span>
          <b>Bundle identifier: </b>
          <br />
          ${bundle}
        </span>
      `}
    >
      {name}
    </TooltipWrapper>
  </span>
);

const getMaxProbability = (vulns: IVulnerability[]) =>
  vulns.reduce(
    (max, { epss_probability }) => Math.max(max, epss_probability || 0),
    0
  );

const generateEPSSColumnHeader = () => {
  return {
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
            The probability that this software will be exploited
            <br />
            in the next 30 days (EPSS probability). This data is
            <br />
            reported by FIRST.org.
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
  isPremiumTier?: boolean
): Column[] => {
  const softwareTableHeaders = [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: IStringCellProps): JSX.Element => {
        const { id, name, bundle_identifier: bundle } = cellProps.row.original;

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();

          router?.push(PATHS.SOFTWARE_DETAILS(id.toString()));
        };

        return (
          <Button onClick={onClickSoftware} variant="text-link">
            {bundle ? renderBundleTooltip(name, bundle) : name}
          </Button>
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
      Cell: (cellProps: IStringCellProps): JSX.Element => (
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
    isPremiumTier ? generateEPSSColumnHeader() : generateVulnColumnHeader(),
    {
      title: "Hosts",
      Header: (cellProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={cellProps.column.title}
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
              queryParams={{ software_id: cellProps.row.original.id }}
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
