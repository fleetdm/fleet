import React, { useState } from "react";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import macadminsAPI from "services/entities/macadmins";
import {
  IMacadminAggregate,
  IDataTableMDMFormat,
  IMDMSolution,
} from "interfaces/macadmins";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import LastUpdatedText from "components/LastUpdatedText";
import generateSolutionsTableHeaders from "./MDMSolutionsTableConfig";
import generateEnrollmentTableHeaders from "./MDMEnrollmentTableConfig";
interface IMDMCardProps {
  showMDMUI: boolean;
  currentTeamId: number | undefined;
  setShowMDMUI: (showMDMTitle: boolean) => void;
  // setActionURL?: (url: string) => void; // software example
  setTitleDetail?: (content: JSX.Element | string | null) => void;
}

const DEFAULT_SORT_DIRECTION = "desc";
const SOLUTIONS_DEFAULT_SORT_HEADER = "hosts_count";
const ENROLLMENT_DEFAULT_SORT_HEADER = "hosts";
const PAGE_SIZE = 8;
const baseClass = "home-mdm";

const EmptyMDMEnrollment = (): JSX.Element => (
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

const EmptyMDMSolutions = (): JSX.Element => (
  <div className={`${baseClass}__empty-mdm`}>
    <h1>No MDM solutions detected</h1>
    <p>
      This report is updated every hour to protect the performance of your
      devices.
    </p>
  </div>
);

const MDM = ({
  showMDMUI,
  currentTeamId,
  setShowMDMUI,
  setTitleDetail,
}: IMDMCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState<number>(0);
  const [formattedMDMData, setFormattedMDMData] = useState<
    IDataTableMDMFormat[]
  >([]);
  const [solutions, setSolutions] = useState<IMDMSolution[]>([]);

  const { isFetching: isMDMFetching, error: errorMDM } = useQuery<
    IMacadminAggregate,
    Error
  >(["MDM", currentTeamId], () => macadminsAPI.loadAll(currentTeamId), {
    keepPreviousData: true,
    onSuccess: (data) => {
      const {
        counts_updated_at,
        mobile_device_management_enrollment_status,
        mobile_device_management_solution,
      } = data.macadmins;
      const {
        enrolled_manual_hosts_count,
        enrolled_automated_hosts_count,
        unenrolled_hosts_count,
      } = mobile_device_management_enrollment_status;

      setShowMDMUI(true);
      setTitleDetail &&
        setTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={counts_updated_at}
            whatToRetrieve={"MDM enrollment"}
          />
        );
      setFormattedMDMData([
        {
          status: "Enrolled (manual)",
          hosts: enrolled_manual_hosts_count,
        },
        {
          status: "Enrolled (automatic)",
          hosts: enrolled_automated_hosts_count,
        },
        { status: "Unenrolled", hosts: unenrolled_hosts_count },
      ]);
      setSolutions(mobile_device_management_solution);
    },
    onError: () => {
      setShowMDMUI(true);
    },
  });

  const onTabChange = (index: number) => {
    // const { MANAGE_SOFTWARE } = paths; // software example
    setNavTabIndex(index);
    // setActionURL &&
    //   setActionURL(
    //     index === 1 ? `${MANAGE_SOFTWARE}?vulnerable=true` : MANAGE_SOFTWARE
    //   );  // software example
  };

  const solutionsTableHeaders = generateSolutionsTableHeaders();
  const enrollmentTableHeaders = generateEnrollmentTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showMDMUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showMDMUI && (
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
              {errorMDM ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={solutionsTableHeaders}
                  data={solutions}
                  isLoading={isMDMFetching}
                  defaultSortHeader={SOLUTIONS_DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"MDM"}
                  emptyComponent={EmptyMDMSolutions}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
                  disableActionButton
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
            <TabPanel>
              {errorMDM ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columns={enrollmentTableHeaders}
                  data={formattedMDMData}
                  isLoading={isMDMFetching}
                  defaultSortHeader={ENROLLMENT_DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  hideActionButton
                  resultsTitle={"MDM"}
                  emptyComponent={EmptyMDMEnrollment}
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

export default MDM;
