import React, { useMemo, useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { IMdmStatusCardData, IMdmSolution } from "interfaces/mdm";

import TabsWrapper from "components/TabsWrapper";
import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import {
  generateSolutionsTableHeaders,
  generateSolutionsDataSet,
} from "./MDMSolutionsTableConfig";
import {
  generateStatusTableHeaders,
  generateStatusDataSet,
} from "./MDMStatusTableConfig";

interface IMdmCardProps {
  error: Error | null;
  isFetching: boolean;
  mdmStatusData: IMdmStatusCardData[];
  mdmSolutions: IMdmSolution[] | null;
  selectedPlatformLabelId?: number;
  selectedTeamId?: number;
}

const DEFAULT_SORT_DIRECTION = "desc";
const SOLUTIONS_DEFAULT_SORT_HEADER = "hosts_count";
const STATUS_DEFAULT_SORT_DIRECTION = "asc";
const STATUS_DEFAULT_SORT_HEADER = "status";
const PAGE_SIZE = 8;
const baseClass = "home-mdm";

const EmptyMdmStatus = (): JSX.Element => (
  <EmptyTable
    header="Unable to detect MDM enrollment"
    info={
      <>
        To see MDM versions, deploy&nbsp;
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
          newTab
          text="Fleet's osquery installer"
        />
      </>
    }
  />
);

const EmptyMdmSolutions = (): JSX.Element => (
  <EmptyTable
    header="No MDM solutions detected"
    info="This report is updated every hour to protect the performance of your
      devices."
  />
);

const Mdm = ({
  isFetching,
  error,
  mdmStatusData,
  mdmSolutions,
  selectedPlatformLabelId,
  selectedTeamId,
}: IMdmCardProps): JSX.Element => {
  const [navTabIndex, setNavTabIndex] = useState(0);

  const onTabChange = (index: number) => {
    setNavTabIndex(index);
  };

  const solutionsTableHeaders = useMemo(
    () => generateSolutionsTableHeaders(selectedTeamId),
    [selectedTeamId]
  );
  const statusTableHeaders = useMemo(
    () => generateStatusTableHeaders(selectedTeamId),
    [selectedTeamId]
  );
  const solutionsDataSet = generateSolutionsDataSet(
    mdmSolutions,
    selectedPlatformLabelId
  );
  const statusDataSet = generateStatusDataSet(
    mdmStatusData,
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
              <Tab>Status</Tab>
            </TabList>
            <TabPanel>
              {error ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columnConfigs={solutionsTableHeaders}
                  data={solutionsDataSet}
                  isLoading={isFetching}
                  defaultSortHeader={SOLUTIONS_DEFAULT_SORT_HEADER}
                  defaultSortDirection={DEFAULT_SORT_DIRECTION}
                  resultsTitle="MDM"
                  emptyComponent={EmptyMdmSolutions}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  isClientSidePagination
                  disableCount
                  pageSize={PAGE_SIZE}
                />
              )}
            </TabPanel>
            <TabPanel>
              {error ? (
                <TableDataError card />
              ) : (
                <TableContainer
                  columnConfigs={statusTableHeaders}
                  data={statusDataSet}
                  isLoading={isFetching}
                  defaultSortHeader={STATUS_DEFAULT_SORT_HEADER}
                  defaultSortDirection={STATUS_DEFAULT_SORT_DIRECTION}
                  resultsTitle="MDM"
                  emptyComponent={EmptyMdmStatus}
                  showMarkAllPages={false}
                  isAllPagesSelected={false}
                  disableCount
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
