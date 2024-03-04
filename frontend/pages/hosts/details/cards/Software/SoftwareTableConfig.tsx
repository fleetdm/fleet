import React from "react";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";

import { formatDistanceToNow } from "date-fns";

import { ISoftware, SOURCE_TYPE_CONVERSION } from "interfaces/software";
import PATHS from "router/paths";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TooltipWrapper from "components/TooltipWrapper";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { COLORS } from "styles/var/colors";
import { getSoftwareBundleTooltipJSX } from "utilities/helpers";

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
    original: ISoftware;
    index: number;
  };
}

interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface IVulnCellProps extends ICellProps {
  cell: {
    value: string[];
  };
}

interface ILastUsedCellProps extends ICellProps {
  cell: {
    value: string;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: IStringCellProps) => JSX.Element)
    | ((props: IVulnCellProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  disableGlobalFilter?: boolean;
  sortType?: string;
  // Filter can be used by react-table to render a filter input inside the column header
  Filter?: () => null | JSX.Element;
  filter?: string; // one of the enumerated `filterTypes` for react-table
  // (see https://github.com/tannerlinsley/react-table/blob/master/src/filterTypes.js)
  // or one of the custom `filterTypes` defined for the `useTable` instance (see `DataTable`)
}

const formatSoftwareType = (source: string) => {
  const DICT = SOURCE_TYPE_CONVERSION;
  return DICT[source] || "Unknown";
};

const condenseVulnerabilities = (vulns: string[]): string[] => {
  const condensed =
    (vulns?.length && vulns.length === 4
      ? vulns.slice(-4).reverse()
      : vulns.slice(-3).reverse()) || [];
  return vulns?.length > 4
    ? condensed.concat(`+${vulns?.length - 3} more`)
    : condensed;
};

const renderBundleTooltip = (name: string, bundle: string) => (
  <span className="name-container">
    <TooltipWrapper
      position="top-start"
      tipContent={
        <span>
          <b>Bundle identifier: </b>
          <br />${bundle}
        </span>
      }
    >
      {name}
    </TooltipWrapper>
  </span>
);

interface IInstalledPathCellProps {
  cell: {
    value: string[];
  };
  row: {
    original: ISoftware;
  };
}

const condenseInstalledPaths = (installedPaths: string[]): string[] => {
  if (!installedPaths?.length) {
    return [];
  }
  const condensed =
    installedPaths.length === 4
      ? installedPaths.slice(-4).reverse()
      : installedPaths.slice(-3).reverse() || [];
  return installedPaths.length > 4
    ? condensed.concat(`+${installedPaths.length - 3} more`) // TODO: confirm limit
    : condensed;
};

const tooltipTextWithLineBreaks = (lines: string[]) => {
  return lines.map((line) => {
    return (
      <span
        className="tooltip__tooptip_text_line"
        key={Math.random().toString().slice(2)}
      >
        {line}
        <br />
      </span>
    );
  });
};

interface ISoftwareTableData extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[];
}

interface ISoftwareTableHeadersProps {
  deviceUser?: boolean;
  setFilteredSoftwarePath: (path: string) => void;
  router?: InjectedRouter;
  pathname: string;
}

