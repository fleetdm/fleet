import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { IDataTableMdmFormat, IMdmSolution } from "interfaces/macadmins";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import {
  generateSolutionsTableHeaders,
  generateSolutionsDataSet,
} from "./MDMSolutionsTableConfig";
import generateEnrollmentTableHeaders from "./MDMEnrollmentTableConfig";

interface IMdmCardProps {
  errorMacAdmins: Error | null;
  isMacAdminsFetching: boolean;
  formattedMdmData: IDataTableMdmFormat[];
  mdmSolutions: IMdmSolution[] | null;
}

const DEFAULT_SORT_DIRECTION = "desc";
const SOLUTIONS_DEFAULT_SORT_HEADER = "hosts_count";
const ENROLLMENT_DEFAULT_SORT_DIRECTION = "asc";
const ENROLLMENT_DEFAULT_SORT_HEADER = "status";
const PAGE_SIZE = 8;
const baseClass = "home-mdm";

const EmptyMdmEnrollment = (): JSX.Element => (
  <div className={`${baseClass}__empty-mdm`}>
    <h1>Unable to detect MDM enrollment</h1>
    <p>
      To see MDM versions, deploy&nbsp;
      <a
        href="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
        target="_blank"
        rel="noopener noreferrer"
      >
        Fleet&apos;s osquery installer
      </a>
      .
    </p>
  </div>
);

const EmptyMdmSolutions = (): JSX.Element => (
  <div className={`${baseClass}__empty-mdm`}>
    <h1>No MDM solutions detected</h1>
    <p>
      This report is updated every hour to protect the performance of your
      devices.
    </p>
  </div>
);

const Mdm = ({
  isMacAdminsFetching,
  errorMacAdmins,
  formattedMdmData,
  mdmSolutions,
}: IMdmCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState(0);

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  const solutionsTableHeaders = generateSolutionsTableHeaders();
  const enrollmentTableHeaders = generateEnrollmentTableHeaders();
  const solutionsDataSet = generateSolutionsDataSet(mdmSolutions);

  // Renders opaque information as host information is loading
  const opacity = isMacAdminsFetching ? { opacity: 0 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {isMacAdminsFetching && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TabsWrapper>
          <Tabs selectedIndex={navTabIndex} onSelect={onTabChange}>
            <TabList>
              <Tab>Solutions</Tab>
              <Tab>Enrollment</Tab>
            </TabList>
            <TabPanel>
              {errorMacAdmins ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={solutionsTableHeaders}
                  data={solutionsDataSet}
                  isLoading={isMacAdminsFetching}
                  defaultSortHeader={SOLUTIONS_DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"MDM"}
                  emptyComponent={EmptyMdmSolutions}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  isClientSidePagination
                  disableCount
                  disableActionButton
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
            <TabPanel>
              {errorMacAdmins ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={enrollmentTableHeaders}
                  data={formattedMdmData}
                  isLoading={isMacAdminsFetching}
                  defaultSortHeader={ENROLLMENT_DEFAULT_SORT_HEADER}
                  defaultSortDirection={ENROLLMENT_DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"MDM"}
                  emptyComponent={EmptyMdmEnrollment}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  disablePagination
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
          </Tabs>
        </TabsWrapper>
      </div>
    </div>
  );
};

export default Mdm;
