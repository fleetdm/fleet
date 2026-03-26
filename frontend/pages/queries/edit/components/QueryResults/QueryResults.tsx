import React, { useState, useContext, useEffect, useCallback } from "react";
import { CellProps, Row, Column } from "react-table";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import FileSaver from "file-saver";
import { QueryContext } from "context/query";
import { useDebouncedCallback } from "use-debounce";

import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";
import { SUPPORT_LINK } from "utilities/constants";
import {
  ICampaign,
  ICampaignError,
  ICampaignPerformanceStats,
} from "interfaces/campaign";
import { ITarget } from "interfaces/target";
import { PerformanceImpactIndicatorValue } from "interfaces/schedulable_query";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import ShowQueryModal from "components/modals/ShowQueryModal";
import LiveResultsHeading from "components/queries/LiveResults/LiveResultsHeading";
import AwaitingResults from "components/queries/LiveResults/AwaitingResults";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import PATHS from "router/paths";

import generateColumnConfigsFromRows from "./QueryResultsTableConfig";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  isQueryClipped: boolean;
  queryName?: string;
  onRunQuery: () => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  // set during target selection, persisted through each step of the flow
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_TITLE = "New Report";
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
  PERFORMANCE: "Performance",
};

const getPerformanceIndicator = (stats: ICampaignPerformanceStats) => {
  const cpuTotal = stats.user_time + stats.system_time;
  if (cpuTotal < 2000) {
    return PerformanceImpactIndicatorValue.MINIMAL;
  }
  if (cpuTotal < 4000) {
    return PerformanceImpactIndicatorValue.CONSIDERABLE;
  }
  return PerformanceImpactIndicatorValue.EXCESSIVE;
};

const perfColumnConfigs: Column<ICampaignPerformanceStats>[] = [
  {
    id: "host_display_name",
    Header: "Host",
    accessor: "host_display_name",
    Cell: (cellProps: CellProps<ICampaignPerformanceStats>) => {
      const hostID = cellProps.row.original.host_id;
      return (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.HOST_DETAILS(hostID)}
        />
      );
    },
  },
  {
    id: "performance_impact",
    Header: () => (
      <TooltipWrapper tipContent="The average performance impact across all hosts.">
        Performance impact
      </TooltipWrapper>
    ),
    disableSortBy: true,
    accessor: (row) => getPerformanceIndicator(row),
    Cell: (cellProps: CellProps<ICampaignPerformanceStats>) => (
      <PerformanceImpactCell
        value={{
          indicator: cellProps.cell.value,
          id: cellProps.row.original.host_id,
        }}
        isHostSpecific
        customIdPrefix="live-perf"
      />
    ),
  },
];

