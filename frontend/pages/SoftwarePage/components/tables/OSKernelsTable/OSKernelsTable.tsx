/**
 software/os/:id > Kernels table (Linux only)
 */

import React from "react";
import classnames from "classnames";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import PATHS from "router/paths";

import { IOperatingSystemKernels } from "interfaces/operating_system";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import generateTableConfig from "./OSKernelsTableConfig";

const baseClass = "os-kernels-table";

const NoKernelsDetected = (): JSX.Element => {
  return (
    <EmptyTable
      header="No kernels detected"
      info={
        <>
          Expecting to see kernels?{" "}
          <CustomLink
            url={GITHUB_NEW_ISSUE_LINK}
            text="File an issue on GitHub"
            newTab
          />
        </>
      }
    />
  );
};

interface ISoftwareVulnerabilitiesTableProps {
  osName: string;
  osVersion: string;
  data: IOperatingSystemKernels[];
  isLoading: boolean;
  className?: string;
  router: InjectedRouter;
  teamIdForApi?: number;
}

interface IRowProps extends Row {
  original: {
    cve?: string;
  };
}

const OSKernelsTable = ({
  osName,
  osVersion,
  data,
  isLoading,
  className,
  router,
  teamIdForApi,
}: ISoftwareVulnerabilitiesTableProps) => {
  const classNames = classnames(baseClass, className);

  const handleRowSelect = (row: IRowProps) => {
    if (row.original.cve) {
      const cveName = row.original.cve.toString();

      const softwareVulnerabilityDetailsPath = getPathWithQueryParams(
        PATHS.SOFTWARE_VULNERABILITY_DETAILS(cveName),
        {
          team_id: teamIdForApi,
        }
      );

      router.push(softwareVulnerabilityDetailsPath);
    }
  };

  const tableHeaders = generateTableConfig({
    teamId: teamIdForApi,
    osName,
    osVersion,
  });

  const rendersOsKernelsVersionCount = () => (
    <TableCount name="items" count={data?.length} />
  );

  return (
    <div className={classNames}>
      <TableContainer
        columnConfigs={tableHeaders}
        data={data}
        defaultSortHeader="hosts_count"
        emptyComponent={NoKernelsDetected}
        isLoading={isLoading}
        isClientSidePagination
        isClientSideFilter
        pageSize={20}
        resultsTitle="items"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        disableTableHeader={data.length === 0}
        renderCount={rendersOsKernelsVersionCount}
      />
    </div>
  );
};

export default OSKernelsTable;
