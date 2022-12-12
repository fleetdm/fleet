import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { IMdmEnrollmentCardData, IMdmSolution } from "interfaces/mdm";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import {
  generateSolutionsTableHeaders,
  generateSolutionsDataSet,
} from "./MDMSolutionsTableConfig";
import {
  generateEnrollmentTableHeaders,
  generateEnrollmentDataSet,
} from "./MDMEnrollmentTableConfig";

interface IMdmCardProps {
  error: Error | null;
  isFetching: boolean;
  mdmEnrollmentData: IMdmEnrollmentCardData[];
  mdmSolutions: IMdmSolution[] | null;
  selectedPlatformLabelId?: number;
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
  isFetching,
  error,
  mdmEnrollmentData,
  mdmSolutions,
  selectedPlatformLabelId,
}: IMdmCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState(0);

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  const solutionsTableHeaders = generateSolutionsTableHeaders();
  const enrollmentTableHeaders = generateEnrollmentTableHeaders();
  const solutionsDataSet = generateSolutionsDataSet(
    mdmSolutions,
    selectedPlatformLabelId
  );
  const enrollmentDataSet = generateEnrollmentDataSet(
    mdmEnrollmentData,
    selectedPlatformLabelId
  );

  // Renders opaque information as host information is loading
  const opacity = isFetching ? { opacity: 0 } : { opacity: 1 };

  return (
    <div className={baseClass}>
      {isFetching && (
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
              {error ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={solutionsTableHeaders}
                  data={solutionsDataSet}
                  isLoading={isFetching}
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
              {error ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={enrollmentTableHeaders}
                  data={enrollmentDataSet}
                  isLoading={isFetching}
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