const QueryResults = ({
  campaign,
  isQueryFinished,
  isQueryClipped,
  queryName,
  onRunQuery,
  onStopQuery,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IQueryResultsProps): JSX.Element => {
  const { lastEditedQueryBody } = useContext(QueryContext);

  const {
    uiHostCounts,
    serverHostCounts,
    queryResults,
    errors,
    performanceStats,
  } = campaign || {};

  const [navTabIndex, setNavTabIndex] = useState(0);
  const [showQueryModal, setShowQueryModal] = useState(false);
  const [filteredResults, setFilteredResults] = useState<Row[]>([]);
  const [filteredErrors, setFilteredErrors] = useState<Row[]>([]);
  const [resultsColumnConfigs, setResultsColumnConfigs] = useState<Column[]>(
    []
  );
  const [errorColumnConfigs, setErrorColumnConfigs] = useState<
    Column<ICampaignError>[]
  >([]);
  const [queryResultsForTableRender, setQueryResultsForTableRender] = useState(
    queryResults
  );

  // immediately reset results
  const onRunAgain = useCallback(() => {
    setQueryResultsForTableRender([]);
    onRunQuery();
  }, [onRunQuery]);

  const debounceQueryResults = useDebouncedCallback(
    setQueryResultsForTableRender,
    1000,
    { maxWait: 2000 }
  );

  // This is throwing an error not to use hook within a useEffect
  useEffect(() => {
    debounceQueryResults(queryResults);
  }, [queryResults, debounceQueryResults]);

  useEffect(() => {
    if (queryResults && queryResults.length > 0) {
      const newResultsColumnConfigs = generateColumnConfigsFromRows(
        queryResults
      );
      // Update tableHeaders if new headers are found
      if (newResultsColumnConfigs !== resultsColumnConfigs) {
        setResultsColumnConfigs(newResultsColumnConfigs);
      }
    }
  }, [queryResults, lastEditedQueryBody]); // Cannot use tableHeaders as it will cause infinite loop with setTableHeaders

  useEffect(() => {
    if (errorColumnConfigs?.length === 0 && !!errors?.length) {
      setErrorColumnConfigs(generateColumnConfigsFromRows(errors));

      if (errorColumnConfigs && errorColumnConfigs.length > 0) {
        const newErrorColumnConfigs = generateColumnConfigsFromRows(errors);

        // Update errorTableHeaders if new headers are found
        if (newErrorColumnConfigs !== resultsColumnConfigs) {
          setErrorColumnConfigs(newErrorColumnConfigs);
        }
      }
    }
  }, [errors]); // Cannot use errorTableHeaders as it will cause infinite loop with setErrorTableHeaders


  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    FileSaver.saveAs(
      generateCSVQueryResults(
        filteredResults,
        generateCSVFilename(`${queryName || CSV_TITLE} - Results`),
        resultsColumnConfigs
      )
    );
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    FileSaver.saveAs(
      generateCSVQueryResults(
        filteredErrors,
        generateCSVFilename(`${queryName || CSV_TITLE} - Errors`),
        errorColumnConfigs
      )
    );
  };

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const onQueryDone = () => {
    setSelectedTargets([]);
    goToQueryEditor();
  };

  const renderNoResults = () => {
    return (
      <p className="no-results-message">
        Your live report returned no results.
        <span>
          Expecting to see results? Check to see if the host
          {`${targetsTotalCount > 1 ? "s" : ""}`} you targeted reported
          &ldquo;Online&rdquo; or check out the &ldquo;Errors&rdquo; table.
        </span>
      </p>
    );
  };

  const renderCount = useCallback(
    (tableType: "errors" | "results" | "performance") => {
      const count =
        tableType === "results"
          ? filteredResults.length
          : filteredErrors.length;

      return <TableCount name={tableType} count={count} />;
    },
    [filteredResults.length, filteredErrors.length]
  );

  const renderTableButtons = (
    tableType: "results" | "errors" | "performance"
  ) => {
    const exportHandlers: Record<string, typeof onExportQueryResults> = {
      results: onExportQueryResults,
      errors: onExportErrorsResults,
    };

    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="inverse"
        >
          <>
            Show query <Icon name="eye" />
          </>
        </Button>
        {exportHandlers[tableType] && (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={exportHandlers[tableType]}
            variant="inverse"
          >
            <>
              Export {tableType}
              <Icon name="download" />
            </>
          </Button>
        )}
      </div>
    );
  };

  const renderTable = (
    tableData: unknown[],
    tableType: "errors" | "results" | "performance"
  ) => {
    const columnConfigsMap = {
      results: resultsColumnConfigs,
      errors: errorColumnConfigs,
      performance: perfColumnConfigs,
    };
    const setExportRowsMap: Record<string, typeof setFilteredResults> = {
      results: setFilteredResults,
      errors: setFilteredErrors,
    };

    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          defaultSortHeader="host_display_name"
          columnConfigs={columnConfigsMap[tableType]}
          data={tableData || []}
          emptyComponent={renderNoResults}
          isLoading={false}
          isClientSidePagination
          isClientSideFilter
          isMultiColumnFilter
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle={tableType}
          customControl={() => renderTableButtons(tableType)}
          setExportRows={setExportRowsMap[tableType]}
          renderCount={() => renderCount(tableType)}
        />
      </div>
    );
  };

  const renderResultsTab = () => {
    // TODO - clean up these conditions
    const hasNoResultsYet =
      !isQueryFinished &&
      (!queryResults?.length || resultsColumnConfigs === null);
    const finishedWithNoResults = isQueryFinished && !queryResults?.length;

    if (hasNoResultsYet) {
      return <AwaitingResults />;
    }

    if (finishedWithNoResults) {
      return renderNoResults();
    }

    return renderTable(queryResultsForTableRender, "results");
  };

  const renderErrorsTab = () => renderTable(errors, "errors");

  const renderPerformanceTab = () =>
    renderTable(performanceStats, "performance");

  const firstTabClass = classnames("react-tabs__tab", "no-count", {
    "errors-empty": !errors || errors?.length === 0,
  });

  return (
    <div className={baseClass}>
      <LiveResultsHeading
        numHostsTargeted={targetsTotalCount}
        numHostsResponded={uiHostCounts.total}
        numHostsRespondedResults={serverHostCounts.countOfHostsWithResults}
        numHostsRespondedNoErrorsAndNoResults={
          serverHostCounts.countOfHostsWithNoResults
        }
        numHostsRespondedErrors={uiHostCounts.failed}
        isFinished={isQueryFinished}
        onClickDone={onQueryDone}
        onClickRunAgain={onRunAgain}
        onClickStop={onStopQuery}
      />
      {isQueryClipped && (
        <InfoBanner
          color="yellow"
          cta={<CustomLink url={SUPPORT_LINK} text="Get help" newTab />}
        >
          <div>
            <b>Results clipped.</b> A sample of this report&apos;s results and
            errors is included below. Please target fewer hosts at once to build
            a full set of results.
          </div>
        </InfoBanner>
      )}
      <TabNav>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              <TabText count={errors?.length} countVariant="alert">
                {NAV_TITLES.ERRORS}
              </TabText>
            </Tab>
            <Tab disabled={!performanceStats?.length}>
              {NAV_TITLES.PERFORMANCE}
            </Tab>
          </TabList>
          <TabPanel>{renderResultsTab()}</TabPanel>
          <TabPanel>{renderErrorsTab()}</TabPanel>
          <TabPanel>{renderPerformanceTab()}</TabPanel>
        </Tabs>
      </TabNav>
      {showQueryModal && (
        <ShowQueryModal
          query={lastEditedQueryBody}
          onCancel={onShowQueryModal}
        />
      )}
    </div>
  );
};

export default QueryResults;