export const generateSoftwareTableData = (
  software: ISoftware[]
): ISoftwareTableData[] => {
  return software.map((s) => {
    return {
      ...s,
      vulnerabilities: s.vulnerabilities?.map((v) => v.cve) || [],
    };
  });
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateSoftwareTableHeaders = ({
  deviceUser = false,
  setFilteredSoftwarePath,
  router,
  pathname,
}: ISoftwareTableHeadersProps): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: IStringCellProps) => {
        const { id, name, bundle_identifier: bundle } = cellProps.row.original;
        if (deviceUser) {
          return bundle ? (
            renderBundleTooltip(name, bundle)
          ) : (
            <span>{name}</span>
          );
        }

        const onClickSoftware = (e: React.MouseEvent) => {
          // Allows for button to be clickable in a clickable row
          e.stopPropagation();
          setFilteredSoftwarePath(pathname);
          router?.push(PATHS.SOFTWARE_VERSION_DETAILS(id.toString()));
        };

        return (
          <LinkCell
            path={PATHS.SOFTWARE_VERSION_DETAILS(id.toString())}
            customOnClick={onClickSoftware}
            value={name}
            tooltipContent={
              bundle ? getSoftwareBundleTooltipJSX(bundle) : undefined
            }
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      disableGlobalFilter: true,
      accessor: "version",
      Cell: (cellProps: IStringCellProps) => {
        return <TextCell value={cellProps.cell.value} />;
      },
    },
    {
      title: "Type",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      disableGlobalFilter: true,
      accessor: "source",
      Cell: (cellProps: IStringCellProps) => (
        <TextCell value={cellProps.cell.value} formatter={formatSoftwareType} />
      ),
    },
    {
      title: "Vulnerabilities",
      Header: "Vulnerabilities",
      accessor: "vulnerabilities",
      disableSortBy: true,
      disableGlobalFilter: false,
      Filter: () => null, // input for this column filter outside of column header
      filter: "hasLength", // filters out rows where vulnerabilities has no length if filter value is `true`
      Cell: (cellProps: IVulnCellProps): JSX.Element => {
        const vulnerabilities = cellProps.cell.value || [];

        const tooltipText = condenseVulnerabilities(vulnerabilities).map(
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
              className={`vulnerabilities ${
                vulnerabilities.length > 1 ? "text-muted tooltip" : ""
              }`}
              data-tip
              data-for={`vulnerabilities__${cellProps.row.original.id}`}
              data-tip-disable={vulnerabilities.length <= 1}
            >
              {vulnerabilities.length === 1
                ? vulnerabilities[0]
                : `${vulnerabilities.length} vulnerabilities`}
            </span>
            <ReactTooltip
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
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
    },
    {
      title: "Last used",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "last_opened_at",
      Cell: (cellProps: ILastUsedCellProps): JSX.Element => {
        const lastUsed = cellProps.cell.value
          ? `${formatDistanceToNow(Date.parse(cellProps.cell.value))} ago`
          : "Unavailable";
        const hasLastUsed = lastUsed !== "Unavailable";
        return (
          <>
            <span
              className={`last-used ${
                lastUsed === "Unavailable" ? "text-muted tooltip" : ""
              }`}
              data-tip
              data-for={`last_used__${cellProps.row.original.id}`}
              data-tip-disable={hasLastUsed}
            >
              {lastUsed}
            </span>
            <ReactTooltip
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={`last_used__${cellProps.row.original.id}`}
              className="last_used_tooltip"
              data-tip-disable={hasLastUsed}
              data-html
            >
              <span className={`last_used tooltip__tooltip-text`}>
                Last used information <br />
                is only available for the <br />
                Application (macOS) <br />
                software type.
              </span>
            </ReactTooltip>
          </>
        );
      },
      sortType: "dateStrings",
    },
    {
      title: "File path",
      Header: () => {
        return (
          <TooltipWrapper
            tipContent={
              <>
                This is where the software is <br />
                located on this host.
              </>
            }
          >
            File path
          </TooltipWrapper>
        );
      },
      disableSortBy: true,
      accessor: "installed_paths",
      Cell: (cellProps: IInstalledPathCellProps): JSX.Element => {
        const numInstalledPaths = cellProps.cell.value?.length || 0;
        const installedPaths = condenseInstalledPaths(
          cellProps.cell.value || []
        );
        if (installedPaths.length) {
          const tooltipText = tooltipTextWithLineBreaks(installedPaths);
          return (
            <>
              <span
                className={`text-cell ${
                  installedPaths.length > 1 ? "text-muted tooltip" : ""
                }`}
                data-tip
                data-for={`installed_paths__${cellProps.row.original.id}`}
                data-tip-disable={installedPaths.length <= 1}
              >
                {numInstalledPaths === 1
                  ? installedPaths[0]
                  : `${numInstalledPaths} paths`}
              </span>
              <ReactTooltip
                effect="solid"
                backgroundColor={COLORS["tooltip-bg"]}
                id={`installed_paths__${cellProps.row.original.id}`}
                className="installed_paths__tooltip"
                data-html
                clickable
                delayHide={300}
              >
                <span className={`tooltip__tooltip-text`}>{tooltipText}</span>
              </ReactTooltip>
            </>
          );
        }
        return <span className="text-muted">{DEFAULT_EMPTY_CELL_VALUE}</span>;
      },
    },
    {
      title: "",
      Header: "",
      disableSortBy: true,
      disableGlobalFilter: true,
      accessor: "linkToFilteredHosts",
      Cell: (cellProps: IStringCellProps) => {
        return (
          <ViewAllHostsLink
            queryParams={{ software_id: cellProps.row.original.id }}
            className="software-link"
          />
        );
      },
      disableHidden: true,
    },
  ];

  // Device user cannot view all hosts software
  if (deviceUser) {
    tableHeaders.pop();
  }

  return tableHeaders;
};

export default { generateSoftwareTableHeaders, generateSoftwareTableData };
