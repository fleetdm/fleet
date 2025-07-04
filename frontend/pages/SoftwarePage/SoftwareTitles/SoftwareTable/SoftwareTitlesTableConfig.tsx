import React from "react";
import { CellProps, Column } from "react-table";
import { InjectedRouter } from "react-router";

import {
  ISoftwareTitle,
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
} from "interfaces/software";
import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";
import { getAutomaticInstallPoliciesCount } from "pages/SoftwarePage/helpers";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";

import VersionCell from "../../components/tables/VersionCell";
import VulnerabilitiesCell from "../../components/tables/VulnerabilitiesCell";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties

type ISoftwareTitlesTableConfig = Column<ISoftwareTitle>;
type ITableStringCellProps = IStringCellProps<ISoftwareTitle>;
type IVersionsCellProps = CellProps<ISoftwareTitle, ISoftwareTitle["versions"]>;
type IVulnerabilitiesCellProps = IVersionsCellProps;
type IHostCountCellProps = CellProps<
  ISoftwareTitle,
  ISoftwareTitle["hosts_count"]
>;
type IViewAllHostsLinkProps = CellProps<ISoftwareTitle>;

type ITableHeaderProps = IHeaderProps<ISoftwareTitle>;

export const getVulnerabilities = <
  T extends { vulnerabilities: string[] | null }
>(
  versions: T[]
) => {
  if (!versions) {
    return [];
  }
  const vulnerabilities = versions.reduce((acc: string[], currentVersion) => {
    if (
      currentVersion.vulnerabilities &&
      currentVersion.vulnerabilities.length !== 0
    ) {
      acc.push(...currentVersion.vulnerabilities);
    }
    return acc;
  }, []);
  return vulnerabilities;
};

/**
 * Gets the data needed to render the software name cell.
 */
const getSoftwareNameCellData = (
  softwareTitle: ISoftwareTitle,
  teamId?: number
) => {
  const softwareTitleDetailsPath = getPathWithQueryParams(
    PATHS.SOFTWARE_TITLE_DETAILS(softwareTitle.id.toString()),
    { team_id: teamId }
  );

  const { software_package, app_store_app } = softwareTitle;
  let hasInstaller = false;
  let isSelfService = false;
  let installType: "manual" | "automatic" | undefined;
  let iconUrl: string | null = null;
  if (software_package) {
    hasInstaller = true;
    isSelfService = software_package.self_service;
    installType =
      software_package.automatic_install_policies &&
      software_package.automatic_install_policies.length > 0
        ? "automatic"
        : "manual";
  } else if (app_store_app) {
    hasInstaller = true;
    isSelfService = app_store_app.self_service;
    iconUrl = app_store_app.icon_url;
    installType =
      app_store_app.automatic_install_policies &&
      app_store_app.automatic_install_policies.length > 0
        ? "automatic"
        : "manual";
  }

  const automaticInstallPoliciesCount = getAutomaticInstallPoliciesCount(
    softwareTitle
  );

  const isAllTeams = teamId === undefined;

  return {
    name: softwareTitle.name,
    source: softwareTitle.source,
    path: softwareTitleDetailsPath,
    hasInstaller: hasInstaller && !isAllTeams,
    isSelfService,
    installType,
    iconUrl,
    automaticInstallPoliciesCount,
  };
};

const generateTableHeaders = (
  router: InjectedRouter,
  teamId?: number
): ISoftwareTitlesTableConfig[] => {
  const softwareTableHeaders: ISoftwareTitlesTableConfig[] = [
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
      ),
      disableSortBy: false,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        const nameCellData = getSoftwareNameCellData(
          cellProps.row.original,
          teamId
        );

        return (
          <SoftwareNameCell
            name={nameCellData.name}
            source={nameCellData.source}
            path={nameCellData.path}
            router={router}
            hasInstaller={nameCellData.hasInstaller}
            isSelfService={nameCellData.isSelfService}
            iconUrl={nameCellData.iconUrl ?? undefined}
            automaticInstallPoliciesCount={
              nameCellData.automaticInstallPoliciesCount
            }
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      Header: "Version",
      disableSortBy: true,
      accessor: "versions",
      Cell: (cellProps: IVersionsCellProps) => (
        <VersionCell versions={cellProps.cell.value} />
      ),
    },
    {
      Header: "Type",
      disableSortBy: true,
      accessor: "source",
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={formatSoftwareType(cellProps.row.original)} />
      ),
    },
    // the "vulnerabilities" accessor is used but the data is actually coming
    // from the version attribute. We do this as we already have a "versions"
    // attribute used for the "Version" column and we cannot reuse. This is a
    // limitation of react-table.
    // With the versions data, we can sum up the vulnerabilities to get the
    // total number of vulnerabilities for the software title
    {
      Header: "Vulnerabilities",
      disableSortBy: true,
      Cell: (cellProps: IVulnerabilitiesCellProps) => {
        const vulnDetectionNotSupported =
          isIpadOrIphoneSoftwareSource(cellProps.row.original.source) ||
          cellProps.row.original.source === "tgz_packages";

        if (vulnDetectionNotSupported) {
          return <TextCell value="Not supported" grey />;
        }
        const vulnerabilities = getVulnerabilities(
          cellProps.row.original.versions ?? []
        );
        return <VulnerabilitiesCell vulnerabilities={vulnerabilities} />;
      },
    },
    {
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Hosts"
          disableSortBy={false}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hosts_count",
      Cell: (cellProps: IHostCountCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      Header: "",
      id: "view-all-hosts",
      disableSortBy: true,
      Cell: (cellProps: IViewAllHostsLinkProps) => {
        const hostCountNotSupported =
          cellProps.row.original.source === "tgz_packages";

        if (hostCountNotSupported) return null;

        return (
          <ViewAllHostsLink
            queryParams={{
              software_title_id: cellProps.row.original.id,
              team_id: teamId,
            }}
            className="software-link"
            rowHover
          />
        );
      },
    },
  ];

  return softwareTableHeaders;
};

export default generateTableHeaders;
